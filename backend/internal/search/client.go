package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"marketplace-backend/internal/observability"
)

// client.go is a thin HTTP wrapper around the Typesense REST API.
// We deliberately avoid the typesense-go SDK to keep the dependency
// graph minimal and because the subset of endpoints we need is small
// enough that hand-rolling them is cleaner than pulling in another
// package just for type-marshalling.
//
// The public surface is intentionally narrow — Ping, collection
// lifecycle, document CRUD, and search. Anything outside that list
// is either out of scope for phase 1 (synonyms, snapshots) or should
// be added in a future phase with a deliberate design review.

// defaultRequestTimeout caps any single Typesense request. Matches
// the 5-second DB query timeout used elsewhere in the backend so
// p95 latency budgets stay consistent across the stack.
const defaultRequestTimeout = 5 * time.Second

// bulkUpsertBatchSize is the number of documents sent per
// `POST /collections/:name/documents/import?action=upsert` call.
// Typesense accepts larger batches but 100 has the best latency /
// memory trade-off on a standard Typesense 28.0 node.
const bulkUpsertBatchSize = 100

// ErrNotFound is returned when the Typesense API responds with a
// 404. Callers use errors.Is for idempotent delete flows that want
// to swallow a "doc already deleted" response.
var ErrNotFound = errors.New("typesense: resource not found")

// ErrUnauthorized is returned when the master API key is missing or
// wrong. Distinct sentinel so the health check can surface a clear
// message to the operator.
var ErrUnauthorized = errors.New("typesense: unauthorized")

// Client is the thin HTTP wrapper. It is safe for concurrent use.
type Client struct {
	baseURL       string
	apiKey        string
	searchAPIKey  string // bootstrapped via EnsureSearchAPIKey, used as HMAC parent for scoped keys
	httpClient    *http.Client
}

// searchKeyDescription is the identifier used to find + recreate our
// dedicated search-only parent key on each backend startup. Typesense
// will not return a key's value after creation, so we cycle the key
// on every boot: delete any stale copies matching this description,
// then POST a fresh one and cache its value in memory.
const searchKeyDescription = "marketplace-search-parent-v1"

// Option mutates a Client during construction. Exposed as a
// functional-options pattern so future configurability (retries,
// tracing hooks, custom TLS) can land without touching the
// constructor signature.
type Option func(*Client)

// WithHTTPClient overrides the default *http.Client. Used by tests
// to inject a transport that speaks to an httptest.Server.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// NewClient builds a Typesense client pointed at the given host.
// Host must include the scheme (`http://localhost:8108`) — no
// implicit defaults, so configuration errors surface early.
func NewClient(host, apiKey string, opts ...Option) (*Client, error) {
	if host == "" {
		return nil, fmt.Errorf("typesense: host is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("typesense: api key is required")
	}
	u, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("typesense: invalid host %q: %w", host, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("typesense: host must include scheme and hostname, got %q", host)
	}

	c := &Client{
		baseURL: strings.TrimRight(host, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout:   defaultRequestTimeout,
			Transport: observability.HTTPClientTransport(http.DefaultTransport, "typesense"),
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// SearchAPIKey returns the bootstrapped search-only parent key used
// as the HMAC parent for scoped search key generation. Returns empty
// string if EnsureSearchAPIKey has not been called yet.
func (c *Client) SearchAPIKey() string {
	return c.searchAPIKey
}

// EnsureSearchAPIKey bootstraps a dedicated "Search API Key" on
// Typesense (via POST /keys with actions: ["documents:search"]) and
// caches its value on the Client for later use as the HMAC parent of
// scoped search keys. Typesense refuses to derive a scoped key from
// an admin/master key — scoped keys MUST be derived from a key whose
// actions list contains documents:search and whose collections list
// includes the target collection.
//
// Called once at backend startup. The method is idempotent across
// restarts: it lists existing keys, deletes any previous
// bootstrapped key matching our sentinel description, then POSTs a
// fresh one. We have to cycle because Typesense only returns the
// key value on creation — a subsequent GET /keys only exposes the
// first 4 characters.
func (c *Client) EnsureSearchAPIKey(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	// 1. List existing keys, find any with our sentinel description.
	var listResp struct {
		Keys []struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
		} `json:"keys"`
	}
	if err := c.do(ctx, http.MethodGet, "/keys", nil, &listResp); err != nil {
		return fmt.Errorf("typesense list keys: %w", err)
	}

	// 2. Delete every stale copy. We cycle on every startup because
	//    Typesense does not expose the full value after creation.
	for _, k := range listResp.Keys {
		if k.Description != searchKeyDescription {
			continue
		}
		path := fmt.Sprintf("/keys/%d", k.ID)
		if err := c.do(ctx, http.MethodDelete, path, nil, nil); err != nil {
			return fmt.Errorf("typesense delete stale search key %d: %w", k.ID, err)
		}
	}

	// 3. Create a fresh search-only key. The collections list is `["*"]`
	//    so the same key parents scoped keys for every persona — the
	//    scoped key's embedded filter_by locks the persona at query time.
	createBody, err := json.Marshal(map[string]any{
		"description": searchKeyDescription,
		"actions":     []string{"documents:search"},
		"collections": []string{"*"},
	})
	if err != nil {
		return fmt.Errorf("typesense create search key: marshal: %w", err)
	}
	var createResp struct {
		Value string `json:"value"`
	}
	if err := c.do(ctx, http.MethodPost, "/keys", bytes.NewReader(createBody), &createResp); err != nil {
		return fmt.Errorf("typesense create search key: %w", err)
	}
	if createResp.Value == "" {
		return fmt.Errorf("typesense create search key: empty value in response")
	}

	c.searchAPIKey = createResp.Value
	return nil
}

// Ping calls GET /health and returns nil on a 200 response. Used by
// the /ready health endpoint (required path since phase 4).
func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("typesense ping: build request: %w", err)
	}
	// /health intentionally does NOT require authentication, so
	// omit the api key header. Passing it would still work but it
	// reveals less about our key material to misconfigured proxies.

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("typesense ping: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("typesense ping: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// CreateCollection posts a schema to POST /collections. Returns nil
// on 201 Created. If the collection already exists (409 Conflict),
// the error is wrapped so EnsureSchema can tell the two cases apart.
func (c *Client) CreateCollection(ctx context.Context, schema CollectionSchema) error {
	body, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("typesense create collection: marshal: %w", err)
	}
	return c.do(ctx, http.MethodPost, "/collections", bytes.NewReader(body), nil)
}

// GetCollection fetches a single collection by name. Returns
// ErrNotFound when the collection does not exist so callers can
// branch on errors.Is without parsing the error message.
func (c *Client) GetCollection(ctx context.Context, name string) (*CollectionSchema, error) {
	var out CollectionSchema
	if err := c.do(ctx, http.MethodGet, "/collections/"+url.PathEscape(name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddFields applies a purely-additive schema update to an existing
// collection via `PATCH /collections/:name`. Typesense 28.0
// supports adding fields in-place without reindexing; the fields
// return a zero/default value until each document is re-upserted.
//
// Used by EnsureSchema when it detects additive drift (the live
// schema is a strict subset of the expected schema — no renames,
// no type changes). Any non-additive drift still requires the
// manual `_vN` alias-swap flow because PATCH cannot drop or retype
// a field.
func (c *Client) AddFields(ctx context.Context, name string, fields []SchemaField) error {
	if len(fields) == 0 {
		return nil
	}
	payload := struct {
		Fields []SchemaField `json:"fields"`
	}{Fields: fields}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("typesense add fields: marshal: %w", err)
	}
	return c.doRaw(ctx, http.MethodPatch,
		"/collections/"+url.PathEscape(name),
		bytes.NewReader(body),
		"application/json", nil)
}

// aliasPayload is the wire format for POST /aliases/:alias.
type aliasPayload struct {
	CollectionName string `json:"collection_name"`
}

// UpsertAlias creates the alias if it does not exist, or atomically
// swaps its target collection if it does. Used by EnsureSchema on
// first boot AND by the future zero-downtime schema migration flow.
func (c *Client) UpsertAlias(ctx context.Context, alias, targetCollection string) error {
	body, err := json.Marshal(aliasPayload{CollectionName: targetCollection})
	if err != nil {
		return fmt.Errorf("typesense upsert alias: marshal: %w", err)
	}
	return c.do(ctx, http.MethodPut, "/aliases/"+url.PathEscape(alias), bytes.NewReader(body), nil)
}

// GetAlias returns the current target of an alias. ErrNotFound if
// the alias has never been created.
func (c *Client) GetAlias(ctx context.Context, alias string) (string, error) {
	var out aliasPayload
	if err := c.do(ctx, http.MethodGet, "/aliases/"+url.PathEscape(alias), nil, &out); err != nil {
		return "", err
	}
	return out.CollectionName, nil
}

// UpsertDocument indexes (or overwrites) a single document. Used by
// the outbox worker when a single actor changes. For initial bulk
// indexing, use BulkUpsert instead — it's ~100x faster.
func (c *Client) UpsertDocument(ctx context.Context, collection string, doc *SearchDocument) error {
	if err := doc.Validate(); err != nil {
		return fmt.Errorf("typesense upsert: %w", err)
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("typesense upsert: marshal: %w", err)
	}
	path := fmt.Sprintf("/collections/%s/documents?action=upsert", url.PathEscape(collection))
	return c.do(ctx, http.MethodPost, path, bytes.NewReader(body), nil)
}

// DeleteDocument removes a document by id. A 404 is treated as
// success (idempotent) because the caller's intent — "this actor
// must not be in the index anymore" — is already satisfied.
func (c *Client) DeleteDocument(ctx context.Context, collection, docID string) error {
	path := fmt.Sprintf("/collections/%s/documents/%s",
		url.PathEscape(collection), url.PathEscape(docID))
	err := c.do(ctx, http.MethodDelete, path, nil, nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

// DeleteDocumentsByFilter removes every document matching the given
// Typesense filter_by expression. Used by the search.delete outbox
// handler so a single event can wipe all persona variants of an
// organisation (composite IDs make per-ID deletes insufficient).
// Returns the number of documents removed; a zero count + no error
// means the filter matched nothing, which is treated as idempotent
// success upstream.
func (c *Client) DeleteDocumentsByFilter(ctx context.Context, collection, filterBy string) (int, error) {
	if filterBy == "" {
		return 0, fmt.Errorf("delete by filter: filter_by is required")
	}
	path := fmt.Sprintf("/collections/%s/documents?filter_by=%s",
		url.PathEscape(collection), url.QueryEscape(filterBy))
	var out struct {
		NumDeleted int `json:"num_deleted"`
	}
	if err := c.do(ctx, http.MethodDelete, path, nil, &out); err != nil {
		if errors.Is(err, ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return out.NumDeleted, nil
}

// BulkUpsert indexes a slice of documents in batches of 100 using
// the JSONL `/documents/import?action=upsert` endpoint. Returns the
// first error encountered so the CLI can surface it without hiding
// partial progress — the successfully-indexed batches stay in the
// collection, which is the behaviour we want for an idempotent
// bulk reindex.
func (c *Client) BulkUpsert(ctx context.Context, collection string, docs []*SearchDocument) error {
	if len(docs) == 0 {
		return nil
	}
	for start := 0; start < len(docs); start += bulkUpsertBatchSize {
		end := start + bulkUpsertBatchSize
		if end > len(docs) {
			end = len(docs)
		}
		if err := c.bulkUpsertBatch(ctx, collection, docs[start:end]); err != nil {
			return fmt.Errorf("bulk upsert batch [%d:%d]: %w", start, end, err)
		}
	}
	return nil
}

// bulkUpsertBatch encodes a slice of documents as JSONL and POSTs
// them to the import endpoint. Extracted from BulkUpsert so the
// outer loop stays readable and the batch-level retry logic has a
// clean hook point in future phases.
func (c *Client) bulkUpsertBatch(ctx context.Context, collection string, batch []*SearchDocument) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, doc := range batch {
		if err := doc.Validate(); err != nil {
			return fmt.Errorf("validate document %s: %w", doc.ID, err)
		}
		if err := enc.Encode(doc); err != nil {
			return fmt.Errorf("encode document %s: %w", doc.ID, err)
		}
	}
	path := fmt.Sprintf("/collections/%s/documents/import?action=upsert",
		url.PathEscape(collection))
	return c.doRaw(ctx, http.MethodPost, path, &buf, "application/x-ndjson", nil)
}

// SearchParams is the query-side struct posted to /collections/:name/documents/search.
// Exposed for phase 2 which implements the query path; phase 3 adds
// VectorQuery for hybrid semantic search.
type SearchParams struct {
	Q                  string `json:"q"`
	QueryBy            string `json:"query_by"`
	FilterBy           string `json:"filter_by,omitempty"`
	FacetBy            string `json:"facet_by,omitempty"`
	SortBy             string `json:"sort_by,omitempty"`
	Page               int    `json:"page,omitempty"`
	PerPage            int    `json:"per_page,omitempty"`
	IncludeFields      string `json:"include_fields,omitempty"`
	ExcludeFields      string `json:"exclude_fields,omitempty"`
	HighlightFields    string `json:"highlight_fields,omitempty"`
	HighlightFullFields string `json:"highlight_full_fields,omitempty"`
	NumTypos           string `json:"num_typos,omitempty"`
	MaxFacetValues     int    `json:"max_facet_values,omitempty"`

	// VectorQuery activates Typesense hybrid search. Format:
	//   embedding:([0.12,0.34,...], k:20)
	// Only set when the caller has a text query to embed — the
	// listing-page fallback (q=*) should leave it empty so the vector
	// distance does not dominate the ranking of millions of profiles.
	VectorQuery string `json:"vector_query,omitempty"`
}

// Query calls the collection's /documents/search endpoint and
// returns the raw JSON response. Phase 1 only uses this in tests —
// phase 2 will build a typed wrapper on top.
func (c *Client) Query(ctx context.Context, collection string, params SearchParams) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("q", params.Q)
	q.Set("query_by", params.QueryBy)
	if params.FilterBy != "" {
		q.Set("filter_by", params.FilterBy)
	}
	if params.FacetBy != "" {
		q.Set("facet_by", params.FacetBy)
	}
	if params.SortBy != "" {
		q.Set("sort_by", params.SortBy)
	}
	if params.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", params.Page))
	}
	if params.PerPage > 0 {
		q.Set("per_page", fmt.Sprintf("%d", params.PerPage))
	}
	if params.IncludeFields != "" {
		q.Set("include_fields", params.IncludeFields)
	}
	if params.ExcludeFields != "" {
		q.Set("exclude_fields", params.ExcludeFields)
	}
	if params.HighlightFields != "" {
		q.Set("highlight_fields", params.HighlightFields)
	}
	if params.HighlightFullFields != "" {
		q.Set("highlight_full_fields", params.HighlightFullFields)
	}
	if params.NumTypos != "" {
		q.Set("num_typos", params.NumTypos)
	}
	if params.MaxFacetValues > 0 {
		q.Set("max_facet_values", fmt.Sprintf("%d", params.MaxFacetValues))
	}
	if params.VectorQuery != "" {
		q.Set("vector_query", params.VectorQuery)
	}

	encoded := q.Encode()
	// Typesense caps GET query strings at 4000 chars. Hybrid queries
	// with a 1536-dim embedding encode to ~10k chars, well past the
	// cap. When we cross the safe threshold we fall back to the
	// `/multi_search` POST endpoint, which accepts the same params in
	// a JSON body. We use a conservative 3500 char trigger so we stay
	// inside the 4000 limit even with a 500-char URL prefix.
	const maxGetQueryStringLen = 3500
	if len(encoded) <= maxGetQueryStringLen {
		path := fmt.Sprintf("/collections/%s/documents/search?%s",
			url.PathEscape(collection), encoded)
		var raw json.RawMessage
		if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
			return nil, err
		}
		return raw, nil
	}
	return c.queryViaMultiSearch(ctx, collection, q)
}

// queryViaMultiSearch posts the same search params to the
// /multi_search endpoint. Typesense wraps single-search results
// under `results[0]`, so we unwrap back to the expected shape
// before returning. Used only when the GET URL would exceed the
// 4000-char cap (hybrid queries with full embeddings).
func (c *Client) queryViaMultiSearch(ctx context.Context, collection string, q url.Values) (json.RawMessage, error) {
	body := map[string]any{"searches": []map[string]any{buildMultiSearchEntry(collection, q)}}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("typesense multi_search: marshal: %w", err)
	}
	var wrapper struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := c.do(ctx, http.MethodPost, "/multi_search", bytes.NewReader(payload), &wrapper); err != nil {
		return nil, err
	}
	if len(wrapper.Results) == 0 {
		return nil, fmt.Errorf("typesense multi_search: empty results array")
	}
	return wrapper.Results[0], nil
}

// buildMultiSearchEntry converts the flat url.Values from Query into
// the JSON object shape /multi_search expects. Keys are the same,
// values stay strings — Typesense parses them server-side.
func buildMultiSearchEntry(collection string, q url.Values) map[string]any {
	entry := map[string]any{"collection": collection}
	for key, vals := range q {
		if len(vals) == 0 {
			continue
		}
		entry[key] = vals[0]
	}
	return entry
}

// do is the JSON-in-JSON-out helper. body is nil for GET/DELETE.
// out is nil when the caller does not care about the response body.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader, out any) error {
	return c.doRaw(ctx, method, path, body, "application/json", out)
}

// doRaw is like do but lets the caller override the Content-Type,
// needed for NDJSON bulk uploads.
func (c *Client) doRaw(ctx context.Context, method, path string, body io.Reader, contentType string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("typesense %s %s: build request: %w", method, path, err)
	}
	req.Header.Set("X-TYPESENSE-API-KEY", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("typesense %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	return decodeResponse(resp, method, path, out)
}

// decodeResponse inspects the status code and either decodes the
// body into `out` or wraps it into a typed sentinel (ErrNotFound,
// ErrUnauthorized) so callers can branch cleanly.
func decodeResponse(resp *http.Response, method, path string, out any) error {
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		if out == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			return nil
		}
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil && err != io.EOF {
			return fmt.Errorf("typesense %s %s: decode body: %w", method, path, err)
		}
		return nil
	case http.StatusNotFound:
		_, _ = io.Copy(io.Discard, resp.Body)
		return ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		_, _ = io.Copy(io.Discard, resp.Body)
		return ErrUnauthorized
	}

	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("typesense %s %s: status %d: %s",
		method, path, resp.StatusCode, strings.TrimSpace(string(b)))
}
