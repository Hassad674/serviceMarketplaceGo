# BATCH-KYC — W-05 Stripe Connect KYC visual port

> Worktree: `/tmp/mp-kyc` · Branch: `feat/design-kyc-visual` · Base: `origin/main` (12af167d)

## Goal — VISUAL ONLY (CRITICAL)
Port the KYC / Stripe Connect onboarding pages to Soleil v2 — **VISUAL identity only**. The user explicitly said: "faut juste l'identité visuel faut pas toucher au system de casser stripe embeded". DO NOT touch Stripe Embedded Components logic, the iframe wiring, or the API integration. Restyle the SHELL around it.

## TOUCHABLE files

### Web
- `web/src/app/[locale]/(app)/payment-info/page.tsx` — the page hosting Stripe Connect onboarding
- `web/src/shared/components/kyc-banner.tsx` (the nudge banner shown across the dashboard)
- `web/src/features/wallet/components/kyc-incomplete-modal.tsx` (modal — possibly already Soleil from W-18, verify)
- `web/messages/fr.json` and `en.json` — NEW keys ONLY, prefix `kyc_w05_*`

### Mobile
- `mobile/lib/features/payment_info/presentation/screens/payment_info_screen.dart`
- Sub-widgets in `mobile/lib/features/payment_info/presentation/widgets/` if any (read first)
- `mobile/lib/l10n/app_fr.arb` and `app_en.arb` — NEW keys ONLY, prefix `kyc_w05_*`

## OFF-LIMITS — STRICT
- **Stripe Embedded Components** wiring — the `<stripe-connect-...>` web components or any equivalent mobile iframe / WebView. DO NOT change behavior, props, callbacks, mount/unmount logic.
- **Stripe Connect API** calls — anywhere they happen, leave untouched.
- All `*/api/*.ts`, `*/hooks/use-*.ts`, `*/schemas/`
- `web/src/features/wallet/api/**` and `hooks/**`
- `mobile/lib/features/payment_info/data/**`, `mobile/lib/features/payment_info/domain/**`
- All `web/src/features/wallet/**` EXCEPT `kyc-incomplete-modal.tsx` (and only if it needs polish)
- All `web/src/features/account/**`, `web/src/features/team/**`, `web/src/features/messaging/**`, `web/src/features/proposal/**`, `web/src/features/invoicing/**`, `web/src/features/billing/**`, `web/src/features/job/**`, `web/src/features/notification/**`, `web/src/features/freelance-profile/**` (Soleil already merged elsewhere — locked)
- `package.json`, `pubspec.yaml`, lockfiles, generated l10n
- All existing `*.test.tsx`, `*_test.dart`
- Anything under `backend/`
- Sibling agents A2 (team+search), A3 (mobile dashboards), A4 (mobile invoicing)

## Acceptance criteria

### Page shell `/payment-info` (web)
- Editorial header: corail mono eyebrow "ATELIER · IDENTITÉ FISCALE" or "ATELIER · KYC", Fraunces title with italic corail accent (e.g. "Vérifie ton *identité fiscale*."), tabac subtitle explaining why this is needed
- Step indicator pills if existing UX has steps — Soleil pill (corail-soft active / ivoire off)
- Stripe iframe / Embedded mount: WRAP it inside a Soleil card (`bg-card rounded-2xl border border-border shadow-card padding`) but DO NOT touch the iframe component itself
- Status messages around the iframe (loading / error / completed) — Soleil styling
- Success / error banners — Soleil tokens (success-soft / corail-soft / amber-soft)

### `kyc-banner.tsx`
- Restyle as Soleil banner: amber-soft bg, Fraunces title, tabac body, corail pill CTA "Compléter mon identité"

### `kyc-incomplete-modal.tsx`
- Verify it's already Soleil (was touched in W-18). If still legacy, port. Otherwise leave alone.

### Mobile `payment_info_screen.dart`
- Soleil AppBar: Fraunces title
- Editorial header: corail eyebrow + Fraunces italic-corail title
- WebView / iframe wrapper inside Soleil card if applicable
- Status banners with SoleilTextStyles + AppColors
- DO NOT touch the WebView controller, navigation handlers, postMessage logic

## Validation pipeline (MANDATORY)

```bash
cd /tmp/mp-kyc
git diff --name-only origin/main...HEAD | grep -E "^(backend/|.*\.test\.|.*_test\.|features/(team|messaging|proposal|invoicing|billing|job|notification|freelance-profile|account)/|app/\[locale\]/\((auth|app)/(team|messages|projects|invoices|jobs|notifications|profile|opportunities|account|billing)\)|mobile/lib/features/(team|messaging|proposal|invoicing|billing|job|notification|freelance_profile|account|dashboard)/)" && echo "OUT-OF-SCOPE TOUCHED" || echo "scoped clean"
cd web && npm ci && npx tsc --noEmit && npx vitest run src/features/wallet src/app/\[locale\]/\(app\)/payment-info && npm run build
cd ../mobile && flutter pub get && flutter analyze --no-pub lib/features/payment_info && flutter test --no-pub test/features/payment_info/ 2>&1 || echo "(may not exist)"
cd .. && bash design/scripts/check-api-untouched.sh && bash design/scripts/check-imports-stable.sh
```

ALL must pass. Fix loop max 3.

## Quality bar
- Behavior preservation is CRITICAL — Stripe Embedded must keep working identically post-port
- ZERO new hooks/mutations/repositories
- ZERO touch to existing tests
- ONE squashed commit, no `git config` mutation

## Push + PR
- Message: `feat(design/kyc): visual port of Stripe Connect / KYC pages to Soleil v2`
- PR title: `[design/web/W-05+mobile] KYC visual port to Soleil v2 (Stripe untouched)`

## Final report (under 500 words)
Standard structure + EMPHASIZE that Stripe Embedded behavior was not modified.
