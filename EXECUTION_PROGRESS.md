# EXECUTION_PROGRESS — Roadmap F.1 + F.2 (top-1% target)

> Tracking séquentiel des plans d'implémentation issus de l'audit `chore/final-audit-deep` (PR #67).
> Ambition : prod-ready ET top-1% mondial sur les 6 axes (architecture, sécurité, perf, scalabilité, DRY, qualité).
> Source de vérité audit : `auditperf.md`, `auditsecurite.md`, `auditqualite.md`, `bugacorriger.md`, `rapportTest.md`, `ROADMAP_FINALE.md`.

---

## Légende statuts

- ⏳ **pending** — pas commencé
- 🔄 **in-progress** — agent en cours
- ⏸️ **blocked** — bloqué par dépendance ou décision externe
- ✅ **done** — mergé sur main
- ❌ **failed** — abandonné, raison documentée

---

## F.1 — CRITICAL (~5 jours, must-close avant prod)

| # | Plan | Stack | Effort | Tool | Status | PR | Notes |
|---|---|---|---|---|---|---|---|
| **P1** | RLS callers migration (38 sites — voir `docs/plans/P1_brief.md`) | backend | 3j est. / **1h réel** | agent dispatched | ✅ **done 2026-05-01** | **#69** | SEC-FINAL-01 fermé. 11/11 integration tests PASS sous NOSUPERUSER NOBYPASSRLS. 10 commits atomiques. |
| **P2** | `func main()` 870 → ≤200 + cleanup helpers extraction | backend | ½j | foreground | ⏳ pending | — | CLAUDE.md violation visible. |
| **P3** | Web shadcn primitives (Button/Input/Card/Modal/Select) + migrate 309 buttons + 95 inputs | web | 1j | agent dispatched | ⏳ pending | — | Asymétrie web vs admin. DRY massif. |
| **P4** | 27 raw `<img>` → `next/image` + LCP hints | web | ½j | agent dispatched | 🔄 in-progress | — | Started 2026-05-01. Regression flagée (était 7). |
| **P5** | GDPR endpoints (`/me/export` + `DELETE /me/account` + cascade tests) | backend | 1j | foreground | ⏳ pending | — | Disqualifying RGPD enterprise EU sans ça. |

### GATE 1 — après F.1
- [ ] Full backend test suite green (`go test ./... -count=1 -race`)
- [ ] Web `npx tsc --noEmit && npx vitest run` green
- [ ] Mobile `flutter analyze && flutter test` green
- [ ] Manual smoke test: register, message, propose, pay, complete, review, GDPR export, GDPR delete
- [ ] All 5 PRs merged on main
- [ ] DB role rotation test : create `marketplace_test_app NOSUPERUSER NOBYPASSRLS`, run smoke against it, verify zero `ErrNotFound` on legitimate reads

---

## F.2 — HIGH (~12-15 jours)

| # | Plan | Stack | Effort | Tool | Status | PR | Notes |
|---|---|---|---|---|---|---|---|
| **P6** | Migration 133 dénormalisation `last_message` sur `conversations` + N+1 elim | backend | 1j | foreground | ⏳ pending | — | Décision : trigger vs app maintenance. |
| **P7** | ISP consumer migration (50+ call sites → segregated interfaces de Phase 3 J) | backend | 2j | `claude -p` | ⏳ pending | — | Mécanique mais scope large. |
| **P8** | Stripe webhook async via `pending_events` worker + scheduler RLS migration | backend | 2j | foreground | ⏳ pending | — | Orchestration à designer. |
| **P9** | Cross-feature imports cleanup web (33 violations) + 96 hardcoded API paths → typed client | web | 2j | foreground | ⏳ pending | — | Refactor architectural. |
| **P10** | Slow queries observability + slowloris DoS guard + mutation rate limit | backend | 1j | foreground | ⏳ pending | — | Infra hardening. |
| **P11** | OpenTelemetry traces + metrics export + graceful shutdown polish | backend | 1.5j | foreground | ⏳ pending | — | Observability foundation. |
| **P12** | Mobile build_runner + subscription DTOs Freezed + 48 broken tests | mobile | 1j | `claude -p` | ⏳ pending | — | Sweep mécanique. |

### GATE 2 — après F.2
- [ ] Full backend test suite green
- [ ] Web Lighthouse budget : LCP < 2.5s, FID < 100ms, CLS < 0.1, JS init < 200KB gzipped
- [ ] Mobile `flutter test` 100% green (les 48 broken tests fix)
- [ ] OTel traces visible sur Jaeger / equivalent
- [ ] Manual smoke pass complet
- [ ] All 7 PRs merged on main
- [ ] Re-run audit léger : F.1 + F.2 items closés sont effectivement closés

---

## F.3 — MEDIUM (~5-7 jours, polish post-launch)

À détailler après F.2 si le user veut continuer le polish. Items dans `auditperf.md`, `auditqualite.md`, `auditsecurite.md` sections MEDIUM.

## F.4 — LOW (~2-3 jours, perfectionnement final)

Idem.

---

## Décisions clés à valider en cours de route

- [ ] **Migration 133** — trigger PostgreSQL vs maintenance applicatif au INSERT messages (P6)
- [ ] **GDPR retention** — quelle durée pour les audit_logs après account deletion ? (P5)
- [ ] **OTel exporter** — Jaeger self-hosted vs Honeycomb vs Datadog free tier ? (P11)
- [ ] **Stripe webhook async** — split idempotency table from RLS scope OR keep webhook handler on privileged role ? (P8)

---

## Out-of-scope (jamais touché)

- LiveKit / call code — fragile, off-limits par décision user (memory `feedback_no_touch_livekit.md`)
- `.github/workflows/*` — token n'a pas le scope `workflow`, manuel via UI GitHub (`/tmp/security.yml.phase6` + `/tmp/ci.yml.phase5.patch` à appliquer)

---

_Maj : 2026-05-01. Ce fichier est la source de vérité statut. Mettre à jour après chaque GATE et après chaque PR mergée._
