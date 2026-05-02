# 0008. Cursor pagination over OFFSET

Date: 2026-04-30

## Status

Accepted

## Context

List endpoints return paginated collections (search results,
messages, jobs, proposals, audit log entries). The classic
SQL pagination tool — `LIMIT N OFFSET K` — has two well-known
failure modes that bite us at marketplace scale:

1. **Performance degradation**. `OFFSET 10000` requires the
   database to scan the first 10 000 rows and discard them.
   For a busy listing (search results sorted by ranking score)
   this becomes O(K + N) on the database side. We measured
   `OFFSET 5000 LIMIT 20` on the proposals listing taking 850 ms
   p95 in dev — well past our 100 ms p95 target.
2. **Drift under concurrent writes**. If a row is inserted
   between two page fetches, the user either sees the same row
   on consecutive pages or skips one. On a high-write surface
   (messaging) this is a daily UX bug.

Cursor pagination — encode the position as an opaque token
embedding the last row's sort key — solves both:

- The query becomes `WHERE (sort_key, id) > ($cursor_key,
  $cursor_id) ORDER BY sort_key, id LIMIT N`. With an index on
  `(sort_key, id)`, the database does an index range scan
  bounded by `N` rows, regardless of how deep the user is.
- Inserts after the cursor was issued do not affect already-seen
  pages. The cursor stays consistent.

The trade-off: cursor pagination does not support **arbitrary
page jumps** ("go to page 5"). The user can only paginate
forward (or backward, if the cursor encodes both directions).
For our UX — infinite-scroll lists, "load more" buttons —
this is acceptable. We do not expose page numbers.

## Decision

Every list endpoint uses **cursor pagination**. We provide a
shared `pkg/cursor` helper that encodes the (sort_key, id)
tuple as a base64-URL string and decodes it on the next
request.

Concrete pattern:

1. **Request shape**:
   ```
   GET /api/v1/proposals?cursor=<opaque>&limit=20
   ```
   `limit` is bounded server-side (default 20, max 100).
2. **Response shape**:
   ```json
   {
     "data": [...],
     "next_cursor": "eyJrIjoxNzAxMjM0NTY3OCwiaSI6ImFiYyJ9",
     "has_more": true
   }
   ```
   `has_more` is a derived field: `true` iff the query returned
   `limit + 1` rows (we fetch one extra and discard for the
   detection). When `has_more` is `false`, `next_cursor` is `""`.
3. **SQL pattern**:
   ```sql
   SELECT id, sort_key, ...
   FROM   proposals
   WHERE  organization_id = $1
     AND  (sort_key, id) < ($2, $3)  -- cursor decode
   ORDER  BY sort_key DESC, id DESC
   LIMIT  21;                         -- limit + 1
   ```
4. **Cursor encoding** (`pkg/cursor/cursor.go`):
   ```go
   type Cursor struct {
       SortKey int64  `json:"k"`  // unix nanos for time-sorted lists
       ID      string `json:"i"`
   }
   func Encode(c Cursor) string { /* JSON + base64.URLEncoding */ }
   func Decode(s string) (Cursor, error)
   ```
   The cursor is opaque to clients — they pass it back unchanged.
5. **Index requirement**: every paginated list MUST have a
   composite index `(sort_key, id)` with the same DESC/ASC
   direction as the query. We verify this with `EXPLAIN ANALYZE`
   in the integration tests.

Admin endpoints follow the same pattern; admin listing of
disputes, invoices, conversations all use cursor pagination.

## Consequences

### Positive

- Pagination performance is O(N) regardless of depth. We measured
  the same `proposals` listing at 12 ms p95 with cursor
  pagination — a 70x improvement.
- Lists stay consistent under concurrent writes. The "missing
  message" bug we used to hit on chat is gone.
- The shared `pkg/cursor` helper means every feature follows
  the same shape. New endpoints get pagination "for free" from
  the boilerplate.
- Frontend infrastructure (TanStack Query's `useInfiniteQuery`)
  natively supports cursor pagination, so the integration is
  trivial.

### Negative

- No "go to page 5" jump. Users who expect a page number control
  in the UI find it absent. We accept this — every modern
  product (Stripe Dashboard, GitHub PRs, Linear) uses
  infinite-scroll.
- Cursor format is opaque. If we ever need to migrate cursor
  format (e.g. add a third field), every in-flight client
  request becomes invalid for a moment. We mitigate by versioning
  the cursor (current version is `v=1`; a `v=2` cursor would be
  decoded by both old and new code paths during a deprecation
  window).
- Composite index requirement adds one index per list. Storage
  cost is minor; insertion cost (one extra B-tree update) is
  measurable but acceptable on our write rates.

## Alternatives considered

- **OFFSET pagination** — what we had originally. Performance
  collapsed past page 50 on hot lists. Rejected.
- **Keyset pagination with explicit `since=<id>`** — a simpler
  variant that works only on monotonic IDs. Insufficient for
  lists sorted by mutable fields (search ranking score, last
  message timestamp). Rejected.
- **Database-side window functions for "page N"** — clever but
  pushes complexity into PostgreSQL with no UX benefit. Rejected.

## References

- `backend/pkg/cursor/cursor.go` — encode / decode helpers.
- `backend/pkg/cursor/cursor_test.go` — round-trip + invalid
  input tests.
- `backend/internal/adapter/postgres/proposal_repository.go`
  — representative SQL pattern.
- `web/src/shared/lib/search/search-api.ts` and
  `web/src/shared/lib/review/review-api.ts` — representative
  client-side cursor consumers via TanStack Query.
- Mark Callaghan, *Pagination using cursors*,
  <https://use-the-index-luke.com/no-offset>.
