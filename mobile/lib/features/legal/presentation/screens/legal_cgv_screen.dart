import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';
import '../widgets/legal_document_screen.dart';

/// `/legal/cgv` — Terms of Sale (Conditions Générales de Vente).
///
/// Mirrors the web `/fr/legal/cgv` page. Body lives in
/// `assets/legal/cgv.md`.
class LegalCgvScreen extends StatelessWidget {
  const LegalCgvScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return LegalDocumentScreen(
      title: l10n.legalDocCgvTitle,
      subtitle: l10n.legalDocCgvSubtitle,
      assetPath: 'assets/legal/cgv.md',
      englishNotice: l10n.legalEnglishNotice,
      lastUpdatedLabel: l10n.legalLastUpdated('2026-05-11'),
    );
  }
}
