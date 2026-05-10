import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../job/presentation/providers/job_provider.dart';
import '../../../proposal/presentation/providers/proposal_provider.dart';
import '../../domain/stats_period.dart';
import '../providers/stats_visibility_provider.dart';
import 'actions_todo_card.dart';
import 'stat_tile.dart';

/// Provider/Agency layout — 4 stat tiles + actions todo + stats CTA.
///
/// Tiles (mirror web's provider dashboard exactly):
/// 1. Profile views (7d) - visibilityStats.totalViews
/// 2. Search appearances (7d) - visibilityStats.searchAppearances
/// 3. Average search position - visibilityStats.avgSearchPosition (em-dash on null)
/// 4. Monthly revenue (placeholder em-dash — wallet/monthly-revenue hook
///    pending in D3+; FLAGGED in agent report).
class ProviderRoleLayout extends ConsumerWidget {
  const ProviderRoleLayout({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final visibility =
        ref.watch(visibilityStatsProvider(StatsPeriod.sevenDays));
    final tiles = visibility.when(
      data: (stats) => [
        StatTile(
          label: 'Profile views',
          value: '${stats.totalViews}',
          subtitle: 'last 7 days',
        ),
        StatTile(
          label: 'Search appearances',
          value: '${stats.searchAppearances}',
          subtitle: 'last 7 days',
        ),
        StatTile(
          label: 'Avg search position',
          value: stats.avgSearchPosition?.toStringAsFixed(1),
          subtitle: stats.avgSearchPosition == null
              ? 'no rankings yet'
              : 'last 7 days',
        ),
        const StatTile(
          label: 'Monthly revenue',
          value: null,
          subtitle: 'last 30 days',
        ),
      ],
      loading: () => const [
        StatTile(label: 'Profile views', value: null, isLoading: true),
        StatTile(label: 'Search appearances', value: null, isLoading: true),
        StatTile(label: 'Avg search position', value: null, isLoading: true),
        StatTile(label: 'Monthly revenue', value: null, isLoading: true),
      ],
      error: (_, __) => const [
        StatTile(label: 'Profile views', value: null),
        StatTile(label: 'Search appearances', value: null),
        StatTile(label: 'Avg search position', value: null),
        StatTile(label: 'Monthly revenue', value: null),
      ],
    );

    return _LayoutColumn(
      tiles: tiles,
      footer: const _StatsDetailButton(),
    );
  }
}

/// Enterprise layout — 4 tiles + actions todo.
///
/// Tiles:
/// 1. Active recruitments (count of own jobs)
/// 2. Applications received (7d) — applicationsSeries.totalCount
/// 3. Spending (30d) — placeholder em-dash (wallet hook missing — FLAGGED)
/// 4. Pending proposals — projectsProvider filtered by status='pending'
class EnterpriseRoleLayout extends ConsumerWidget {
  const EnterpriseRoleLayout({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final apps = ref
        .watch(enterpriseApplicationsStatsProvider(StatsPeriod.sevenDays));
    final myJobs = ref.watch(myJobsProvider);
    final proposals = ref.watch(projectsProvider);

    final activeJobsValue = myJobs.maybeWhen(
      data: (jobs) =>
          '${jobs.where((j) => (j.status) == 'open' || (j.status) == 'active').length}',
      orElse: () => null,
    );
    final applicationsValue = apps.maybeWhen(
      data: (s) => '${s.totalCount}',
      orElse: () => null,
    );
    final pendingValue = proposals.maybeWhen(
      data: (p) => '${p.where((x) => x.status == 'pending').length}',
      orElse: () => null,
    );

    final tiles = <StatTile>[
      StatTile(
        label: 'Active recruitments',
        value: activeJobsValue,
        subtitle: 'open or active',
        isLoading: myJobs.isLoading,
      ),
      StatTile(
        label: 'Applications',
        value: applicationsValue,
        subtitle: 'last 7 days',
        isLoading: apps.isLoading,
      ),
      const StatTile(
        label: 'Spending',
        value: null,
        subtitle: 'last 30 days',
      ),
      StatTile(
        label: 'To review',
        value: pendingValue,
        subtitle: 'pending proposals',
        isLoading: proposals.isLoading,
      ),
    ];

    return _LayoutColumn(tiles: tiles);
  }
}

/// Referrer layout — 4 placeholder tiles + actions todo.
///
/// All four metrics need backend hooks not yet exposed on mobile (web
/// uses a /referrals/stats endpoint that has no Flutter binding yet).
/// Tiles render em-dashes; D3+ wires real values when the referrer
/// repository ships.
class ReferrerRoleLayout extends ConsumerWidget {
  const ReferrerRoleLayout({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return const _LayoutColumn(
      tiles: [
        StatTile(label: 'Active referrals', value: null),
        StatTile(label: 'Pending commissions', value: null),
        StatTile(
          label: 'Paid out',
          value: null,
          subtitle: 'last 30 days',
        ),
        StatTile(label: 'Lifetime', value: null),
      ],
    );
  }
}

class _LayoutColumn extends StatelessWidget {
  const _LayoutColumn({required this.tiles, this.footer});

  final List<StatTile> tiles;
  final Widget? footer;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        StatTileGrid(tiles: tiles),
        const SizedBox(height: 24),
        const ActionsTodoCard(),
        if (footer != null) ...[
          const SizedBox(height: 16),
          footer!,
        ],
      ],
    );
  }
}

class _StatsDetailButton extends StatelessWidget {
  const _StatsDetailButton();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: () => GoRouter.of(context).push('/stats'),
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 4),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text(
                'See detailed stats',
                style: SoleilTextStyles.button.copyWith(
                  color: theme.colorScheme.primary,
                ),
              ),
              const SizedBox(width: 6),
              Icon(
                Icons.arrow_forward_rounded,
                size: 18,
                color: theme.colorScheme.primary,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
