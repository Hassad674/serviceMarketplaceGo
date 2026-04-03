# Mobile KYC Dynamic Form + Stripe Notifications — Test Report

## Date: 2026-04-03

## Summary
Two major features implemented in this session:
1. Mobile payment info form refactored from static/hardcoded to fully dynamic (country-aware)
2. Stripe requirements + notifications pipeline fixed and expanded

---

## Part 1: Dynamic Form Tests

### Essential Tests (4/4 PASS)
| # | Country | Type | Status | Notes |
|---|---------|------|--------|-------|
| 1 | FR | Individual | PASS | IBAN bank, standard fields |
| 2 | FR | Business | PASS | Company section, representative prefix |
| 3 | US | Individual | PASS | Routing+account, state dropdown, SSN |
| 4 | US | Business | PASS | Company with state, local bank |

### Supplementary Tests (8/8 PASS)
| # | Country | Type | Status | Notes |
|---|---------|------|--------|-------|
| 5 | DE | Individual | PASS | IBAN bank, no extra fields |
| 6 | DE | Business | PASS | IBAN bank, company section |
| 7 | GB | Individual | PASS | IBAN bank, no extra fields |
| 8 | GB | Business | PASS | IBAN bank, company section |
| 9 | SG | Individual | PASS | Local bank (account+routing), no SSN, no state |
| 10 | SG | Business | PASS | Local bank, company section |
| 11 | IN | Individual | PASS | Local bank, state dropdown (fixed overflow with isExpanded) |
| 12 | IN | Business | PASS | Local bank, company section with state |

### Integrity Tests (7/7 PASS)
| # | Test | Status | Notes |
|---|------|--------|-------|
| 1 | Stripe error from saved entity shows on load | PASS | Entity with stripeError shows red banner |
| 2 | No stripe error when entity has no error | PASS | Clean save shows no error banner |
| 3 | Selecting a country shows dynamic sections | PASS | Country dropdown triggers field loading |
| 4 | Save button exists when country is selected | PASS | Valid form enables save |
| 5 | Save button shown when no country (disabled) | PASS | Button rendered but form invalid |
| 6 | US data with extra_fields persists | PASS | State + SSN in extraFields round-trip |
| 7 | Business data with company fields persists | PASS | Company name visible after reload |

### Total Part 1
- **Essential**: 4/4 PASS
- **Supplementary**: 8/8 PASS
- **Integrity**: 7/7 PASS
- **Overall**: 19/19 PASS (100%)

---

## Part 2: Stripe Requirements + Notifications

### Backend Unit Tests
| Test Suite | Status | Notes |
|-----------|--------|-------|
| `go test ./internal/domain/notification/...` | PASS | 2 new types validated + email defaults |
| `go test ./internal/domain/payment/...` | PASS | AccountRequirements struct |
| `go test ./internal/app/payment/...` (39 tests) | PASS | All requirements + status tests |
| `go build ./...` | PASS | Full compilation |

### Notification Pipeline Fix
- **Before**: All `stripe_requirements` notifications failed with "invalid notification type"
- **After**: Notifications send via in-app (WebSocket) + push (FCM) + email (Resend)
- Backend logs show no more "invalid notification type" errors

### Regression (all prior tests still pass after changes)
| Suite | Result |
|-------|--------|
| Essential (FR/US) | 4/4 PASS |
| Supplementary (DE/GB/SG/IN) | 8/8 PASS |
| Integrity | 7/7 PASS |

---

## Architecture Changes

### Mobile Dynamic Form
- Entity: added `country` and `extraFields` fields
- New: `form_data_mapper.dart` — path-key mapping (responseToFormData, valuesToFlatData, isFormValid)
- New: `dynamic_section.dart` — renders any FieldSection from API
- Activated dead code: CountrySelectorSection, ExtraFieldsSection, countryFieldsProvider
- Screen: refactored from 665→320 lines, hardcoded sections replaced with dynamic rendering

### Backend — Stripe Error Surfacing
- `updateExternalAccount`: fixed delete-then-create → create-then-delete order
- `UpdateConnectedAccount`: now returns bank account errors instead of swallowing them
- Bank comparison skips update when details haven't changed

### Backend — Notifications
- Added `TypeStripeRequirements` + `TypeStripeAccountStatus` notification types (email=ON)
- `GetAccountRequirements` returns full struct: currently_due, eventually_due, past_due, pending_verification, current_deadline
- Requirements endpoint merges all 3 lists with deduplication and urgency tagging
- `HandleAccountUpdated` detects charges_enabled/payouts_enabled changes → sends notification
- Migration 032: added charges_enabled + payouts_enabled columns

### Frontend — Urgency Banners
- Web: red banner for currently_due/past_due, amber for eventually_due, deadline display
- Mobile: same urgency color coding + deadline display
