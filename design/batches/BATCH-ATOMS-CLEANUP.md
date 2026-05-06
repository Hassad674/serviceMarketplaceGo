# BATCH-ATOMS-CLEANUP — fix critical Soleil v2 atoms

> Worktree: `/tmp/mp-atoms-cleanup` · Branch: `fix/design-atoms-primitives` · Base: `origin/main` (49650494)

## Context — independent V6 audit findings

An independent Claude audit identified 3 CRITICAL Soleil v2 leakages that survived the entire 30+ PR design chantier because they live in shared atoms / config files outside the per-screen wave scopes:

- **C1 — Web shared primitives still rose-500**: `web/src/shared/components/ui/{button,input,select}.tsx` use `focus:ring-rose-500`, `focus:border-rose-500`, `border-slate-200`, `gradient-primary`, `shadow-glow` (legacy Rose Contra). Every Soleil page using these primitives shows rose focus rings on corail backgrounds. `button.tsx`'s doc-comment header (lines 7-30) literally describes the spec as "rose primary gradient + shadow-glow on hover" — a future agent reading this will continue shipping Rose.
- **C2 — Mobile bottom-nav badge rose**: `mobile/lib/core/router/app_router.dart:400,413` uses `AppPalette.rose500` for the unread message badge visible on every authenticated screen.
- **C3 — Admin fonts not loaded**: `admin/index.html` has zero `<link>` / `<script>` / `@import` for fonts. CSS declares `--font-sans: 'Inter Tight'` but the browser has no resolver — admin renders in `system-ui`. Web loads via `next/font/google` — admin is silent. Cross-app cohérence is broken.

## Goal — ONE squashed commit
Fix all 3 critical atoms findings. No new features. Pure surgical primitive corrections.

## TOUCHABLE files (exhaustive)

### Web — primitive atoms
- `web/src/shared/components/ui/button.tsx` (variant classes + base focus ring + doc comment)
- `web/src/shared/components/ui/input.tsx` (focus border + ring + slate borders)
- `web/src/shared/components/ui/select.tsx` (focus border + ring + slate borders)

### Mobile — bottom-nav badge
- `mobile/lib/core/router/app_router.dart` (badge backgroundColor on lines ~400 + ~413)

### Admin — font loading
- `admin/index.html` (add `<link>` to Google Fonts for Inter Tight + Fraunces + Geist Mono OR add `@fontsource/*` imports in main.tsx — pick one, document in commit)
- `admin/src/main.tsx` IF you go the fontsource route (read first)
- `admin/package.json` IF fontsource is the choice (this is the ONLY exception to the "no package.json" rule for this batch — fonts are explicitly in scope)
- `admin/src/styles/globals.css` IF the `--font-sans` declaration needs adjustment (read first)

### Tests
- READ `web/src/shared/components/ui/__tests__/{button,input,select}.test.tsx` carefully — they may pin specific class names. If they expect literal `rose-500` strings, the test will fail after the Soleil migration. In that case, the test was actively pinning the legacy state and it's OK to update the literal expectation BUT only the literal class string (e.g. `bg-rose-500` → `bg-primary`). Do NOT change test structure or assertions semantics.

## OFF-LIMITS — STRICT
- ALL other web files (not in shared/components/ui/{button,input,select}.tsx)
- ALL other mobile files (not app_router.dart)
- `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/`, `shared/lib/api-client.ts`, `middleware.ts`
- `web/package.json`, `mobile/pubspec.yaml`, `admin/package.json` IF you don't go the fontsource route
- Backend
- Existing tests UNLESS they pin a literal `rose-500` / `rose-600` / `slate-200` class string that needs updating to `primary` / `border` — and only that string
- `mobile/lib/core/theme/app_palette.dart` (don't deprecate AppPalette — that's a separate larger work)
- All other shared atoms in `web/src/shared/components/ui/` (badge, card, modal, skeleton-block, portrait, etc. — only button/input/select in scope)

## Acceptance criteria

### Button (`button.tsx`)
- Update the doc-comment header at lines 7-30: drop "rose primary gradient", "shadow-glow on hover" — replace with Soleil v2 description (corail primary, calm shadow-card, ivoire/sable-foncé borders for outline/secondary variants)
- Base classes (line 35): `focus-visible:ring-rose-500/50` → `focus-visible:ring-primary/30`
- `primary` variant (line 42): `gradient-primary text-white shadow-sm hover:shadow-glow active:scale-[0.98]` → `bg-primary text-white shadow-sm hover:bg-primary/90 active:scale-[0.98]` (or keep gradient-primary if it already maps to corail in globals.css — verify, then keep `shadow-card` instead of `shadow-glow`)
- `secondary` variant: replace slate-* with semantic Soleil tokens (`bg-secondary text-secondary-foreground hover:bg-secondary/80` or similar — read globals.css for the right tokens)
- `outline` variant (line 46): `border-slate-200 bg-white text-slate-900 hover:bg-slate-50 hover:border-rose-200` → `border-border bg-card text-foreground hover:bg-muted hover:border-border-strong`
- `ghost` variant: same migration to Soleil tokens
- `destructive` variant: keep using destructive tokens but verify they exist (`bg-destructive text-destructive-foreground` or similar Soleil-tinted)

### Input (`input.tsx`) and Select (`select.tsx`)
- `border-slate-200` → `border-border` (or `border-border-strong` for stronger contrast on the resting state)
- `focus:border-rose-500` → `focus:border-primary`
- `focus:ring-rose-500/10` → `focus:ring-primary/15`
- Dark mode `dark:border-slate-700` → use semantic dark token if available, else leave the dark variant alone (Soleil dark tokens already inherit via [data-theme="dark"] block in globals.css)
- Doc comment in input.tsx line 11 — update to drop "ring-rose-500/10" and describe Soleil corail focus

### Mobile (`app_router.dart`)
- Lines 400 + 413: `backgroundColor: AppPalette.rose500` → `backgroundColor: Theme.of(context).colorScheme.primary`
- Verify the `BuildContext` is in scope at those lines. If not, the badge widget probably has `context` as parameter — pass it through.

### Admin fonts
Choose ONE of:
- **Option A — Google Fonts CDN** (simplest): add to `admin/index.html` `<head>`:
  ```html
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Fraunces:ital,wght@0,400;0,500;1,400;1,500&family=Inter+Tight:wght@400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
  ```
- **Option B — @fontsource** (offline-capable, larger bundle): add `@fontsource/inter-tight`, `@fontsource/fraunces`, `@fontsource/geist-mono` to admin/package.json, import in admin/src/main.tsx with `import '@fontsource/inter-tight/index.css'` etc.

Pick A (simpler, matches the spirit of "kill leakage with minimal config change") UNLESS the project already uses @fontsource somewhere — then go B for consistency. Document the choice in the commit message.

## Validation pipeline (MANDATORY)

```bash
cd /tmp/mp-atoms-cleanup

# 1. Scope check
git diff --name-only origin/main...HEAD | grep -E "^(backend/|.*\.test\.|.*_test\.|mobile/lib/(?!core/router/app_router\.dart)|web/src/(?!shared/components/ui/(button|input|select)\.tsx)|web/messages/|admin/(?!(index\.html|src/main\.tsx|package\.json|src/styles/globals\.css)))" | head
# Verify no unexpected paths

# 2. Web
cd web
npm ci
npx tsc --noEmit
npx vitest run src/shared/components/ui
npm run build

# 3. Admin
cd ../admin
npm ci
npx tsc --noEmit
# admin uses vitest? check package.json scripts. If yes:
npx vitest run 2>&1 | tail -15 || echo "(admin may have no vitest setup)"
npm run build

# 4. Mobile
cd ../mobile
flutter pub get
flutter analyze --no-pub lib/core/router
flutter test --no-pub test/core/router/ 2>&1 | tail -10 || echo "(may not exist)"

# 5. Design guardrails
cd ..
bash design/scripts/check-api-untouched.sh
bash design/scripts/check-imports-stable.sh
```

ALL must pass. Fix loop max 3.

## Quality bar
- ZERO new hooks/mutations
- ZERO touch to existing test SEMANTICS (only literal class strings if they pin legacy colors)
- The 3 atoms file diffs should be tight (~40 LOC web combined, ~3 LOC mobile, ~6 LOC admin)
- ONE squashed commit
- DO NOT modify `git config` — use per-command `-c user.email=...`

## Push + PR
- Message: `fix(design): port shared atoms (button/input/select web + mobile badge + admin fonts) to Soleil v2`
- PR title: `[fix/design-atoms] Port shared primitives to Soleil v2 (closes V6 critical findings)`

## Final report (under 500 words)
Standard structure + EMPHASIZE that the 3 CRITICAL audit findings are now closed.
