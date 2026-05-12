# Plan — Presence Bidirectionality Fix

## Bug

User A connects → presence broadcast informs nobody (no peers).
User B connects → `SetOnline(B)` + `broadcastPresenceChange(B, online)` fans
out to B's contacts, **including A** ⇒ A learns B is online via the
`presence` frame and invalidates `conversations`.

But: B never receives a `presence` snapshot for A. B sees the cached state
from REST → `Offline` for A until the next `presence` event (e.g. A
disconnects). This is the unidirectional bug.

## Fix — two complementary layers

### A. Backend — `presence_snapshot` frame on connect (primary)

After `SetOnline(newUser)` in `ServeWS`, send a NEW frame to `newUser`
only:

```json
{"type":"presence_snapshot","payload":{"online_user_ids":["uuid",...]}}
```

Scope: intersection of (`newUser.contactIDs` from
`MessagingSvc.GetContactIDs`) with `presence.BulkIsOnline`. Single batch
query — no N+1. Privacy-safe (no global leak).

### B. Web — refetch on WS `open` (safety net)

In `use-messaging-ws.ts`, on `ws.onopen`, also call
`queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })`.
Backend `BulkIsOnline` already powers the conversation `online` field, so
this gives a fresh answer.

### C. Web — handle `presence_snapshot` frame

Add a case in the frame switch: invalidate `conversations` query. (Direct
cache patching is a future optimization.)

### D. Mobile parity — handle `presence_snapshot`

`messaging_ws_service.dart` already streams events as Maps; the existing
`presence` listener path is in higher layers. We forward
`presence_snapshot` through the same stream so existing presence
listeners can react. Add a synthetic invalidation hook trigger.

## Files modified

### Backend
- `backend/internal/adapter/ws/types.go` — add `TypePresenceSnapshot` constant.
- `backend/internal/adapter/ws/connection.go` — after `SetOnline`, compute scoped online set + send snapshot.
- `backend/internal/adapter/ws/send_or_drop_test.go` — extend fakes if needed.
- `backend/internal/adapter/ws/presence_snapshot_test.go` — NEW: 3 tests (snapshot scoped to partners, empty when no convos, batched query).

### Web
- `web/src/features/messaging/types.ts` — add `presence_snapshot` variant.
- `web/src/features/messaging/hooks/use-messaging-ws.ts` — handle `presence_snapshot` + invalidate on `open`.
- `web/src/features/messaging/hooks/__tests__/use-messaging-ws.test.ts` — 2 new tests.

### Mobile
- `mobile/lib/features/messaging/data/messaging_ws_service.dart` — no code change needed (passes through `presence_snapshot` as-is via the existing event stream). Document it in the dartdoc.
- Higher-layer consumer (whoever reacts to `presence`) — we ensure the snapshot is observable; add a unit test that asserts the snapshot map shape is decoded into the events stream.
- `mobile/test/features/messaging/data/messaging_ws_service_presence_snapshot_test.dart` — NEW: 1 widget/unit test asserting frame forwarded through stream.

## Scope decision

- Snapshot scope = conversation partners (other users sharing a conversation with `newUser`). Uses existing `GetContactIDs` port — no new infra. Aligned with privacy + scale.
- Reuse `BulkIsOnline` (single batched call) on the existing `PresenceService` port.
- No new wiring outside `ServeWS` + the existing types.
- Frontend: invalidate conversations query (already the existing handler for `presence`).

## Test count

- Backend: 3 new tests in `presence_snapshot_test.go`
- Web: 2 new tests in `use-messaging-ws.test.ts`
- Mobile: 1 new test in `messaging_ws_service_presence_snapshot_test.dart`

Plus a regression assertion that the existing `presence` handler still
exists.

## Compliance

- All file sizes < 600 LOC; new function < 50 LOC.
- Hexagonal: ws adapter calls `service.PresenceService.BulkIsOnline` + `service.MessagingQuerier.GetContactIDs` (ports). No new ports.
- LiveKit untouched. Legal/wallet/proposal/billing untouched.
