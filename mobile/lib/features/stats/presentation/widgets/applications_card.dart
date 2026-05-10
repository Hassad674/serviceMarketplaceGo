import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../providers/stats_data_providers.dart';
import 'sparkline_painter.dart';
import 'stats_card_shell.dart';

/// Job applications card. Only renders when the requesting org is an
/// enterprise — provider/agency/referrer orgs have no applications
/// timeline to show, so the card collapses to `SizedBox.shrink()`
/// rather than displaying a placeholder.
///
/// Data: [statsApplicationsProvider] (auto-disposed, period-driven).
class ApplicationsCard extends ConsumerWidget {
  const ApplicationsCard({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final orgType =
        ref.watch(authProvider).organization?['type'] as String?;
    if (orgType != 'enterprise_company') return const SizedBox.shrink();

    final l10n = AppLocalizations.of(context)!;
    final async = ref.watch(statsApplicationsProvider);
    return async.when(
      loading: () => StatsCardShell(
        title: l10n.statsApplicationsTitle,
        child: const StatsCardSkeleton(height: 100),
      ),
      error: (_, __) => StatsCardShell(
        title: l10n.statsApplicationsTitle,
        child: StatsCardError(
          message: l10n.statsLoadError,
          onRetry: () => ref.invalidate(statsApplicationsProvider),
        ),
      ),
      data: (series) {
        if (series.series.isEmpty || series.totalCount == 0) {
          return StatsCardShell(
            title: l10n.statsApplicationsTitle,
            child: StatsCardEmpty(message: l10n.statsInsufficientData),
          );
        }
        final theme = Theme.of(context);
        final appColors = theme.extension<AppColors>();
        return StatsCardShell(
          title: l10n.statsApplicationsTitle,
          subtitle: l10n.statsApplicationsSubtitle,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                '${series.totalCount}',
                style: SoleilTextStyles.headlineLarge.copyWith(
                  color: theme.colorScheme.primary,
                ),
              ),
              const SizedBox(height: 12),
              SizedBox(
                height: 56,
                child: RepaintBoundary(
                  child: CustomPaint(
                    size: const Size.fromHeight(56),
                    painter: SparklinePainter(
                      values:
                          series.series.map((p) => p.count).toList(),
                      lineColor: theme.colorScheme.primary,
                      fillColor: appColors?.accentSoft ??
                          theme.colorScheme.primaryContainer,
                    ),
                    child: const SizedBox.expand(),
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}
