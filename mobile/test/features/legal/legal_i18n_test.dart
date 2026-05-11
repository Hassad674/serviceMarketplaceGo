import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

/// Ensures every legal ARB key resolves to a non-empty translation in
/// both supported locales. Catches typos or missing keys before they
/// land in production — running `flutter gen-l10n` does not flag a
/// blank value, only a missing key.
void main() {
  late AppLocalizations fr;
  late AppLocalizations en;

  setUpAll(() async {
    fr = await AppLocalizations.delegate.load(const Locale('fr'));
    en = await AppLocalizations.delegate.load(const Locale('en'));
  });

  // Every text getter we expect to be populated in both locales. The
  // last-updated key takes a `{date}` arg, so it is tested separately.
  Iterable<String Function(AppLocalizations)> staticAccessors() => [
        (l) => l.legalIndexTitle,
        (l) => l.legalIndexIntro,
        (l) => l.legalSectionDocs,
        (l) => l.legalEnglishNotice,
        (l) => l.legalReferenceLabel,
        (l) => l.legalDocRegistreTitle,
        (l) => l.legalDocRegistreSubtitle,
        (l) => l.legalDocRegistreSummary,
        (l) => l.legalDocRegistreReference,
        (l) => l.legalDocAipdTitle,
        (l) => l.legalDocAipdSubtitle,
        (l) => l.legalDocAipdSummary,
        (l) => l.legalDocAipdReference,
        (l) => l.legalDocDpaTitle,
        (l) => l.legalDocDpaSubtitle,
        (l) => l.legalDocDpaSummary,
        (l) => l.legalDocDpaReference,
        (l) => l.legalDocPrivacyTitle,
        (l) => l.legalDocPrivacySubtitle,
        (l) => l.legalDocPrivacySummary,
        (l) => l.legalDocPrivacyReference,
        (l) => l.legalDocCguTitle,
        (l) => l.legalDocCguSubtitle,
        (l) => l.legalDocCguSummary,
        (l) => l.legalDocCguReference,
        (l) => l.legalDocCgvTitle,
        (l) => l.legalDocCgvSubtitle,
        (l) => l.legalDocCgvSummary,
        (l) => l.legalDocCgvReference,
        (l) => l.accountSectionLegal,
        (l) => l.accountSectionLegalDesc,
        (l) => l.accountLegalCta,
      ];

  test('every legal ARB key resolves to non-empty FR + EN strings', () {
    for (final accessor in staticAccessors()) {
      expect(accessor(fr), isNotEmpty);
      expect(accessor(en), isNotEmpty);
    }
  });

  test('legalLastUpdated interpolates the date placeholder', () {
    expect(fr.legalLastUpdated('2026-05-11'), contains('2026-05-11'));
    expect(en.legalLastUpdated('2026-05-11'), contains('2026-05-11'));
  });
}
