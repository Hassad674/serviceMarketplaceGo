# Mobile KYC Dynamic Form — Test Report

## Date: 2026-04-03

## Summary
The mobile payment info form was refactored from a static hardcoded form to a fully dynamic, country-aware form matching the web implementation.

## Essential Tests (4/4 PASS)
| # | Country | Type | Status | Notes |
|---|---------|------|--------|-------|
| 1 | FR | Individual | PASS | IBAN bank, standard fields |
| 2 | FR | Business | PASS | Company section, representative prefix |
| 3 | US | Individual | PASS | Routing+account, state dropdown, SSN |
| 4 | US | Business | PASS | Company with state, local bank |

## Supplementary Tests (8 tests)
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

## Integrity Tests (7/7 PASS)
| # | Test | Status | Notes |
|---|------|--------|-------|
| 1 | Stripe error from saved entity shows on load | PASS | Entity with stripeError shows red banner |
| 2 | No stripe error when entity has no error | PASS | Clean save shows no error banner |
| 3 | Selecting a country shows dynamic sections | PASS | Country dropdown triggers field loading |
| 4 | Save button exists when country is selected | PASS | Valid form enables save |
| 5 | Save button shown when no country (disabled) | PASS | Button rendered but form invalid |
| 6 | US data with extra_fields persists | PASS | State + SSN in extraFields round-trip |
| 7 | Business data with company fields persists | PASS | Company name visible after reload |

## Architecture Changes
- Entity: added `country` and `extraFields` fields
- New: `form_data_mapper.dart` — path-key mapping (responseToFormData, valuesToFlatData, isFormValid)
- New: `dynamic_section.dart` — renders any FieldSection from API
- Activated dead code: CountrySelectorSection, ExtraFieldsSection, countryFieldsProvider
- Screen: refactored from 665->320 lines, hardcoded sections replaced with dynamic rendering

## Total Results
- **Essential**: 4/4 PASS
- **Supplementary**: 8/8 PASS
- **Integrity**: 7/7 PASS
- **Overall**: 19/19 PASS (100%)

## Backend Fix (same session)
- `updateExternalAccount`: fixed delete-then-create -> create-then-delete order
- `UpdateConnectedAccount`: now returns bank account errors instead of swallowing them
- Bank comparison skips update when details haven't changed
