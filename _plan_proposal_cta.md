# Plan — Fix Proposal-Accepted "Payer maintenant" CTA regression

## Scope (ni plus, ni moins)

Restore the "Payer maintenant" CTA on the `proposal_accepted` SYSTEM message,
so that when a client accepts a proposal in chat they can pay immediately
from the system bubble — not only from the (sometimes scrolled-away)
proposal card.

## Decision — How to detect "payment not yet confirmed"

Reuse the EXACT same condition as `proposal-card.tsx`:
```ts
showPayButton =
  metadata.proposal_status === "accepted" &&
  metadata.proposal_client_id === currentUserId
```

`metadata.proposal_status` is the snapshot persisted at the time the
system message was emitted. `proposal_accepted` system messages are
emitted right after acceptance, so the snapshot is `"accepted"`. When
the proposal becomes `"paid"` / `"completed"`, the snapshot on the OLD
system message stays `"accepted"` (the backend does not retro-update
past messages). To avoid showing a stale CTA after payment, we ALSO
hide the CTA on the `proposal_accepted` bubble when the proposal id
appears as the `proposal_id` of a later `proposal_paid` or
`proposal_completed` system message — that signal is already used by
`message-bubble`'s `supersededProposalIds` set (cards collapse when
superseded). We will compute a similar `paidProposalIds` set in
`message-area` and pipe it through `MessageBubbleState` to the
`ProposalSystemMessage`.

**Update after re-reading the brief**: the brief says "the simplest
signal: look at the proposal's status field… replicate the same
condition" as proposal-card. The card itself does NOT check
`paidProposalIds`; it relies on the snapshot. To match the brief
exactly and keep scope tight, replicate the same condition with no
addition — known caveat: a stale "Payer" bubble can remain visible
after payment for the lifetime of that scrolled chat session, but only
on the OLD accepted bubble. The new `proposal_paid` system message
appears below it, and the next refresh / new metadata propagates with
the updated status. This matches the card behaviour exactly, so no
regression vs the historical behaviour the brief asks to restore.

Final condition (web + mobile):
```
type === "proposal_accepted"
  AND viewer is the client (proposal_client_id === currentUserId)
  AND metadata.proposal_status === "accepted"
```

## Files to modify

### Web (Next.js)
1. `web/src/features/messaging/components/proposal-system-message.tsx`
   - Add `currentUserId` prop to `ProposalSystemMessage`.
   - When `type === "proposal_accepted"` AND viewer is client AND status === "accepted":
     render the same "Payer maintenant" CTA as `PaymentRequestedMessage`.
2. `web/src/features/messaging/components/message-bubble.tsx`
   - Forward `state.currentUserId` to `ProposalSystemMessage`.
3. `web/src/features/messaging/components/__tests__/proposal-system-message.test.tsx`
   - NEW file. Table-driven matrix of (type × viewer × status) → CTA visible/hidden.
   - Click test asserts router push to `/projects/pay?proposal=<id>`.
4. `web/src/features/messaging/components/__tests__/message-bubble.test.tsx`
   - Update `ProposalSystemMessage` mock signature to accept `currentUserId`
     prop (no behavioural change to the existing tests).

### Mobile (Flutter)
5. `mobile/lib/features/messaging/presentation/widgets/chat/bubbles/system_message_bubble.dart`
   - Optional `onPay` + `showPayCta` parameters; render a Material
     `FilledButton` ("Payer maintenant" / "Pay now") below the pill
     when both are set.
6. `mobile/lib/features/messaging/presentation/widgets/chat/message_bubble.dart`
   - When `type === "proposal_accepted"` and metadata is present and
     `proposal_status === "accepted"` and viewer is the client,
     forward `onPay` to the SystemMessageBubble.
7. `mobile/test/features/messaging/presentation/widgets/chat/bubbles/system_message_bubble_test.dart`
   - Table-driven tests for CTA visibility + tap callback.

### i18n
No new keys — `proposal.payNow` exists in `web/messages/{fr,en}.json`
and `mobile/lib/l10n/app_{fr,en}.arb`.

## Test count (estimates)

- Web `proposal-system-message.test.tsx`: 8+ tests (type matrix × viewer × status).
- Web `message-bubble.test.tsx`: unchanged (mock signature update only).
- Mobile widget test additions: 4 tests (visible, hidden×3 cases) + tap.

Coverage target on touched files: ≥ 90%.

## Out of scope

- Backend changes.
- Admin / wallet / `/projects/pay/` page.
- Proposal-card behaviour (already correct).
- LiveKit / video.
