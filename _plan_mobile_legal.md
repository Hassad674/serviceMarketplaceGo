# Plan — Mobile parity for `legal/*` (web D4)

Brief: parity for the 7 web routes shipped under `/fr/legal/*` (sommaire +
6 long-form documents) on the Flutter mobile app. Ni plus, ni moins.

## Asset list

The 6 canonical markdown documents from `/legal/` are copied verbatim
into `mobile/assets/legal/`:

| Asset path | Source |
|---|---|
| `assets/legal/registre.md` | `legal/registre.md` |
| `assets/legal/aipd.md` | `legal/aipd.md` |
| `assets/legal/dpa-template.md` | `legal/dpa-template.md` |
| `assets/legal/politique-confidentialite.md` | `legal/politique-confidentialite.md` |
| `assets/legal/cgu.md` | `legal/cgu.md` |
| `assets/legal/cgv.md` | `legal/cgv.md` |

The source markdown files in `/legal/` are NOT modified. The mobile
assets are a copy — keeping them in `mobile/assets/legal/` decouples
the mobile bundle from repo layout changes, and lets the Flutter build
system bundle them through `rootBundle`.

Registered in `pubspec.yaml` under `flutter.assets:` as `assets/legal/`.

## Package decision

Need a markdown renderer. The mobile app does not currently render
markdown. The de-facto choice in the Flutter ecosystem is
`flutter_markdown`. Reviewed alternatives:

- `flutter_markdown` — official `flutter/packages` repo, no other dep,
  exposes `MarkdownBody` widget + `MarkdownStyleSheet` for theming.
- `markdown_widget` — community pkg, more features but heavier API and
  a wider surface than we need.
- Custom renderer — overkill for static long-form docs.

**Decision**: add `flutter_markdown: 0.7.7+1` (exact pin, no caret).
This is the last release that fully supports Flutter 3.41 / Dart 3.11
and is the standard the rest of the Soleil v2 web markdown content
mirrors. Justification documented inline in `pubspec.yaml`.

## Widget structure

```
lib/features/legal/
└── presentation/
    ├── screens/
    │   ├── legal_index_screen.dart           ~120 LOC  (sommaire + 6 cards)
    │   ├── legal_registre_screen.dart        ~12 LOC   (delegates to shared widget)
    │   ├── legal_aipd_screen.dart            ~12 LOC
    │   ├── legal_dpa_template_screen.dart    ~12 LOC
    │   ├── legal_privacy_screen.dart         ~12 LOC
    │   ├── legal_cgu_screen.dart             ~12 LOC
    │   └── legal_cgv_screen.dart             ~12 LOC
    └── widgets/
        └── legal_document_screen.dart        ~220 LOC  (shared scaffolding)
```

The 6 detail screens are 1-line wrappers around `LegalDocumentScreen`,
passing (titleKey, subtitleKey, assetPath, sourceUrl).

`LegalDocumentScreen` is responsible for:
- AppBar (with title from i18n).
- Loading the markdown asset via `rootBundle.loadString`.
- Wrapping the `MarkdownBody` with Soleil v2 typography (Fraunces for
  headings, Inter Tight for body, accent for links).
- Showing the "English version available on request" banner at top.
- Showing a "Last updated" footer.
- Async error state (returns a small inline error if the asset can't
  be loaded — should never happen in prod).

## i18n keys (mirror web `legal.docs.*`)

New ARB keys (FR + EN identical structure):

```
legalIndexTitle                                "Documents légaux"
legalIndexIntro                                "Documents publiés à des fins…"
legalSectionDocs                               "Documents disponibles"
legalEnglishNotice                             "Version anglaise complète sur demande…"
legalLastUpdated                               "Dernière mise à jour : {date}"
legalDocRegistreTitle                          "Registre des activités de traitement"
legalDocRegistreSummary                        "11 traitements documentés…"
legalDocAipdTitle                              "Analyse d'impact (AIPD)"
legalDocAipdSummary                            "Trois AIPD couvrant…"
legalDocDpaTitle                               "Modèle de contrat de sous-traitance (DPA)"
legalDocDpaSummary                             "Modèle générique conforme…"
legalDocPrivacyTitle                           "Politique de confidentialité (version longue)"
legalDocPrivacySummary                         "Version étendue avec tableaux…"
legalDocCguTitle                               "Conditions Générales d'Utilisation"
legalDocCguSummary                             "17 articles couvrant inscription…"
legalDocCgvTitle                               "Conditions Générales de Vente"
legalDocCgvSummary                             "15 articles couvrant modèle…"
accountSectionLegal                            "Mentions légales"
accountSectionLegalDesc                        "RGPD, CGU, CGV…"
accountLegalCta                                "Lire les documents"
```

Last-updated date is a static `2026-05-11` (same as web's
`docs.lastUpdatedISO` — the documents shipped together D4).

EN strings translate titles and CTA labels; the markdown body itself
stays in French (matches web behavior — see `englishNotice`).

## Routes

Add to `core/router/routes/team_routes.dart` (membership shell routes):
- `/legal` — index
- `/legal/registre`
- `/legal/aipd`
- `/legal/dpa-template`
- `/legal/politique-confidentialite`
- `/legal/cgu`
- `/legal/cgv`

`RoutePaths` constants in `app_router.dart`.

## Navigation entry point

A new `_AccountSection` is added at the bottom of `AccountScreen` — a
"Mentions légales" tile with `Icons.gavel_outlined`, description, and
an `OutlinedButton` linking to `/legal`. No bottom-nav change. No
drawer change.

## Test plan (target ≥ 90% coverage on `lib/features/legal/`)

| Test file | Cases |
|---|---|
| `legal_document_screen_test.dart` | 5 cases: (1) renders title, (2) renders subtitle, (3) renders english-notice banner, (4) renders last-updated footer, (5) renders the markdown body after async load. |
| `legal_index_screen_test.dart` | 3 cases: (1) all 6 cards visible, (2) tapping each fires a router push to the right path (table-driven), (3) tapping back returns to caller. |
| `legal_registre_screen_test.dart` | 1 case: contains a known FR fragment from `registre.md`. |
| `legal_aipd_screen_test.dart` | 1 case: same. |
| `legal_dpa_template_screen_test.dart` | 1 case: same. |
| `legal_privacy_screen_test.dart` | 1 case: same. |
| `legal_cgu_screen_test.dart` | 1 case: same. |
| `legal_cgv_screen_test.dart` | 1 case: same. |
| `legal_assets_test.dart` | 1 case: every of the 6 asset paths loads as a String of length > 1000 (catches missing pubspec entries). |
| `legal_i18n_test.dart` | 1 case: every new ARB key resolves in both FR and EN; titles/summaries are non-empty. |

Total: ~15 widget/unit tests.

## Commits sequence

1. **plan** — this file.
2. **assets + pubspec** — copy 6 markdown files into `mobile/assets/legal/`,
   register asset folder + pin `flutter_markdown` in pubspec.yaml.
3. **shared widget** — `LegalDocumentScreen` widget.
4. **screens + i18n** — 7 screen files + ARB additions in both locales,
   regenerated localizations.
5. **routing + nav** — `RoutePaths` constants, `team_routes.dart`
   bindings, account-screen tile.
6. **tests** — the 10 test files described above.

First commit (this plan) ≤ 10 tool uses.
