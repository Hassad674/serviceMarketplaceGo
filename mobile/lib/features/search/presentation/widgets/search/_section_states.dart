import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';

/// Empty state shown when the search returns zero profiles. Renders a
/// calm corail-soft icon chip with Fraunces title + tabac italic copy
/// and a reset CTA. Persona-aware: switches to the M-12 freelance copy
/// when the screen is the freelancer search.
///
/// Extracted from `search_screen.dart` as part of the NF-9 file split
/// (V7 audit). Behaviour unchanged.
class SearchEmptyState extends StatelessWidget {
  const SearchEmptyState({super.key, required this.onReset});

  final VoidCallback onReset;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    final isFreelancer = ModalRoute.of(context)?.settings.name?.contains('freelancer') ?? false;
    final title = isFreelancer
        ? l10n.freelancesSearchM12EmptyTitle
        : l10n.searchEmptyTitle;
    final description = isFreelancer
        ? l10n.freelancesSearchM12EmptyDescription
        : l10n.searchEmptyDescription;
    final cta =
        isFreelancer ? l10n.freelancesSearchM12EmptyCta : l10n.searchEmptyCta;

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Container(
          padding: const EdgeInsets.fromLTRB(24, 32, 24, 28),
          decoration: BoxDecoration(
            color: colorScheme.surfaceContainerLowest,
            border: Border.all(color: colors.border),
            borderRadius: BorderRadius.circular(AppTheme.radius2xl),
            boxShadow: AppTheme.cardShadow,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 56,
                height: 56,
                decoration: BoxDecoration(
                  color: colors.accentSoft,
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.search_off_rounded,
                  size: 26,
                  color: colorScheme.primary,
                ),
              ),
              const SizedBox(height: 14),
              Text(
                title,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.titleLarge.copyWith(
                  fontSize: 20,
                  color: colorScheme.onSurface,
                ),
              ),
              const SizedBox(height: 6),
              Text(
                description,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.body.copyWith(
                  fontSize: 13.5,
                  fontStyle: FontStyle.italic,
                  color: colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 18),
              OutlinedButton.icon(
                onPressed: onReset,
                icon: const Icon(Icons.refresh_rounded, size: 16),
                label: Text(cta),
                style: OutlinedButton.styleFrom(
                  side: BorderSide(color: colors.borderStrong),
                  foregroundColor: colorScheme.onSurface,
                  shape: const StadiumBorder(),
                  padding:
                      const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Error state shown when the search request fails. Same Soleil card
/// layout as the empty state but with the error icon + retry CTA.
///
/// Extracted from `search_screen.dart` as part of the NF-9 file split.
class SearchErrorState extends StatelessWidget {
  const SearchErrorState({super.key, required this.onRetry});

  final Future<void> Function() onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Container(
          padding: const EdgeInsets.fromLTRB(24, 32, 24, 28),
          decoration: BoxDecoration(
            color: colorScheme.surfaceContainerLowest,
            border: Border.all(color: colors.border),
            borderRadius: BorderRadius.circular(AppTheme.radius2xl),
            boxShadow: AppTheme.cardShadow,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 56,
                height: 56,
                decoration: BoxDecoration(
                  color: colors.accentSoft,
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.error_outline_rounded,
                  size: 26,
                  color: colorScheme.error,
                ),
              ),
              const SizedBox(height: 14),
              Text(
                l10n.somethingWentWrong,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.titleLarge.copyWith(
                  fontSize: 20,
                  color: colorScheme.onSurface,
                ),
              ),
              const SizedBox(height: 6),
              Text(
                l10n.couldNotLoadProfiles,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.body.copyWith(
                  fontSize: 13.5,
                  fontStyle: FontStyle.italic,
                  color: colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 18),
              FilledButton.icon(
                onPressed: onRetry,
                icon: const Icon(Icons.refresh_rounded, size: 16),
                label: Text(l10n.retry),
                style: FilledButton.styleFrom(
                  backgroundColor: colorScheme.primary,
                  foregroundColor: colorScheme.onPrimary,
                  shape: const StadiumBorder(),
                  padding:
                      const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
