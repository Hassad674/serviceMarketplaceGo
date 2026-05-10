import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/theme/theme_text_styles.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../providers/stats_period_provider.dart';
import '../widgets/applications_card.dart';
import '../widgets/keywords_card.dart';
import '../widgets/period_selector.dart';
import '../widgets/visibility_card.dart';

/// `/stats` — detailed statistics screen for Provider/Agency/Referrer
/// orgs (visibility, search appearances, avg position, top keywords).
///
/// Enterprise orgs see a "Coming soon" placeholder — the enterprise-side
/// stats page (applications timeline) will land in a follow-up agent.
///
/// Routed via [RoutePaths.stats]; reached from the role-aware home's
/// "See detailed stats" CTA shipped in D2.
class StatsScreen extends ConsumerWidget {
  const StatsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final orgType =
        ref.watch(authProvider).organization?['type'] as String?;
    final isEnterprise = orgType == 'enterprise_company';

    return Scaffold(
      backgroundColor: theme.colorScheme.surface,
      appBar: AppBar(
        backgroundColor: theme.colorScheme.surface,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
        title: Text(
          l10n.statsScreenTitle,
          style: SoleilTextStyles.titleLarge.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
      ),
      body: isEnterprise
          ? const _EnterprisePlaceholder()
          : const _ProviderStatsBody(),
    );
  }
}

/// Provider/Agency/Referrer body: period selector + 3 cards + keywords.
class _ProviderStatsBody extends ConsumerWidget {
  const _ProviderStatsBody();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final period = ref.watch(statsPeriodProvider);

    Future<void> onRefresh() async {
      // Invalidating the period family entries forces a refetch of all
      // dependent providers (visibility + keywords).
      ref.invalidate(statsPeriodProvider);
    }

    return RefreshIndicator(
      color: Theme.of(context).colorScheme.primary,
      onRefresh: onRefresh,
      child: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          PeriodSelector(
            value: period,
            onChanged: (next) {
              ref.read(statsPeriodProvider.notifier).set(next);
            },
          ),
          const SizedBox(height: 16),
          const VisibilityCard(),
          const SizedBox(height: 12),
          const ApplicationsCard(),
          const SizedBox(height: 12),
          const KeywordsCard(),
        ],
      ),
    );
  }
}

/// Enterprise placeholder card — full-screen "Bientôt disponible" until
/// the enterprise-side stats page (applications timeline) ships.
class _EnterprisePlaceholder extends StatelessWidget {
  const _EnterprisePlaceholder();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.bar_chart_rounded,
              size: 48,
              color: appColors?.mutedForeground ?? theme.colorScheme.outline,
            ),
            const SizedBox(height: 16),
            Text(
              l10n.comingSoon,
              textAlign: TextAlign.center,
              style: SoleilTextStyles.titleMedium.copyWith(
                color: theme.colorScheme.onSurface,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              l10n.statsEnterprisePlaceholderBody,
              textAlign: TextAlign.center,
              style: SoleilTextStyles.body.copyWith(
                color: appColors?.mutedForeground ??
                    theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
