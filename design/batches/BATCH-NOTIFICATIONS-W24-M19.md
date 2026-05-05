# BATCH-NOTIFICATIONS — W-24 (web) + M-19 (mobile)

> Worktree: `/tmp/mp-notifications` · Branch: `feat/design-notifications` · Base: `origin/main` (cdf6beb4)

## Goal
Port the Notifications surface to Soleil v2, both web (W-24) and mobile (M-19). Single-screen each, low complexity. This is the simplest of Wave A — a clean Soleil-style list with empty state, filters (if present), and Mark-as-read affordances.

## Source design
- JSX: `design/assets/sources/phase1/soleil-lotE.jsx` (look for `Notifications` / `NotificationsPage` block)
- PDF: `design/assets/pdf/atelier-web-design-stash.pdf` (notifications page if present)
- Cross-reference current production: `web/src/app/[locale]/(app)/notifications/page.tsx`

## TOUCHABLE files (exhaustive)

### Web
- `web/src/app/[locale]/(app)/notifications/page.tsx`
- `web/src/app/[locale]/(app)/notifications/loading.tsx` (if exists)
- `web/src/features/notification/components/**.tsx` — list item, header, empty state, filter pills
- `web/src/messages/fr.json` and `en.json` — only NEW keys, prefix `notifications_w24_`. DO NOT touch existing notification i18n keys (keep their identifiers + values).

### Mobile
- `mobile/lib/features/notification/presentation/screens/**.dart`
- `mobile/lib/features/notification/presentation/widgets/**.dart`
- `mobile/lib/l10n/app_fr.arb` and `app_en.arb` — only NEW keys, prefix `notifications_m19_`

## OFF-LIMITS — DO NOT TOUCH
- Anything under `backend/`
- `web/src/features/messaging/**`, `web/src/features/job/**`, `web/src/features/proposal/**` (other Wave A agents OR Wave B)
- `mobile/lib/features/messaging/**`, `mobile/lib/features/job/**`, `mobile/lib/features/proposal/**`
- `web/src/shared/components/chat-widget/**` (Wave B owns this)
- `web/src/shared/components/layouts/**` (changes here would conflict with everyone)
- All `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/`, `shared/lib/api-client.ts`, `middleware.ts`
- `web/src/features/notification/api/**`, `web/src/features/notification/hooks/**` (data layer locked)
- `mobile/lib/features/notification/data/**`, `mobile/lib/features/notification/domain/**`
- All `package.json`, `pubspec.yaml`, lockfiles, generated l10n `.dart` files
- All existing `*.test.tsx`, `*_test.dart` files

## Acceptance criteria

### Web W-24
- Page header: Fraunces title with italic corail accent (e.g. "Tes *notifications*."), eyebrow "ATELIER · NOTIFICATIONS" font-mono uppercase corail, subtitle 15px muted-foreground.
- Filter pills row (if backend exposes filter dimensions: type / read-state / time): rounded-full pills, ivoire-soft when off / corail-soft when active with corail text. If no filter exists in current data layer, SKIP and FLAG.
- Notification card: ivoire bg, rounded-2xl, border, shadow-card, 18-20px padding. Layout: small icon chip (corail-soft or sapin-soft tone per category) + title (semibold 14.5px) + body (13px tabac) + relative timestamp (Geist Mono mini-pill on the right).
- Read vs unread: unread shows a corail dot (h-2 w-2) at the top-right of the card OR a left corail accent border. Read fades the title (text-muted-foreground 80%).
- Hover: -translate-y-0.5 + border-strong. Click: marks as read (existing handler — read but don't modify behavior).
- Empty state: Soleil card with corail-soft icon plate, Fraunces title "Tu es à jour", body italic Fraunces small "Aucune notification pour le moment.", small CTA "Découvrir le marketplace" (secondary, only if a route exists).

### Mobile M-19
- Same visual rules but Flutter idiom: SoleilTextStyles, AppColors extension, `Card` style aligned with notification mobile widgets.
- AppBar: Fraunces "Notifications" title.
- ListView of notification cards (same anatomy as web).
- Empty state: same Soleil card pattern (icon plate + Fraunces title + tabac body + corail FilledButton if a CTA exists).
- Pull-to-refresh: keep existing behavior.

## Validation pipeline (MANDATORY)

```bash
cd /tmp/mp-notifications

# 1. Scope check
git diff --name-only origin/main...HEAD | grep -E "^(backend/|.*\.test\.|.*_test\.|features/messaging/|features/job/|features/proposal/|shared/components/chat-widget/|shared/components/layouts/)" && echo "OUT-OF-SCOPE TOUCHED" || echo "scoped clean"

# 2. Web
cd web
npm ci
npx tsc --noEmit
npx vitest run src/features/notification src/app/\[locale\]/\(app\)/notifications
npm run build

# 3. Mobile
cd ../mobile
flutter pub get
flutter analyze --no-pub lib/features/notification
flutter test --no-pub test/features/notification/ 2>&1 || echo "(may not exist)"

# 4. Design guardrails
cd ..
bash design/scripts/check-api-untouched.sh
bash design/scripts/check-imports-stable.sh
```

ALL must pass. Fix loop max 3. Blocker file `BLOCKED-notifications.md` if stuck.

## Quality bar (non-negotiable)
- ZERO new hooks, mutations, repositories
- ZERO touch to existing tests
- All user-visible strings via i18n (no hardcoded FR in `.tsx` / `.dart`)
- No `Color(0xFF...)` hardcoded in mobile — use SoleilColors / colorScheme
- ONE squashed commit at the end (use `git reset --soft origin/main` + recommit if multiple WIP)
- DO NOT modify `git config` even temporarily — use `git -c user.email=... -c user.name=... commit ...` per-command if the env identity is wrong

## Push + PR
```bash
git push -u origin feat/design-notifications
gh pr create --title "[design/web/W-24+mobile/M-19] Port Notifications to Soleil v2" --body "<full report>"
```

## Final report (under 700 words)
Standard structure: summary / web changes / mobile changes / out-of-scope flagged / validation pipeline output VERBATIM / brief feedback / visual diffs `design/diffs/W-24-M-19/`.
