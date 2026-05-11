import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';
import '../widgets/legal_document_screen.dart';

/// `/legal/dpa-template` — RGPD art. 28 sub-processor contract template.
///
/// Mirrors the web `/fr/legal/dpa-template` page. Body lives in
/// `assets/legal/dpa-template.md`.
class LegalDpaTemplateScreen extends StatelessWidget {
  const LegalDpaTemplateScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return LegalDocumentScreen(
      title: l10n.legalDocDpaTitle,
      subtitle: l10n.legalDocDpaSubtitle,
      assetPath: 'assets/legal/dpa-template.md',
      englishNotice: l10n.legalEnglishNotice,
      lastUpdatedLabel: l10n.legalLastUpdated('2026-05-11'),
    );
  }
}
