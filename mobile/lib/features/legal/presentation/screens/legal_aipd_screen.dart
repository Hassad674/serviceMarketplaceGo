import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';
import '../widgets/legal_document_screen.dart';

/// `/legal/aipd` — RGPD art. 35 Data Protection Impact Assessment.
///
/// Mirrors the web `/fr/legal/aipd` page. Body lives in
/// `assets/legal/aipd.md` (copy of the canonical source at
/// `/legal/aipd.md`).
class LegalAipdScreen extends StatelessWidget {
  const LegalAipdScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return LegalDocumentScreen(
      title: l10n.legalDocAipdTitle,
      subtitle: l10n.legalDocAipdSubtitle,
      assetPath: 'assets/legal/aipd.md',
      englishNotice: l10n.legalEnglishNotice,
      lastUpdatedLabel: l10n.legalLastUpdated('2026-05-11'),
    );
  }
}
