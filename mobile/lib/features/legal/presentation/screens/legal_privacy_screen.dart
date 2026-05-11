import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';
import '../widgets/legal_document_screen.dart';

/// `/legal/politique-confidentialite` — long-form GDPR privacy policy.
///
/// Mirrors the web `/fr/legal/politique-confidentialite` page. Body
/// lives in `assets/legal/politique-confidentialite.md`.
class LegalPrivacyScreen extends StatelessWidget {
  const LegalPrivacyScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return LegalDocumentScreen(
      title: l10n.legalDocPrivacyTitle,
      subtitle: l10n.legalDocPrivacySubtitle,
      assetPath: 'assets/legal/politique-confidentialite.md',
      englishNotice: l10n.legalEnglishNotice,
      lastUpdatedLabel: l10n.legalLastUpdated('2026-05-11'),
    );
  }
}
