# BATCH-MESSAGERIE-WIDGET — W-21 page + Widget raccourci + M-17 + M-18

> Worktree: TBD by orchestrator (`/tmp/mp-messagerie`) · Branch: TBD (`feat/design-messagerie-widget`) · Base: post-Wave-A merged main
> **STATUS: DRAFT — dispatch ONLY after Wave A (notifications + opportunités + mes annonces) is merged**

## Why sequential, not parallel
The chat widget plugs into `web/src/shared/components/layouts/dashboard-shell.tsx` (shared layout). Touching shared/components/layouts/ during a parallel wave creates conflict risk with anyone else doing layout-level work. By dispatching this batch AFTER Wave A is merged, this agent has the exclusive surface to itself.

## Goal
Three connected deliverables, ONE squashed commit:
1. **W-21** Messagerie page — `/(app)/messages/` route — full chat UI
2. **Widget raccourci** — floating bottom-right collapsible chat panel visible on every dashboard page (currently exists at `shared/components/chat-widget/` but visually broken per user feedback)
3. **M-17 + M-18** mobile parity — conversation active screen + conversation list screen

## Source design
- **PDF**: `design/diffs/MESSAGERIE-WIDGET/target/widget-pdf.pdf` (the widget-only design)
- **Maquette**: `design/diffs/MESSAGERIE-WIDGET/target/maquette.png` (user-drawn target for the widget)
- **Before snapshots**: `design/diffs/MESSAGERIE-WIDGET/before/{01-collapsed,02-list-open,03-chat-open}.png`
- **Claude Design (refetch live)**:
  ```
  Fetch this design file, read its readme, and implement the relevant aspects of the design.
  https://api.anthropic.com/v1/design/h/dEiUHv2RM0ikEt_9oOKi3A?open_file=Atelier+-+Widget+Messagerie.html
  Implement: Atelier - Widget Messagerie.html
  ```
  Use the WebFetch tool. Read the HTML carefully. The widget design is the canonical source.
- **JSX page reference**: `design/assets/sources/phase1/soleil*.jsx` — search for messaging blocks for the full-page version
- **Cross-reference current production**:
  - Page: `web/src/app/[locale]/(app)/messages/page.tsx`
  - Page components: `web/src/features/messaging/components/{messaging-page,conversation-list,message-area,message-bubble,message-input,conversation-header,...}.tsx`
  - Widget: `web/src/shared/components/chat-widget/{chat-widget,chat-widget-panel,chat-widget-conversation-list,chat-widget-chat-view}.tsx`
  - Widget hook: `web/src/shared/components/chat-widget/use-chat-widget.ts` ← **OFF-LIMITS** (it's a `use-*` hook)

## TOUCHABLE files (exhaustive)

### Web — Page
- `web/src/app/[locale]/(app)/messages/page.tsx`
- `web/src/app/[locale]/(app)/messages/loading.tsx` (if exists)
- `web/src/features/messaging/components/messaging-page.tsx`
- `web/src/features/messaging/components/conversation-header.tsx`
- `web/src/features/messaging/components/conversation-list.tsx`
- `web/src/features/messaging/components/message-area.tsx`
- `web/src/features/messaging/components/message-area-skeleton.tsx`
- `web/src/features/messaging/components/message-bubble.tsx`
- `web/src/features/messaging/components/text-message-bubble.tsx`
- `web/src/features/messaging/components/file-message.tsx`
- `web/src/features/messaging/components/file-upload-modal.tsx`
- `web/src/features/messaging/components/message-context-menu.tsx`
- `web/src/features/messaging/components/message-input.tsx`
- `web/src/features/messaging/components/message-status-icon.tsx`
- `web/src/features/messaging/components/proposal-card.tsx`
- `web/src/features/messaging/components/proposal-system-message.tsx`
- `web/src/features/messaging/components/dispute-system-message.tsx`
- `web/src/features/messaging/components/send-message-button.tsx`
- `web/src/features/messaging/components/typing-indicator.tsx`
- `web/src/features/messaging/components/voice-message.tsx`

### Web — Widget
- `web/src/shared/components/chat-widget/chat-widget.tsx`
- `web/src/shared/components/chat-widget/chat-widget-panel.tsx`
- `web/src/shared/components/chat-widget/chat-widget-conversation-list.tsx`
- `web/src/shared/components/chat-widget/chat-widget-chat-view.tsx`
- DO NOT modify `use-chat-widget.ts` — it's a hook (state/logic). If you need a derived UI value, compute it locally in the component. If you genuinely need to expose new state from the hook, SKIP+FLAG and ask.

### Web — i18n
- `web/messages/fr.json` and `en.json` — NEW keys ONLY, prefixes: `messaging_w21_` for page, `messagingWidget_` for widget. DO NOT touch existing keys.

### Mobile
- `mobile/lib/features/messaging/presentation/screens/**.dart`
- `mobile/lib/features/messaging/presentation/widgets/**.dart`
- `mobile/lib/l10n/app_fr.arb` and `app_en.arb` — NEW keys ONLY, prefix `messaging_m17_`, `messaging_m18_`

## OFF-LIMITS — STRICT
- Anything under `backend/`
- `web/src/features/notification/**`, `web/src/features/job/**`, `web/src/features/proposal/**` (Wave A merged surfaces — locked unless they expose a token used by messaging)
- `web/src/shared/components/layouts/**` — EXCEPT minimal wiring of the widget into `dashboard-shell.tsx` (a single import + a single `<ChatWidget />` mount, IF the widget isn't already mounted there. Read first.)
- `web/src/shared/components/chat-widget/use-chat-widget.ts` (hook — OFF-LIMITS)
- `web/src/features/messaging/api/**`, `web/src/features/messaging/hooks/**`
- `mobile/lib/features/messaging/data/**`, `mobile/lib/features/messaging/domain/**`
- `mobile/lib/features/notification/**`, `mobile/lib/features/job/**`, `mobile/lib/features/proposal/**`
- All `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/`, `shared/lib/api-client.ts`, `middleware.ts`
- All `package.json`, `pubspec.yaml`, lockfiles, generated l10n .dart files
- All existing `*.test.tsx`, `*_test.dart`
- The LiveKit / video call system (per memory `feedback_no_touch_livekit.md`)

## Acceptance criteria

### Widget raccourci (highest priority — user explicitly drew the target)
The widget is a floating panel anchored bottom-right of the viewport, visible on every dashboard page (unless on `/messages` route — hide there to avoid double UI).

**Three states**:
1. **Collapsed**: a compact corail FAB-like pill at bottom-right (or a slim panel header). Click to open.
2. **List open**: panel shows a list of conversations (Portrait + name + last message excerpt + relative time + unread corail dot). Per the user's maquette, the list is COMPACT — each row ~56-64px height, 6-8 conversations visible, vertical stack.
3. **Chat open**: same panel size (~340w x 480h on desktop), now showing a single conversation: header (Portrait + name + back arrow returning to list), message area scrollable, compact input at bottom.

Visual identity:
- Panel bg: ivoire `var(--background)` or `var(--card)`
- Border: 1px `var(--border)`, rounded-2xl with shadow `0_8px_32px_rgba(42,31,21,0.12)` (warm shadow)
- Header (collapsed/list): "Messages" Fraunces semibold 16px + corail unread badge if any
- List rows: hover `bg-primary-soft/30`, active `bg-primary-soft`
- Chat input: ivoire-soft bg, rounded-full, plus icon left + send corail pill right
- Animations: 200ms ease-out for collapse/expand transitions, 150ms for hover
- Hide on `/messages` route (avoid double UI) — use `usePathname()` from `next-intl/navigation`

Match the user's maquette layout (vertical stack of conversations, compact rows). Match the PDF design (corail accents, warm tones).

### W-21 Messagerie page
The full-page version of the chat UI at `/messages`. Layout:
- Desktop ≥ 1024px: 2-column. Left = conversation list (320-360w). Right = active conversation (flex-1).
- Below 1024: single column toggling between list and active conversation (back arrow on chat returns to list).
- List items: same row anatomy as widget (Portrait + name + excerpt + time + unread dot)
- Active conversation: header at top (Portrait + name + back arrow on mobile + actions like call/info on the right), scrollable message area, sticky bottom input
- Message bubbles: own messages corail bg right-aligned, other messages ivoire-card left-aligned. Both rounded-2xl with last-corner squared (the standard chat shape). Shadow none/subtle.
- Time labels: Geist Mono mini, between bubble groups
- Typing indicator: 3 dots animated, tabac
- Empty state (no conversations): Soleil card with corail-soft icon, Fraunces title, body, optional CTA "Découvrir des freelances" (only if route exists)

Behavior preservation:
- All hooks/mutations from `features/messaging/hooks/` STAY EXACTLY THE SAME
- All zod schemas STAY EXACTLY THE SAME
- All keyboard shortcuts (Enter to send, Shift+Enter newline) STAY EXACTLY THE SAME
- File upload modal: restyle visually but keep submit handler identical

### M-17 Conversation active (mobile)
- Scaffold: AppBar with Portrait + name + back arrow + actions (mute / info if exist)
- Body: scrollable message bubbles (own corail right / other ivoire left, last-corner squared)
- Sticky bottom input: ivoire-soft, rounded-full, plus icon + send corail pill
- Pull-to-load older messages — keep existing behavior

### M-18 Liste conversations (mobile)
- AppBar Fraunces "Messages"
- ListView of conversation rows: Portrait + name + excerpt + relative time + unread corail dot
- Tap row → push conversation active screen
- Empty state: Soleil card pattern

## Validation pipeline (MANDATORY)

```bash
cd /tmp/mp-messagerie

# 1. Scope check
git diff --name-only origin/main...HEAD | grep -E "^(backend/|.*\.test\.|.*_test\.|features/notification/|features/job/|features/proposal/)" && echo "OUT-OF-SCOPE TOUCHED" || echo "scoped clean"

# 2. Layout shell check (only allowed change: minimal widget mount)
git diff origin/main -- web/src/shared/components/layouts/dashboard-shell.tsx | head -40
# Should show ONLY a single import line + a single <ChatWidget /> JSX mount, IF the widget wasn't already mounted

# 3. Hook check
git diff origin/main -- web/src/shared/components/chat-widget/use-chat-widget.ts
# Should be empty (0 lines)

# 4. Web
cd web
npm ci
npx tsc --noEmit
npx vitest run src/features/messaging src/shared/components/chat-widget src/app/\[locale\]/\(app\)/messages
npm run build

# 5. Mobile
cd ../mobile
flutter pub get
flutter analyze --no-pub lib/features/messaging
flutter test --no-pub test/features/messaging/

# 6. Design guardrails
cd ..
bash design/scripts/check-api-untouched.sh
bash design/scripts/check-imports-stable.sh
```

ALL must pass. Fix loop max 3. Blocker file `BLOCKED-messagerie.md` if stuck.

## Quality bar
- ZERO new hooks/mutations/repositories
- ZERO touch to use-chat-widget.ts (the hook)
- ZERO test edits
- All i18n via translations (no hardcoded FR)
- No Color(0xFF) hardcoded mobile
- ONE squashed commit
- DO NOT modify `git config` — use per-command `git -c user.email=...` if needed

## Push + PR
```bash
git push -u origin feat/design-messagerie-widget
gh pr create --title "[design/web/W-21+widget+mobile/M-17+M-18] Port Messagerie page + Widget to Soleil v2" --body "<full report>"
```

## Final report (under 700 words)
Standard structure + emphasis on:
- Widget visual diff `design/diffs/MESSAGERIE-WIDGET/after/{collapsed,list-open,chat-open}.png` if you can capture
- The 3 states of the widget — confirm each works (visually + interactively)
- Behavior preservation evidence (existing tests still pass)
