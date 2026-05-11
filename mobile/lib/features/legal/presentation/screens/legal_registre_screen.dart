import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';
import '../widgets/legal_document_screen.dart';

/// `/legal/registre` — RGPD art. 30 register of processing activities.
///
/// Thin wrapper around [LegalDocumentScreen]: every detail screen is
/// a fixed (title, subtitle, asset path) triple resolved from i18n.
/// Keeping each one in its own file lets the router import a single
/// symbol per route — symmetric with the 6 web `page.tsx` files under
/// `web/src/app/[locale]/(public)/legal/<slug>/`.
class LegalRegistreScreen extends StatelessWidget {
  const LegalRegistreScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return LegalDocumentScreen(
      title: l10n.legalDocRegistreTitle,
      subtitle: l10n.legalDocRegistreSubtitle,
      assetPath: 'assets/legal/registre.md',
      englishNotice: l10n.legalEnglishNotice,
      lastUpdatedLabel: l10n.legalLastUpdated('2026-05-11'),
    );
  }
}
