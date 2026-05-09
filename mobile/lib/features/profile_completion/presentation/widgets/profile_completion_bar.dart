import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/theme_colors.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../domain/entities/profile_completion_report.dart';
import '../providers/profile_completion_providers.dart';

/// ProfileCompletionBar — Soleil v2 progress card surfacing the
/// "Profil rempli à X%" report on every profile-related screen.
///
/// Composes three states:
///
///   * loading / empty data -> renders nothing (no skeleton flash on
///     screens where the bar is a secondary affordance).
///   * complete -> hidden when [hideWhenComplete] is true to avoid a
///     dead UI block on a fully completed profile.
///   * partial -> renders the corail-filled bar plus a chevron pill
///     showing the missing-section count. Tapping pushes the user's
///     own profile screen so they can complete sections in place
///     (matches the web behaviour: no intermediate sheet, single
///     tap = land on the editor).
class ProfileCompletionBar extends ConsumerWidget {
  const ProfileCompletionBar({super.key, this.hideWhenComplete = false});

  /// When true, the widget collapses to `SizedBox.shrink()` once the
  /// report reaches 100%. Defaults to false: surfaces that want to
  /// celebrate completion (e.g. the profile page) keep the bar visible.
  final bool hideWhenComplete;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(profileCompletionProvider);
    return async.when(
      data: (report) =>
          _buildContent(context, ref, report, hideWhenComplete: hideWhenComplete),
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
    );
  }

  Widget _buildContent(
    BuildContext context,
    WidgetRef ref,
    ProfileCompletionReport report, {
    required bool hideWhenComplete,
  }) {
    if (hideWhenComplete && report.isComplete) {
      return const SizedBox.shrink();
    }
    if (report.totalSections == 0) {
      return const SizedBox.shrink();
    }

    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;

    return InkWell(
      onTap: () => _navigateToOwnProfile(context, ref),
      borderRadius: BorderRadius.circular(16),
      child: Ink(
        decoration: BoxDecoration(
          color: theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: colors.border),
        ),
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l10n.profileCompletionTitle(report.percent),
                        style: theme.textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        report.isComplete
                            ? l10n.profileCompletionSubtitleComplete
                            : l10n.profileCompletionSubtitle(
                                report.filledSections,
                                report.totalSections,
                              ),
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: colors.mutedForeground,
                        ),
                      ),
                    ],
                  ),
                ),
                if (!report.isComplete)
                  _MissingPill(count: report.missingCount),
              ],
            ),
            const SizedBox(height: 12),
            ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: LinearProgressIndicator(
                value: (report.percent / 100).clamp(0, 1).toDouble(),
                minHeight: 8,
                backgroundColor: colors.muted,
                valueColor: AlwaysStoppedAnimation<Color>(
                  report.isComplete
                      ? colors.success
                      : theme.colorScheme.primary,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// Navigates to the authenticated user's own profile screen, picking
  /// the right route per role:
  ///
  ///   * enterprise -> [RoutePaths.clientProfile]
  ///   * everyone else (provider, agency) -> [RoutePaths.profile]
  ///
  /// Falls back to [RoutePaths.profile] when the auth state is not
  /// hydrated yet — the provider profile screen is the safest default
  /// because it dispatches between freelance / agency layouts on its
  /// own and gracefully handles a missing org type.
  void _navigateToOwnProfile(BuildContext context, WidgetRef ref) {
    final auth = ref.read(authProvider);
    final role = (auth.user?['role'] as String?) ?? '';
    final orgType = (auth.organization?['type'] as String?) ?? '';
    final target = role == 'enterprise' || orgType == 'enterprise'
        ? RoutePaths.clientProfile
        : RoutePaths.profile;
    context.go(target);
  }
}

class _MissingPill extends StatelessWidget {
  const _MissingPill({required this.count});
  final int count;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: colors.accentSoft,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            '$count',
            style: theme.textTheme.labelSmall?.copyWith(
              color: colors.primaryDeep,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(width: 4),
          Icon(Icons.chevron_right, size: 14, color: colors.primaryDeep),
        ],
      ),
    );
  }
}
