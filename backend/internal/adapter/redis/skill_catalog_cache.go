package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	domainskill "marketplace-backend/internal/domain/skill"
)

// DefaultSkillCatalogCacheTTL is 10 minutes. The skills catalog is a
// curated list seeded by an admin migration plus user-contributed
// long-tail entries; both surfaces change rarely (curated entries
// only with a release, user contributions amortize per-second).
// 10 minutes balances "fresh enough that a brand-new skill shows up
// quickly" against "cold enough that the catalog DB barely runs".
const DefaultSkillCatalogCacheTTL = 10 * time.Minute

// skillCatalogKeyPrefix namespaces the curated-by-expertise list
// (high cardinality: O(expertise keys * limit values) ≈ 30 entries
// total).
const (
	skillCatalogListKeyPrefix  = "skills:curated:list:"
	skillCatalogCountKeyPrefix = "skills:curated:count:"
)

// SkillCatalogReader is the local interface the cache adapter
// expects from the underlying service. The skill app service
// satisfies it by definition. Defined here (not in port/service)
// to keep the cache adapter's contract co-located with its only
// consumer.
//
// The cache only wraps the two highest-traffic, lowest-volatility
// reads: GetCuratedForExpertise (list) and CountCuratedForExpertise
// (count). Autocomplete is intentionally NOT cached — its key
// space (every prefix the user could type) is too wide to cache
// usefully and the underlying Postgres trigram query is already
// fast.
type SkillCatalogReader interface {
	GetCuratedForExpertise(ctx context.Context, expertiseKey string, limit int) ([]*domainskill.CatalogEntry, error)
	CountCuratedForExpertise(ctx context.Context, expertiseKey string) (int, error)
}

// CachedSkillCatalogReader wraps a SkillCatalogReader with a
// Redis-backed cache. Implements the same interface so callers
// never know the cache is in play.
//
// Cache semantics:
//   - List hit: JSON value wins, no DB read.
//   - List miss: delegate to inner, cache the result keyed by
//     (expertiseKey, limit) — the cache must respect the limit
//     because two callers asking for limit=20 vs limit=50 receive
//     different slices.
//   - Count: cached separately under skills:curated:count:{key}.
//   - Inner error: bubble up, do NOT cache.
//   - Redis blip: log + degrade to inner so a flaky cache never
//     takes down the catalog.
//
// Stampede protection: a singleflight.Group coalesces concurrent
// misses on the same (key, limit) pair into a single DB call.
type CachedSkillCatalogReader struct {
	client *goredis.Client
	inner  SkillCatalogReader
	ttl    time.Duration
	group  singleflight.Group
}

// NewCachedSkillCatalogReader wires the cache decorator. Pass
// DefaultSkillCatalogCacheTTL unless an integration test requires
// a shorter window. Non-positive TTL falls back to the default.
func NewCachedSkillCatalogReader(client *goredis.Client, inner SkillCatalogReader, ttl time.Duration) *CachedSkillCatalogReader {
	if ttl <= 0 {
		ttl = DefaultSkillCatalogCacheTTL
	}
	return &CachedSkillCatalogReader{
		client: client,
		inner:  inner,
		ttl:    ttl,
	}
}

// GetCuratedForExpertise satisfies SkillCatalogReader. The cache
// key includes the limit so two callers asking for different limit
// values receive distinct entries.
func (c *CachedSkillCatalogReader) GetCuratedForExpertise(ctx context.Context, expertiseKey string, limit int) ([]*domainskill.CatalogEntry, error) {
	key := skillCatalogListKeyPrefix + expertiseKey + ":" + strconv.Itoa(limit)

	if entries, ok := c.tryGetList(ctx, key); ok {
		return entries, nil
	}

	v, err, _ := c.group.Do(key, func() (any, error) {
		return c.fillListFromInner(ctx, key, expertiseKey, limit)
	})
	if err != nil {
		return nil, err
	}
	entries, _ := v.([]*domainskill.CatalogEntry)
	return entries, nil
}

func (c *CachedSkillCatalogReader) tryGetList(ctx context.Context, key string) ([]*domainskill.CatalogEntry, bool) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if err == nil {
		var entries []*domainskill.CatalogEntry
		if jerr := json.Unmarshal(raw, &entries); jerr == nil {
			return entries, true
		} else {
			slog.Warn("skill catalog cache: corrupt list, treating as miss",
				"key", key, "error", jerr)
			return nil, false
		}
	}
	if !errors.Is(err, goredis.Nil) {
		slog.Warn("skill catalog cache: redis get list failed, falling back to inner",
			"key", key, "error", err)
	}
	return nil, false
}

func (c *CachedSkillCatalogReader) fillListFromInner(ctx context.Context, key, expertiseKey string, limit int) ([]*domainskill.CatalogEntry, error) {
	entries, err := c.inner.GetCuratedForExpertise(ctx, expertiseKey, limit)
	if err != nil {
		return nil, err
	}
	payload, jerr := json.Marshal(entries)
	if jerr != nil {
		slog.Warn("skill catalog cache: marshal failed, skipping cache write",
			"key", key, "error", jerr)
		return entries, nil
	}
	if sErr := c.client.Set(ctx, key, payload, c.ttl).Err(); sErr != nil {
		slog.Warn("skill catalog cache: redis set list failed",
			"key", key, "error", sErr)
	}
	return entries, nil
}

// CountCuratedForExpertise satisfies SkillCatalogReader. Counts
// are stored as plain decimal strings — no JSON envelope, no
// micro-marshaling cost on the hot path.
func (c *CachedSkillCatalogReader) CountCuratedForExpertise(ctx context.Context, expertiseKey string) (int, error) {
	key := skillCatalogCountKeyPrefix + expertiseKey

	if count, ok := c.tryGetCount(ctx, key); ok {
		return count, nil
	}

	v, err, _ := c.group.Do(key, func() (any, error) {
		return c.fillCountFromInner(ctx, key, expertiseKey)
	})
	if err != nil {
		return 0, err
	}
	count, _ := v.(int)
	return count, nil
}

func (c *CachedSkillCatalogReader) tryGetCount(ctx context.Context, key string) (int, bool) {
	raw, err := c.client.Get(ctx, key).Result()
	if err == nil {
		count, perr := strconv.Atoi(raw)
		if perr == nil {
			return count, true
		}
		slog.Warn("skill catalog cache: corrupt count, treating as miss",
			"key", key, "error", perr)
		return 0, false
	}
	if !errors.Is(err, goredis.Nil) {
		slog.Warn("skill catalog cache: redis get count failed, falling back to inner",
			"key", key, "error", err)
	}
	return 0, false
}

func (c *CachedSkillCatalogReader) fillCountFromInner(ctx context.Context, key, expertiseKey string) (int, error) {
	count, err := c.inner.CountCuratedForExpertise(ctx, expertiseKey)
	if err != nil {
		return 0, err
	}
	if sErr := c.client.Set(ctx, key, strconv.Itoa(count), c.ttl).Err(); sErr != nil {
		slog.Warn("skill catalog cache: redis set count failed",
			"key", key, "error", sErr)
	}
	return count, nil
}

// InvalidateAll flushes every cached skills:curated:* entry. Used
// when a curator-side change (admin seed update) or a user-driven
// CreateUserSkill could meaningfully shift the curated lists.
//
// Implementation note: we do NOT use SCAN or KEYS to iterate the
// namespace — that would introduce O(N) Redis pressure. Instead,
// the caller uses Invalidate(expertiseKey) for targeted flushes
// and accepts a short staleness window for cross-key changes.
//
// For the create-user-skill path we don't need to flush anything
// (user skills are never curated), so InvalidateAll exists mostly
// as a defensive escape hatch. Returns nil on missing keys.
func (c *CachedSkillCatalogReader) InvalidateAll(ctx context.Context) error {
	// FLUSHDB-like behaviour is intentionally NOT exposed — keep
	// the operation scoped to our prefix to avoid clobbering
	// unrelated entries. We use a SCAN loop with COUNT=100 so the
	// command never blocks Redis for long.
	var (
		cursor uint64
		all    []string
	)
	for {
		batch, next, err := c.client.Scan(ctx, cursor, skillCatalogListKeyPrefix+"*", 100).Result()
		if err != nil {
			return err
		}
		all = append(all, batch...)
		batchCount, nextCount, err := c.client.Scan(ctx, cursor, skillCatalogCountKeyPrefix+"*", 100).Result()
		if err != nil {
			return err
		}
		all = append(all, batchCount...)
		cursor = next
		if cursor == 0 && nextCount == 0 {
			break
		}
	}
	if len(all) == 0 {
		return nil
	}
	_, err := c.client.Del(ctx, all...).Result()
	return err
}

// InvalidateExpertise removes all cached entries for a single
// expertise key (every limit variant + the count). Used by
// targeted flushes — the typical caller is the skill replacement
// path which bumps usage_count, potentially reordering the
// curated-by-usage list.
//
// Implementation note: we use SCAN with the precise prefix
// `skills:curated:list:{key}:` so we only ever touch the
// (expertise, limit) tuples relevant to this key. This keeps
// Redis time complexity bounded by the number of distinct limit
// values used by the API (typically 1-2: limit=20 from
// autocomplete, limit=50 from the panel header).
func (c *CachedSkillCatalogReader) InvalidateExpertise(ctx context.Context, expertiseKey string) error {
	listPattern := skillCatalogListKeyPrefix + expertiseKey + ":*"
	keys, err := c.scanAll(ctx, listPattern)
	if err != nil {
		return err
	}
	keys = append(keys, skillCatalogCountKeyPrefix+expertiseKey)
	if len(keys) == 0 {
		return nil
	}
	_, err = c.client.Del(ctx, keys...).Result()
	return err
}

// scanAll walks every key matching `pattern` using SCAN to avoid
// the O(N) blocking behaviour of KEYS. Returns an empty slice
// when no key matches.
func (c *CachedSkillCatalogReader) scanAll(ctx context.Context, pattern string) ([]string, error) {
	var (
		cursor uint64
		out    []string
	)
	for {
		batch, next, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		out = append(out, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return out, nil
}
