package searchindex

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/pendingevent"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/search"
)

// publisher.go exposes the small helper that other feature services
// call to schedule a `search.reindex` or `search.delete` event on
// the pending_events outbox. It is the ONLY public surface the
// search engine asks non-search features to import, and the
// interface is deliberately narrow:
//
//   - PublishReindex(ctx, orgID, persona)
//   - PublishDelete(ctx, orgID)
//
// Feature services take this helper as an OPTIONAL dependency in
// their constructor (nil = no publishing). The service wiring in
// cmd/api/main.go injects a live publisher when Typesense is
// configured, and nil otherwise. Removing the search engine means
// wiring nil — no other change required.
//
// Debouncing:
// Some signals (messages sent, login, cache refresh) can publish
// many events per minute for the same actor. To avoid reindex
// storms the publisher holds a short cooldown keyed on
// `search:last_publish:{orgID}` in a tiny in-memory map. Phase 2
// will promote this to a Redis-backed key so it is shared across
// processes; phase 1 keeps it in-process because the outbox is
// idempotent and a duplicate event is harmless (just slightly
// wasteful).

// Publisher is the thin wrapper around the pending_events repo
// that emits search.* events. Callers depend on the *Publisher
// concrete type (or a matching interface of their choice) — we
// do not define a global interface here to keep the API minimal
// and to avoid forcing every consumer into a wider contract.
type Publisher struct {
	events   repository.PendingEventRepository
	cooldown time.Duration

	// lastPublish is an in-process debounce map keyed by
	// org-id+persona so PublishReindexAllPersonas can emit one
	// event per persona without the cooldown swallowing two of
	// them. Access is serialised via a mutex; the map is bounded
	// by the number of distinct {org, persona} pairs that have
	// published during the cooldown window — < 3k even at peak
	// load.
	mu          sync.Mutex
	lastPublish map[debounceKey]time.Time
}

// debounceKey is the composite cooldown key. Defined as a typed
// struct (not a string) so the map allocation stays cheap and the
// equality semantics are explicit.
type debounceKey struct {
	OrgID   uuid.UUID
	Persona search.Persona
}

// PublisherConfig groups constructor options for the publisher.
type PublisherConfig struct {
	// Events is the pending_events repository to schedule into.
	// Required.
	Events repository.PendingEventRepository

	// Cooldown is the minimum duration between two reindex
	// events for the same org. When Cooldown is zero, the
	// default of 5 minutes applies. Deletes are never
	// debounced — they must propagate immediately for RGPD.
	Cooldown time.Duration
}

// DefaultReindexCooldown is the default window during which the
// publisher deduplicates back-to-back reindex requests for the
// same org.
const DefaultReindexCooldown = 5 * time.Minute

// NewPublisher builds a publisher from its config. Returns an
// error if the required repository is missing.
func NewPublisher(cfg PublisherConfig) (*Publisher, error) {
	if cfg.Events == nil {
		return nil, fmt.Errorf("search publisher: pending events repository is required")
	}
	cooldown := cfg.Cooldown
	if cooldown <= 0 {
		cooldown = DefaultReindexCooldown
	}
	return &Publisher{
		events:      cfg.Events,
		cooldown:    cooldown,
		lastPublish: make(map[debounceKey]time.Time),
	}, nil
}

// PublishReindex schedules a search.reindex event on the outbox via
// the repository's own short-lived transaction.
//
// Use this only when the caller does NOT need the event to share an
// atomic boundary with the domain mutation that triggered it. For
// any flow where Postgres-Typesense drift is unacceptable
// (profile / availability / expertise updates, see BUG-05) prefer
// PublishReindexTx and run the domain UPDATE + the event INSERT
// inside the same caller-owned transaction.
//
// Debounced: if the same {orgID, persona} pair was published within
// the cooldown window, the call is a silent no-op.
func (p *Publisher) PublishReindex(ctx context.Context, orgID uuid.UUID, persona search.Persona) error {
	if p == nil {
		// Nil-safe: a feature that receives a nil publisher
		// silently skips the publish. Keeps the call sites
		// clean of optional-chaining.
		return nil
	}
	event, ok, err := p.buildReindexEvent(orgID, persona)
	if err != nil {
		return err
	}
	if !ok {
		return nil // suppressed by cooldown
	}
	if err := p.events.Schedule(ctx, event); err != nil {
		return fmt.Errorf("search publisher: schedule reindex: %w", err)
	}
	p.recordPublish(debounceKey{OrgID: orgID, Persona: persona})
	return nil
}

// PublishReindexTx schedules a search.reindex event inside the
// caller's transaction. The pending_events row commits or rolls
// back together with whatever else the caller writes on `tx`.
//
// This is the outbox path: combined with the worker that drains
// pending_events on a forever-retrying loop, it guarantees that
// once a profile UPDATE commits the search index will eventually
// catch up — even if Typesense, the worker, or the publisher
// itself were unavailable when the mutation landed.
//
// Debounced exactly like PublishReindex. The cooldown is updated
// only when the row is successfully inserted; a tx that later
// rolls back leaves the cooldown stamped — this is acceptable
// because the cooldown only suppresses redundant work, not
// correctness, and the next mutation past the cooldown window
// will re-trigger the indexing.
func (p *Publisher) PublishReindexTx(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, persona search.Persona) error {
	if p == nil {
		return nil
	}
	if tx == nil {
		return fmt.Errorf("search publisher: tx is required for transactional publish")
	}
	event, ok, err := p.buildReindexEvent(orgID, persona)
	if err != nil {
		return err
	}
	if !ok {
		return nil // suppressed by cooldown
	}
	if err := p.events.ScheduleTx(ctx, tx, event); err != nil {
		return fmt.Errorf("search publisher: schedule reindex tx: %w", err)
	}
	p.recordPublish(debounceKey{OrgID: orgID, Persona: persona})
	return nil
}

// buildReindexEvent validates the input and produces a pending
// event ready to insert. ok=false signals the cooldown swallowed
// the call and the caller should return nil.
func (p *Publisher) buildReindexEvent(orgID uuid.UUID, persona search.Persona) (*pendingevent.PendingEvent, bool, error) {
	if orgID == uuid.Nil {
		return nil, false, fmt.Errorf("search publisher: orgID is required")
	}
	if !persona.IsValid() {
		return nil, false, fmt.Errorf("search publisher: invalid persona %q", persona)
	}

	key := debounceKey{OrgID: orgID, Persona: persona}
	if p.isWithinCooldown(key) {
		return nil, false, nil
	}

	payload, err := json.Marshal(ReindexPayload{
		OrganizationID: orgID,
		Persona:        persona,
	})
	if err != nil {
		return nil, false, fmt.Errorf("search publisher: marshal reindex payload: %w", err)
	}
	event, err := pendingevent.NewPendingEvent(pendingevent.NewPendingEventInput{
		EventType: pendingevent.TypeSearchReindex,
		Payload:   payload,
		FiresAt:   time.Now(),
	})
	if err != nil {
		return nil, false, fmt.Errorf("search publisher: build pending event: %w", err)
	}
	return event, true, nil
}

// PublishReindexAllPersonas fires a reindex event for every persona
// the given org could have. Used by mutation handlers that touch
// persona-agnostic signals (skills, social links, shared profile
// photo / location / languages) without knowing which persona the
// org currently exposes.
//
// The 5-minute debounce in PublishReindex means the practical cost
// is at most one event per persona per cooldown window — the
// downstream worker handles the "this persona has no profile row"
// case by indexing the document as not-published.
//
// We deliberately accept the duplication (3 events vs 1) instead of
// dragging org-type knowledge into every handler that touches a
// shared signal. Removing the search engine still drops to a no-op
// because *Publisher is the receiver.
func (p *Publisher) PublishReindexAllPersonas(ctx context.Context, orgID uuid.UUID) error {
	if p == nil {
		return nil
	}
	for _, persona := range []search.Persona{
		search.PersonaFreelance,
		search.PersonaAgency,
		search.PersonaReferrer,
	} {
		if err := p.PublishReindex(ctx, orgID, persona); err != nil {
			return fmt.Errorf("publish reindex all personas: %w", err)
		}
	}
	return nil
}

// PublishDelete schedules a search.delete event using the
// repository's short-lived transaction. Never debounced — user
// deletions must propagate to the index as fast as possible to
// satisfy the RGPD right to erasure.
//
// For tx-aware callers (account deletion that wraps several writes
// in one transaction) prefer PublishDeleteTx so the index removal
// commits together with the user-data wipe.
func (p *Publisher) PublishDelete(ctx context.Context, orgID uuid.UUID) error {
	if p == nil {
		return nil
	}
	event, err := p.buildDeleteEvent(orgID)
	if err != nil {
		return err
	}
	if err := p.events.Schedule(ctx, event); err != nil {
		return fmt.Errorf("search publisher: schedule delete: %w", err)
	}
	return nil
}

// PublishDeleteTx schedules a search.delete event inside the
// caller's transaction. Mirrors PublishReindexTx for the delete
// path: a rollback wipes the event row alongside whatever else the
// caller had already touched, so the index can never end up
// referring to a wiped organization.
func (p *Publisher) PublishDeleteTx(ctx context.Context, tx *sql.Tx, orgID uuid.UUID) error {
	if p == nil {
		return nil
	}
	if tx == nil {
		return fmt.Errorf("search publisher: tx is required for transactional publish")
	}
	event, err := p.buildDeleteEvent(orgID)
	if err != nil {
		return err
	}
	if err := p.events.ScheduleTx(ctx, tx, event); err != nil {
		return fmt.Errorf("search publisher: schedule delete tx: %w", err)
	}
	return nil
}

// buildDeleteEvent assembles a pending search.delete event.
func (p *Publisher) buildDeleteEvent(orgID uuid.UUID) (*pendingevent.PendingEvent, error) {
	if orgID == uuid.Nil {
		return nil, fmt.Errorf("search publisher: orgID is required")
	}
	payload, err := json.Marshal(DeletePayload{OrganizationID: orgID})
	if err != nil {
		return nil, fmt.Errorf("search publisher: marshal delete payload: %w", err)
	}
	event, err := pendingevent.NewPendingEvent(pendingevent.NewPendingEventInput{
		EventType: pendingevent.TypeSearchDelete,
		Payload:   payload,
		FiresAt:   time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("search publisher: build pending event: %w", err)
	}
	return event, nil
}

// isWithinCooldown reports whether a reindex event was published
// for this {org, persona} pair within the cooldown window.
// Thread-safe.
func (p *Publisher) isWithinCooldown(key debounceKey) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	last, ok := p.lastPublish[key]
	if !ok {
		return false
	}
	return time.Since(last) < p.cooldown
}

// recordPublish stamps the current time against the {org, persona}
// pair so the next call within the cooldown window is a no-op.
// Also opportunistically evicts stale entries so the map does not
// grow unbounded.
func (p *Publisher) recordPublish(key debounceKey) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	p.lastPublish[key] = now

	// Cheap housekeeping: if the map has grown beyond 10k
	// entries, drop everything older than twice the cooldown.
	// 10k was chosen as a safe upper bound for local dev — prod
	// will flip to Redis in phase 2.
	if len(p.lastPublish) > 10_000 {
		threshold := now.Add(-2 * p.cooldown)
		for k, v := range p.lastPublish {
			if v.Before(threshold) {
				delete(p.lastPublish, k)
			}
		}
	}
}
