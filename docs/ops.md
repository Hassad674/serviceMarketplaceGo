# Operations runbook — search engine

Operational playbook for everything that can wake an on-call engineer:
deploys, reindexes, key rotation, snapshots, drift alerts, slow-query
triage, and incident response.

Read `docs/search-engine.md` for the architecture; this doc is for the
moments when architecture is not the question.

---

## 1. Deploy order

The search engine spans three processes and three clients. Deploying
out of order causes user-visible errors (401, stale filters, missing
fields). Always follow this sequence.

1. **Apply migrations** first, on a DB copy, then on the shared DB:
   ```bash
   cd backend
   DATABASE_URL=<staging> make migrate-up
   DATABASE_URL=<prod>    make migrate-up
   ```
   Migrations are additive — old app code still runs fine against the
   new schema.
2. **Deploy backend** (zero-downtime rolling deploy). Readiness probe
   (`GET /ready`) now requires a healthy Typesense connection since
   phase 4 — a misconfigured instance is rotated out automatically.
3. **Deploy web** after backend — `/api/v1/search/key` + `/api/v1/search`
   must exist before the client code reaches them.
4. **Release mobile** last. Mobile stores bundle the old backend URL
   for 12-48h while the store approval completes, so the backend must
   stay backwards-compatible during that window.

### What to restart when

| Change | Restart |
|--------|---------|
| Go code under `internal/search/**` | backend |
| Go code under `internal/app/searchindex/**` | backend + worker (same binary) |
| SQL migration | backend (after migrations apply) |
| Typesense config (synonyms, schema) | no restart — `EnsureSchema` runs on boot and upserts synonyms idempotently |
| OpenAI API key rotation | backend (picks up the new env var) |
| `TYPESENSE_API_KEY` (master key rotation) | backend — see §3 |
| Web search components | web (Next.js redeploy) |
| Mobile search code | mobile (store release) |

---

## 2. Reindex in production

### Full reindex (idempotent, ~30s per 1k profiles)

```bash
cd backend
DATABASE_URL=<prod> \
TYPESENSE_HOST=<prod> \
TYPESENSE_API_KEY=<prod> \
OPENAI_API_KEY=<prod> \
make reindex-bulk
```

Flags: `ARGS="--persona=freelance --batch=200"` scopes to one persona
and tunes the batch size.

### Safety flags

- Always run `--persona=<one>` first on prod. Verify drift dropped to
  0. Then repeat for the others.
- Never run a full reindex during peak hours. The live OpenAI calls
  rate-limit around 3000 req/min; a 50k-profile reindex takes ~18
  minutes and keeps embeddings warm.
- The reindex is safe to re-run on failure — the CLI is idempotent
  and upserts by `id`. Interrupting mid-run is safe.

### Zero-downtime schema migration (alias swap)

For any change to the SearchDocument shape:

1. Create `marketplace_actors_v2` with the new schema (CLI or
   manually via Typesense admin API).
2. Bulk reindex into `v2` via a one-off flag we have not yet
   implemented — document the intent: `--collection=marketplace_actors_v2`.
3. Swap the alias atomically:
   ```
   PUT /aliases/marketplace_actors { "collection_name": "marketplace_actors_v2" }
   ```
4. Verify traffic shifted (monitor `typesense_drift_ratio` gauge).
5. After 24h, drop `marketplace_actors_v1`.

---

## 3. Rotate the search master API key

The master `TYPESENSE_API_KEY` signs every scoped key handed to
clients. Rotating it without care causes mass 401s for clients still
holding the old scoped key.

### Graceful rollout (no user 401s)

1. Create a new master key in Typesense admin with identical ACL:
   ```
   POST /keys { "description": "search master v2", "actions": ["*"], "collections": ["*"] }
   ```
2. Deploy backend with BOTH keys present:
   ```
   TYPESENSE_API_KEY=<new-master>
   TYPESENSE_API_KEY_LEGACY=<old-master>
   ```
   (Requires a small code change: `scoped_key.go` accepts an array of
   signing keys and tries each on verify. Out of scope for phase 5C —
   flagged in the future-work list.)
3. Wait 2 hours (worst case scoped-key TTL).
4. Remove `TYPESENSE_API_KEY_LEGACY` and redeploy.
5. Delete the old master key from Typesense.

### Emergency rotation (some user 401s acceptable)

1. Generate the new key.
2. Deploy backend with the new key (single-key mode).
3. All existing scoped keys fail HMAC verification — clients retry
   `/api/v1/search/key` and get a fresh key. Expect a spike of 401s
   for ~55 minutes (the web cache TTL).

---

## 4. Typesense snapshot + restore

### Daily snapshot (production orchestrator)

`make snapshot-typesense` triggers Typesense's `/operations/snapshot`,
tars + gzips the resulting `/data/snapshot` directory, and uploads to
MinIO at `snapshots/typesense/YYYY-MM-DD.tar.gz`.

Production runs this via a systemd timer (example unit file in
`cmd/typesense-snapshot/main.go`'s package doc comment). The
`.github/workflows/snapshot.yml` workflow exists as cadence
documentation, not the actual runner.

### Restore

1. Stop the Typesense container.
2. Download the target snapshot from MinIO and extract into a fresh
   data volume:
   ```bash
   aws s3 cp s3://snapshots/typesense/2026-04-17.tar.gz .
   mkdir -p /var/lib/typesense
   tar -xzf 2026-04-17.tar.gz -C /var/lib/typesense
   ```
3. Start Typesense pointing at the new volume:
   ```bash
   docker run -v /var/lib/typesense:/data typesense/typesense:28.0 \
     --data-dir /data --api-key <master>
   ```
4. Verify collection integrity:
   ```bash
   curl -H "X-TYPESENSE-API-KEY: $KEY" http://localhost:8108/collections/marketplace_actors
   ```
5. Swap the alias if you restored into a fresh name.
6. Reindex anything that happened AFTER the snapshot by replaying the
   `pending_events` table (filter by `created_at > snapshot_time`).

### Retention

- Keep 7 daily snapshots.
- Keep 4 weekly (Sunday) snapshots.
- Keep 12 monthly (1st of month) snapshots.
- Archive to MinIO "cold" tier after 30 days.

---

## 5. Drift alerts

An hourly cron (`.github/workflows/drift.yml`) runs
`make drift-check` against staging. When drift exceeds 0.5%, a GitHub
issue is opened (or commented on if already open) with the full log.

### Triage flow

1. **Read the issue body** — it contains the per-persona counts. Is
   Postgres ahead or Typesense ahead?
2. **Postgres ahead = index lag.** A reindex event was dropped or the
   worker is paused. Check:
   ```bash
   # Is the search worker running?
   kubectl logs -l app=marketplace-worker --tail=100 | grep search
   # Are there stuck events?
   psql $DATABASE_URL -c "SELECT status, count(*) FROM pending_events WHERE event_type LIKE 'search.%' GROUP BY status"
   ```
   Fix: restart the worker, or run `make reindex-bulk --persona=<drifted>`.
3. **Typesense ahead = orphan docs.** A user was deleted but the
   `search.delete` event failed. Check:
   ```bash
   psql $DATABASE_URL -c "SELECT id FROM organizations WHERE deleted_at IS NOT NULL AND deleted_at > now() - interval '1 day' ORDER BY deleted_at DESC LIMIT 20"
   ```
   Compare those IDs to Typesense hits. Run delete commands manually
   if needed; then file a bug so the deletion pipeline is fixed.
4. **Close the issue** once drift returns to 0. Do not close prematurely
   — the hourly cron will reopen it if drift reappears.

---

## 6. Slow query triage

Every `/api/v1/search` call emits one structured JSON log line. To
find slow queries:

```bash
# Last 200 slow queries (>500ms) with context
grep '"event":"search.query"' /var/log/marketplace/backend.jsonl \
  | jq 'select(.latency_ms > 500)' \
  | tail -200
```

Narrow further:

```bash
# By persona
jq 'select(.persona == "freelance" and .latency_ms > 500)' backend.jsonl

# Frequent zero-result queries
jq 'select(.results_count == 0) | .query' backend.jsonl \
  | sort | uniq -c | sort -rn | head -20
```

### Admin stats endpoint

`GET /api/v1/admin/search/stats?from=2026-04-10&to=2026-04-17&persona=freelance`
returns top queries, zero-result queries, and p95 latency from the
`search_queries` table. Use this as the first stop for a user-reported
"search is slow" ticket.

### EXPLAIN ANALYZE on the analytics table

If the admin stats endpoint itself is slow:

```sql
EXPLAIN (ANALYZE, BUFFERS)
SELECT lower(query), count(*)
FROM search_queries
WHERE created_at BETWEEN '2026-04-10' AND '2026-04-17'
  AND persona = 'freelance'
GROUP BY 1 ORDER BY 2 DESC LIMIT 50;
```

The `idx_search_queries_persona_created_at` index must serve the
`WHERE` clause. If you see a Seq Scan, the index is missing or the
planner chose wrong — check `pg_stat_user_indexes`.

### Typesense-side latency

When the Typesense response itself is slow:

1. `GET /debug?api-key=<master>` returns cluster stats (memory, disk,
   query count).
2. `GET /collections/marketplace_actors` returns `num_documents`.
   Above 500k, vector query latency grows noticeably; consider
   sharding or moving vector search to a separate collection.
3. Check the hybrid-vs-BM25 ratio in logs:
   `jq 'select(.hybrid == true)' | wc -l`. Hybrid queries are 3-5x
   slower than BM25; a sudden ratio shift points at a client-side
   regression that forgot to disable hybrid on empty queries.

---

## 7. Incident response

### Typesense cluster down

**Symptoms**: `/ready` endpoint returns 503 on every backend instance;
`/api/v1/search` returns 503.

**Immediate action** (within 5 minutes):
1. Check the cluster health: `curl $TYPESENSE_HOST/health`.
2. If it returns 503 or times out, the cluster is genuinely down.
3. Notify the team in #search-engine-incidents.
4. Begin the failover playbook:
   - Restart the Typesense container. The snapshot restore (§4) is
     the last resort — only if the data volume is corrupt.
   - `/api/v1/search` remains 503 during the restart. Users see the
     error banner; no silent degradation.

**Fallback to SQL search**: phase 4 removed the `SEARCH_ENGINE=sql`
feature flag. To restore SQL fallback in an emergency, revert commit
`c98e602` in a hotfix branch and deploy. The SQL path is slower and
keyword-only (no semantic), but it keeps search functional while
Typesense is rebuilt.

### OpenAI embeddings API down

**Symptoms**: `search_embedding_retries_total` metric rises sharply;
`search.reindex` events pile up in `pending_events`.

**Immediate action**:
1. The `RetryingEmbeddingsClient` already retries 3x with exponential
   backoff. If all retries fail, the event is left in `pending_events`
   for the next retry cycle.
2. Queries continue to work — they use stored embeddings.
3. If the outage lasts >1 hour, consider pausing the search worker
   to prevent queue buildup.
4. When OpenAI is back, the queue drains automatically.

### Scoped-key HMAC verification fails

**Symptoms**: every user sees 401 on `/api/v1/search`; hockey-stick
spike on `search_requests_total{status="error"}`.

**Cause**: either the master key was rotated without a graceful
rollout (§3), or the HMAC signing code was changed and deployed
without a matching client-side rebuild.

**Immediate action**:
1. Inspect a failing response — does it contain "signature mismatch"?
   Confirms HMAC breakage.
2. Rollback the latest backend deploy.
3. Users retry `/api/v1/search/key`, get a valid key, and recover
   within one cache cycle (55 minutes max).

---

## 8. Observability quick reference

### Prometheus endpoint

`GET /metrics` on every backend instance returns:

```
search_requests_total{persona="freelance",status="success"} 42
search_duration_seconds_bucket{persona="freelance",hybrid="false",le="0.1"} 30
search_results_count_bucket{persona="freelance",le="10"} 12
search_embedding_retries_total 3
search_reindex_duration_seconds_bucket{le="30"} 5
typesense_drift_ratio 0.001
```

Scrape with Prometheus, Grafana Agent, or any compatible collector.
The endpoint is unauthenticated — bind the backend port to an internal
network or front it with a reverse-proxy ACL.

### Structured log locations

- Search query log: every `/api/v1/search` request emits one JSON
  line with 12 pinned fields (`event="search.query"`).
- Search error log: `event="search.error"` with `error` (sanitised)
  and `request_id`.
- Reindex log: `event="search.reindex.done"` with duration + count.
- Drift log: `event="search.drift"` with per-persona diff.

### Health endpoints

| Endpoint | Response |
|----------|----------|
| `GET /health` | Liveness. Always 200 if process up. |
| `GET /ready` | Readiness. 200 if Postgres + Redis + Typesense all reachable. Returns 503 if any probe fails. |
| `GET /metrics` | Prometheus scrape. 200. |

---

## 9. What this runbook does NOT cover

- New collection creation → `docs/search-engine.md` §schema migration.
- Adding a new persona → product spec in memory
  (`project_search_engine_spec.md`).
- Debugging vector quality → the live golden suite (§ testing.md).
- MinIO / Postgres / Redis recovery → not search-specific; follow the
  main infra runbook.
