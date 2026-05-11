import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';
import '../widgets/legal_document_screen.dart';

/// `/legal/cgu` — Terms of Use (Conditions Générales d'Utilisation).
///
/// Mirrors the web `/fr/legal/cgu` page. Body lives in
/// `assets/legal/cgu.md`.
class LegalCguScreen extends StatelessWidget {
  const LegalCguScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return LegalDocumentScreen(
      title: l10n.legalDocCguTitle,
      subtitle: l10n.legalDocCguSubtitle,
      assetPath: 'assets/legal/cgu.md',
      englishNotice: l10n.legalEnglishNotice,
      lastUpdatedLabel: l10n.legalLastUpdated('2026-05-11'),
    );
  }
}
