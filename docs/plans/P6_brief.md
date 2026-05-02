# P6 — Migration 133 dénormalisation `last_message` sur `conversations`

**Phase:** F.2 HIGH #3
**Source audit:** PERF-FINAL (`auditperf.md`) — N+1 sur conversation list
**Effort:** 1j est.
**Tool:** 1 fresh agent dispatched
**Branch:** `fix/p6-denormalize-last-message`

## Problem

`/api/v1/messaging/conversations` (conversation list) currently does N+1: for each conversation, a separate query fetches the last message. Audit estimates this is the #2 hottest backend path after profile reads. Eliminate via denormalization.

## Decision (LOCKED — user validated)

**Maintenance applicatif** au moment du `INSERT messages` (PAS trigger PG). Reasons:
- Explicit control (visible in code, debuggable)
- Easier to test
- No magic SQL
- Same transaction = atomic vs other writes

## Plan (6 commits)

### Commit 1 — Migration 133
- `migrations/133_denormalize_last_message.up.sql` :
  ```sql
  ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS last_message_seq INT,
    ADD COLUMN IF NOT EXISTS last_message_content_preview TEXT,
    ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_message_sender_id UUID;
  
  -- Backfill from existing messages
  UPDATE conversations c
  SET last_message_seq = m.seq,
      last_message_content_preview = LEFT(m.content, 100),
      last_message_at = m.created_at,
      last_message_sender_id = m.sender_id
  FROM (
    SELECT DISTINCT ON (conversation_id)
      conversation_id, seq, content, created_at, sender_id
    FROM messages
    ORDER BY conversation_id, seq DESC
  ) m
  WHERE c.id = m.conversation_id;
  ```
- Symmetric down.sql

### Commit 2 — Adapter update
- `adapter/postgres/conversation_repository.go::createMessageInTx` :
  - Add UPDATE conversations SET last_message_* in same tx as INSERT messages
  - Truncate content to 100 chars via `LEFT(content, 100)` or Go `if len > 100`
- Tests

### Commit 3 — List query optimization
- `adapter/postgres/conversation_queries.go::ListConversations` :
  - Use the new denormalized columns instead of correlated subquery
  - Verify EXPLAIN ANALYZE shows index scan, no per-row subquery
- Tests with sqlmock asserting query shape

### Commit 4 — Dispute / system messages
- Audit every `SendSystemMessage` call site to ensure they also update last_message_* (system messages count as last activity)
- Tests

### Commit 5 — Bench + integration
- `BenchmarkListConversations_HappyPath` — measure N=50 conversations
- Integration test with testcontainers : insert N messages, list, assert last_message_* match
- Asserter zero N+1 (1 query for list, 0 followups)

### Commit 6 — Docs + cleanup
- Update `backend/docs/messaging.md` if exists, else add note in domain comment
- Remove the legacy correlated-subquery code path

## Hard constraints

- **Zero behaviour change** on the API surface: same fields returned, same ordering
- **Validation pipeline before EVERY commit**: `go build && go vet && go test ./... -count=1 -short -race`
- **EXPLAIN ANALYZE delta**: paste before/after in PR description, asserter "Seq Scan" → "Index Scan" or N+1 collapse
- Migration up + down both applied + verified locally

## OFF-LIMITS

- LiveKit / call code, workflow files, other plans
- Trigger PG approach (decision locked)

## Branch ownership

Agent creates `fix/p6-denormalize-last-message` from clean main. Single branch only.

## Final report (under 600 words)

PR URL first. Then:
1. EXPLAIN ANALYZE before/after
2. Bench delta (`BenchmarkListConversations_HappyPath`)
3. Migration applied yes/no
4. Tests count
5. Validation pipeline output
6. Branch ownership confirmed
