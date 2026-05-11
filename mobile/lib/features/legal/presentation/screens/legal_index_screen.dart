import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';

/// `/legal` — mobile mirror of the web [LegalIndexPage].
///
/// Renders a short intro paragraph followed by a vertical list of 6
/// cards (one per long-form document). Each card holds the document
/// title, a one-paragraph summary, and a `Référence — …` pill at the
/// bottom; tapping it navigates to the matching detail route via
/// [GoRouter].
///
/// The list of documents is defined as a static const so the order is
/// deterministic and matches the web sommaire exactly.
class LegalIndexScreen extends StatelessWidget {
  const LegalIndexScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();

    return Scaffold(
      appBar: AppBar(title: Text(l10n.legalIndexTitle)),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
          children: [
            Text(
              l10n.legalIndexIntro,
              style: SoleilTextStyles.bodyLarge.copyWith(
                color: colors?.mutedForeground ??
                    theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 16),
            Text(
              l10n.legalSectionDocs,
              style: SoleilTextStyles.headlineMedium.copyWith(
                color: theme.colorScheme.onSurface,
              ),
            ),
            const SizedBox(height: 12),
            for (final doc in _legalDocs(l10n)) ...[
              _LegalDocCard(doc: doc),
              const SizedBox(height: 12),
            ],
          ],
        ),
      ),
    );
  }
}

/// Data shape for the 6 cards on the index screen. Built from the i18n
/// surface (titles, summaries, references) and a static route path so
/// the card widget stays stateless and pure.
@immutable
class LegalDocLink {
  const LegalDocLink({
    required this.title,
    required this.summary,
    required this.reference,
    required this.route,
  });

  final String title;
  final String summary;
  final String reference;
  final String route;
}

/// Builds the 6 document links. Pulled out of `build` so it can be
/// reused in tests and so the order is auditable in one place.
List<LegalDocLink> _legalDocs(AppLocalizations l10n) => [
      LegalDocLink(
        title: l10n.legalDocRegistreTitle,
        summary: l10n.legalDocRegistreSummary,
        reference: l10n.legalDocRegistreReference,
        route: RoutePaths.legalRegistre,
      ),
      LegalDocLink(
        title: l10n.legalDocAipdTitle,
        summary: l10n.legalDocAipdSummary,
        reference: l10n.legalDocAipdReference,
        route: RoutePaths.legalAipd,
      ),
      LegalDocLink(
        title: l10n.legalDocDpaTitle,
        summary: l10n.legalDocDpaSummary,
        reference: l10n.legalDocDpaReference,
        route: RoutePaths.legalDpaTemplate,
      ),
      LegalDocLink(
        title: l10n.legalDocPrivacyTitle,
        summary: l10n.legalDocPrivacySummary,
        reference: l10n.legalDocPrivacyReference,
        route: RoutePaths.legalPrivacy,
      ),
      LegalDocLink(
        title: l10n.legalDocCguTitle,
        summary: l10n.legalDocCguSummary,
        reference: l10n.legalDocCguReference,
        route: RoutePaths.legalCgu,
      ),
      LegalDocLink(
        title: l10n.legalDocCgvTitle,
        summary: l10n.legalDocCgvSummary,
        reference: l10n.legalDocCgvReference,
        route: RoutePaths.legalCgv,
      ),
    ];

/// Visible-for-testing accessor — lets tests verify the deterministic
/// ordering and link mapping without re-implementing the list.
@visibleForTesting
List<LegalDocLink> legalDocsForTesting(AppLocalizations l10n) =>
    _legalDocs(l10n);

/// Single card used on the sommaire — title (Fraunces), summary
/// (Inter Tight), corail reference pill, navigates to its route on tap.
class _LegalDocCard extends StatelessWidget {
  const _LegalDocCard({required this.doc});

  final LegalDocLink doc;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    return Material(
      color: theme.colorScheme.surfaceContainerLowest,
      borderRadius: BorderRadius.circular(AppTheme.radiusXl),
      child: InkWell(
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        onTap: () => context.push(doc.route),
        child: Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(AppTheme.radiusXl),
            border: Border.all(
              color: colors?.border ?? theme.dividerColor,
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                doc.title,
                style: SoleilTextStyles.titleLarge.copyWith(
                  color: theme.colorScheme.onSurface,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                doc.summary,
                style: SoleilTextStyles.body.copyWith(
                  color: colors?.mutedForeground ??
                      theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 12),
              Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 4,
                ),
                decoration: BoxDecoration(
                  color:
                      colors?.accentSoft ?? theme.colorScheme.primaryContainer,
                  borderRadius: BorderRadius.circular(AppTheme.radiusFull),
                ),
                child: Text(
                  '${l10n.legalReferenceLabel} — ${doc.reference}',
                  style: SoleilTextStyles.caption.copyWith(
                    color: colors?.primaryDeep ?? theme.colorScheme.primary,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
