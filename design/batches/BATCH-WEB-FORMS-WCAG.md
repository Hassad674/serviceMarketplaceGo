# BATCH-WEB-FORMS-WCAG — forms primitives + WCAG body/CTA contrast + .gitattributes

> Worktree: `/tmp/mp-web-forms-wcag` · Branch: `fix/web-forms-wcag` · Base: `origin/main` (48b0680e)

## Goal — 3 cleanup tasks, ONE squashed commit

### Task 1 — Forms primitives migration (kills 95 ESLint react/forbid-elements errors)
The audit identified 8 web forms still using raw `<input>` / `<button>` JSX elements instead of the design system primitives `<Input>` / `<Button>`. ESLint rule `react/forbid-elements` flags 95 errors across these forms.

Forms identified:
- `web/src/features/auth/components/login-form.tsx`
- `web/src/features/auth/components/agency-register-form.tsx`
- `web/src/features/auth/components/enterprise-register-form.tsx`
- `web/src/features/auth/components/provider-register-form.tsx`
- `web/src/features/auth/components/forgot-password-form.tsx`
- `web/src/features/auth/components/reset-password-form.tsx`
- `web/src/features/job/components/create-job-form.tsx`
- `web/src/features/account/components/notification-settings.tsx`

Action: replace raw `<input>` → `<Input>` and `<button type="submit">` → `<Button type="submit">`. Forms keep their `react-hook-form` + `zod` wiring untouched. Visual styling already on Soleil — just swap the JSX element.

If a form uses a custom Toggle that's a plain `<button>` (e.g. notification-settings already migrated to plain `<button>` in PR #126), leave it — it's a deliberate primitive, not a leaked `<button>`. The ESLint rule should allow primitives. If it doesn't, add an `eslint-disable-next-line` with a "// custom toggle, primitive" comment.

Goal: `cd web && npx eslint src/features/auth src/features/job src/features/account 2>&1 | grep "react/forbid-elements"` returns 0 errors.

### Task 2 — WCAG AA contrast — body links + CTA buttons
Audit failures:
- corail #e85d4a on ivoire = 3.34:1 (Large Text only — fails for body 14-16px)
- white on corail (button) = 3.45:1 (fails AA 4.5:1)
- sable #a89679 on ivoire = 2.79:1 (FAIL for "subtle/mono labels" usage)

Action — token-level fix in `web/src/styles/globals.css`:

1. **Body links / inline accent text**: where the design uses corail for body-size links (e.g. "Se connecter" footer links), switch to `--primary-deep` (#c43a26) which gives **4.92:1** on ivoire — passes AA. Add a semantic alias if helpful: `--text-link: var(--primary-deep)` and update `.tsx` consumers using `text-primary` for links → `text-[var(--primary-deep)]` or new `text-link` utility.

2. **CTA buttons (white on corail)**: two options:
   - **Option A (visual conservative)**: use `bg-primary-deep` (#c43a26) for primary buttons → white on corail-deep = 5.83:1 ✅ AA pass. Slightly darker visual identity.
   - **Option B (accept marginal)**: keep current `bg-primary` but ensure all CTA labels are bold (`font-semibold`) which slightly bumps contrast perception. Document the deliberate trade-off in a comment in globals.css.
   
   Pick A — it's the cleaner fix. Update `.button` primary variant in `web/src/shared/components/ui/button.tsx` if it doesn't already use the `--primary` token (it should, since PR #144). The token swap is at the `--primary` definition? NO — keep `--primary` as #e85d4a for soft accents, but switch the primary BUTTON BG to `var(--primary-deep)`. Read button.tsx, find the variant, swap the bg class.

3. **Sable subtle labels**: where the design uses `text-[var(--sable)]` or similar for labels, switch to `text-muted-foreground` (which is `--tabac` = #7a6850 on ivoire = 4.65:1 ✅ AA) or `text-foreground/70` (encre at 70% opacity).

Goal: every body-text and CTA label passes AA (4.5:1) on its bg.

NB: Don't touch the corail token DEFINITION (#e85d4a) — that's the brand color, used for non-text accents (icon discs on -soft bg, cards, dividers, illustrations). Only the TEXT usages on white/ivoire need to switch to corail-deep.

### Task 3 — `.gitattributes` for design assets vendor classification
GitHub Linguist counts ~13,500 LOC of `design/assets/sources/*.jsx` as JavaScript, inflating the JS percentage in repo stats. Mark them as vendored docs.

Action: create or update `.gitattributes` at repo root:
```
design/assets/sources/** linguist-vendored=true
design/assets/pdf/** linguist-vendored=true
*.freezed.dart linguist-generated=true
*.g.dart linguist-generated=true
```

The `linguist-vendored=true` tells GitHub these are reference assets, not production code — they're excluded from language stats while staying in the repo. Same trick for Flutter generated files (`*.freezed.dart` / `*.g.dart`).

Goal: GitHub will recompute language stats and the JS % drops to its true value (a few config files only).

## TOUCHABLE files

### Task 1 (forms)
- 8 form files listed above

### Task 2 (WCAG)
- `web/src/styles/globals.css` (token additions: `--text-link` if you add it)
- `web/src/shared/components/ui/button.tsx` (CTA bg → primary-deep)
- 8 forms (CTA classes if they hardcode bg-primary instead of using Button primitive)
- Any other consumer of `text-primary` that's body-size text — but be SURGICAL, don't blanket-replace

### Task 3 (gitattributes)
- `.gitattributes` at repo root (NEW or update)

## OFF-LIMITS — STRICT
- ALL mobile files
- ALL admin files
- Backend
- `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/`
- `package.json`, `pubspec.yaml`, lockfiles
- All other web `features/*/components/*` files (sibling F3 owns the widget rose cleanup)
- `web/src/shared/components/ui/{input,select,badge,card,...}.tsx` (only `button.tsx` for the WCAG CTA bg)
- All existing tests UNLESS the test pins a literal `bg-primary`/`text-primary`/`bg-rose-500` string AND the form's primitive migration changed that — only update the literal class string in that case, never the test SEMANTICS

## Acceptance criteria
- `cd web && npx eslint src/features/auth src/features/job src/features/account` → 0 `react/forbid-elements` errors (down from ~95)
- `npx tsc --noEmit` clean
- `npx vitest run src/features/auth src/features/job src/features/account src/shared/components/ui` → all pass
- `npm run build` succeeds
- All body-text/links/CTA labels pass AA contrast (eyeball check on dev server + a manual check via browser DevTools or a contrast checker tool)
- `.gitattributes` written, committed
- `git check-attr linguist-vendored design/assets/sources/screens-editorial.jsx` returns "linguist-vendored: true"

## Validation pipeline (MANDATORY)

```bash
cd /tmp/mp-web-forms-wcag

# 1. Scope check
git diff --name-only origin/main...HEAD | grep -E "^(backend/|mobile/|admin/|web/src/features/(?!auth|job/components/create-job-form|account/components/notification-settings)/|web/src/shared/components/ui/(?!button\.tsx)|web/(?!src/styles/globals\.css|src/features/auth|src/features/job/components/create-job-form|src/features/account/components/notification-settings|src/shared/components/ui/button\.tsx)|package\.json|pubspec)" | head

# 2. ESLint forbid-elements regression check
cd web
npm ci
npx eslint src/features/auth src/features/job src/features/account 2>&1 | grep "react/forbid-elements" | wc -l
# Should be 0

# 3. Web validation
npx tsc --noEmit
npx vitest run src/features/auth src/features/job src/features/account src/shared/components/ui
npm run build

# 4. .gitattributes verification
cd ..
git check-attr linguist-vendored design/assets/sources/screens-editorial.jsx
# Should return: linguist-vendored: true

# 5. Design guardrails
bash design/scripts/check-api-untouched.sh
bash design/scripts/check-imports-stable.sh
```

ALL must pass. Fix loop max 3.

## Quality bar
- ZERO new hooks/mutations
- ZERO change to form behavior (react-hook-form + zod logic UNCHANGED)
- ZERO touch to existing test SEMANTICS
- The 8 forms diffs should be tight (mostly element replacement + class adjustment)
- ONE squashed commit
- DO NOT modify `git config` — use per-command `-c user.email=...`

## Push + PR
- Message: `fix(design/web): forms → DS primitives + WCAG AA contrast + .gitattributes vendor classification`
- PR title: `[fix/web-forms-wcag] Forms primitives migration + WCAG AA + JS % cleanup`

## Final report (under 600 words)
Standard structure + EMPHASIZE: ESLint error count (before/after), WCAG decisions made (corail-deep for buttons), Linguist impact (95 ESLint errors → 0; 5% JS → expected ~0.5% after Linguist recomputes).
