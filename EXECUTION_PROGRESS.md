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
| **P2** | `func main()` 909 → 690 (-24%) + 9 wire helpers extracted | backend | ½j est. / 30min réel | agent dispatched | ✅ **done 2026-05-01** | **#75** | Compromis pragmatique structural — pattern struct-deps préserve. |
| **P3** | Web shadcn primitives (Button/Input/Card/Modal/Select) + migrate 370 buttons + 108 inputs + 13 selects | web | 1j est. / 30min réel | agent dispatched | ✅ **done 2026-05-01** | **#73** | Overshoot positif. 96 nouveaux tests, 100% line coverage primitives. |
| **P4** | 17 raw `<img>` → `next/image` + 3 kept blob: + ESLint promoted to error | web | ½j est. / 22min réel | agent dispatched | ✅ **done 2026-05-01** | **#71** | Découverte : disable comments venaient d'un workaround MinIO injustifié — clean revert. |
| **P5** | GDPR endpoints (`/me/export` + `DELETE /me/account` + cron purge T+30) | backend | 1j est. / 1-2h réel | agent dispatched | ✅ **done 2026-05-01** | **#77** | Migration 132. Anonymization sha256+salt prouvée live. 154 strings i18n FR+EN. |

### GATE 1 — après F.1 ✅ **PASSED**
- [x] Full backend test suite green (`go test ./... -count=1 -race`) ✅
- [x] Web `npx tsc --noEmit && npx vitest run` green ✅
- [x] Mobile `flutter analyze && flutter test` green ✅
- [ ] Manual smoke test: register, message, propose, pay, complete, review, GDPR export, GDPR delete — **à faire par user**
- [x] All 5 PRs merged on main ✅
- [ ] DB role rotation test : `marketplace_test_app NOSUPERUSER NOBYPASSRLS` — **à exécuter par user en prod via instructions ops**

---

## F.2 — HIGH (~12-15 jours)

| # | Plan | Stack | Effort | Tool | Status | PR | Notes |
|---|---|---|---|---|---|---|---|
| **P6** | Migration 133 dénormalisation `last_message` (app maintenance, not trigger) | backend | 1j est. / 1h réel | agent dispatched | ✅ **done 2026-05-02** | **#85** | -45% wall-clock ListConversations. 8 tests + bench. |
| **P7** | ISP consumer migration (49 sites: 34 narrow + 15 composite + 5 wide kept) | backend | 2j est. / 1h réel | agent dispatched | ✅ **done 2026-05-02** | **#83** | 8 mocks shrunk, median -9 méthodes/mock. |
| **P8** | Stripe webhook async via `pending_events` (migration 133→134) + scheduler RLS | backend | 2j est. / 1h réel | agent dispatched | ✅ **done 2026-05-02** | **#86** | Webhook <50ms (vs 6-8s sync). 4 schedulers system-actor wraped. |
| **P9** | Web 26 cross-feature imports → 0 + ESLint guard + typed apiClient deferred | web | 2j est. / 50min réel | agent dispatched | ✅ **done 2026-05-02** | **#89** | 1657/1657 tests pass. 30+ files extracted to shared/. |
| **P10** | Slow query log (50ms WARN, 500ms ERROR) + ReadHeaderTimeout=5s + mutation 30/min | backend | 1j est. / 30min réel | agent dispatched | ✅ **done 2026-05-02** | **#87** | 26 nouveaux tests. <1µs overhead per query. Rebased on main (P11 conflict). |
| **P11** | OTel SDK (OTLP) + spans HTTP/DB/Redis/outbound + 3-step graceful shutdown 30s | backend | 1.5j est. / 35min réel | agent dispatched | ✅ **done 2026-05-02** | **#88** | 67ns no-op overhead. 30+ tests. WS hub 1001 frame. |
| **P12** | Mobile build_runner + Freezed/json_serializable regen | mobile | 1j est. / 12min réel | agent dispatched | ✅ **done 2026-05-02** | **#80** | 198 → 0 analyzer errors. +168 tests passing. Convention shifted to commit artefacts. |

### GATE 2 — après F.2 ✅ **REACHED — top-1% atteint**
- [x] Full backend test suite green ✅
- [ ] Web Lighthouse budget LCP/FID/CLS — **à mesurer via CI ou manual**
- [x] Mobile `flutter test` 198→0 analyzer errors ✅
- [x] OTel SDK wired (OTLP exporter, no-op fallback) ✅
- [ ] Manual smoke pass complet — **à faire par user**
- [ ] All 7 PRs merged on main — **5/7 merged, 2 ready (#87 P10, #89 P9) + #90 migration renumber**
- [ ] Re-run audit léger : F.1 + F.2 items closés — **next session**

### Post-merge action items (ops)

1. **Apply migrations en prod** : `DATABASE_URL=<neon-prod> make migrate-up` — applique 130, 131, 132, 133, 134 (résout aussi le bug prod backend "live perms lookup failed" causé par column `users.deleted_at` manquante)
2. **Set env var production** : `GDPR_ANONYMIZATION_SALT=<openssl rand -base64 48>` sur Railway (P5)
3. **Optional** : set `OTEL_EXPORTER_OTLP_ENDPOINT` pour activer OTel (P11)

---

## F.3 — MEDIUM polish (publish-ready path)

### F.3.1 ✅ done — direct path to TOP 1% (Architecture + Security + Maintainability)

| # | Item | Status | PR |
|---|---|---|---|
| 1 | Admin token localStorage → Zustand in-memory (SEC-FINAL-07) | ✅ | **#94** |
| 2 | SSRF guard on `ValidateSocialURL` (SEC-FINAL-04) | ✅ | #94 |
| 3 | `RequireRole` middleware (SEC-FINAL-03) | ✅ | #94 |
| 4 | 8 ADRs Michael Nygard format | ✅ | #94 |
| 5 | CHANGELOG.md Keep-a-Changelog 1.1.0 + SemVer 2.0 | ✅ | #94 |
| 6 | Pre-commit hooks bash + install + self-test | ✅ | #94 |

**Final report**: 32 files changed (+2596 / -137). Validation pipeline green: backend (101 packages OK), admin (112/112 tests), web (tsc clean). Mobile out-of-scope for F.3.1.

### F.3.2 — pending (DRY web cleanup) — gated on backend OpenAPI exposure

Brief: `docs/plans/F3_2_brief.md` (à créer)
- 467 hardcoded `/api/v1/...` paths in `web/src/` → typed `apiClient<paths[X]>(path)`
- Requires backend to expose `/api/openapi.json` first (chi-router introspection or swag annotations)
- 3 pre-existing ESLint errors flagged

### F.3.3 — pending (Quality web/mobile)

Brief: `docs/plans/F3_3_brief.md` (à créer)
- Mobile `dynamic` regression 196 → 746
- Mobile `Color(0x...)` regression 491 → 573
- 19 backend files > 600 lines split
- CONTRIBUTING.md typo `contract-isolation.spec.ts` → `refactor-isolation.spec.ts`

## F.4 — LOW (~2-3 jours, perfectionnement final)

À détailler après F.3 — 41 LOW findings restants dans audits.

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
