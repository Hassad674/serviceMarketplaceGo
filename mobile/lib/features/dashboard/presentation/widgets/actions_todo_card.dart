import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/dashboard_action.dart';
import '../providers/dashboard_actions_provider.dart';

/// "Actions à faire" card on the dashboard.
///
/// Subscribes to [dashboardActionsProvider] (which composes existing
/// feature providers — never opens a fresh request) and renders a
/// rounded ivoire card with one tappable row per pending action,
/// sorted by severity. Empty list collapses to a "Tout est à jour"
/// success state.
class ActionsTodoCard extends ConsumerWidget {
  const ActionsTodoCard({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final actions = ref.watch(dashboardActionsProvider);
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(color: colors.border),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          _CardHeader(count: actions.length),
          const SizedBox(height: 12),
          if (actions.isEmpty)
            const _EmptyState()
          else
            _ActionsList(actions: actions),
        ],
      ),
    );
  }
}

class _CardHeader extends StatelessWidget {
  const _CardHeader({required this.count});

  final int count;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      children: [
        Text(
          'Actions à faire',
          style: SoleilTextStyles.titleMedium.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
        const Spacer(),
        if (count > 0)
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
            decoration: BoxDecoration(
              color: theme.colorScheme.primary,
              borderRadius: BorderRadius.circular(999),
            ),
            child: Text(
              '$count',
              style: SoleilTextStyles.mono.copyWith(
                color: theme.colorScheme.onPrimary,
                fontWeight: FontWeight.w700,
                fontSize: 11,
              ),
            ),
          ),
      ],
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Row(
        children: [
          Icon(Icons.check_circle_rounded, color: colors.success, size: 22),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              'Tout est à jour',
              style: SoleilTextStyles.body.copyWith(
                color: theme.colorScheme.onSurface,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _ActionsList extends StatelessWidget {
  const _ActionsList({required this.actions});

  final List<DashboardAction> actions;

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        for (var i = 0; i < actions.length; i++) ...[
          if (i > 0) const Divider(height: 1, thickness: 0.5),
          _ActionRow(action: actions[i]),
        ],
      ],
    );
  }
}

class _ActionRow extends StatelessWidget {
  const _ActionRow({required this.action});

  final DashboardAction action;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return InkWell(
      onTap: () => GoRouter.of(context).go(action.route),
      borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 4),
        child: Row(
          children: [
            _SeverityDot(severity: action.severity),
            const SizedBox(width: 12),
            Expanded(child: _ActionCopy(action: action)),
            Icon(
              Icons.chevron_right_rounded,
              color: theme.colorScheme.onSurfaceVariant,
              size: 20,
            ),
          ],
        ),
      ),
    );
  }
}

class _ActionCopy extends StatelessWidget {
  const _ActionCopy({required this.action});

  final DashboardAction action;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          action.label,
          style: SoleilTextStyles.body.copyWith(
            color: theme.colorScheme.onSurface,
            fontWeight: FontWeight.w500,
          ),
        ),
        if (action.detail != null) ...[
          const SizedBox(height: 2),
          Text(
            action.detail!,
            style: SoleilTextStyles.caption.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ],
    );
  }
}

class _SeverityDot extends StatelessWidget {
  const _SeverityDot({required this.severity});

  final DashboardActionSeverity severity;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    Color tint;
    switch (severity) {
      case DashboardActionSeverity.critical:
        tint = theme.colorScheme.primary;
        break;
      case DashboardActionSeverity.warning:
        tint = colors.warning;
        break;
      case DashboardActionSeverity.info:
        tint = colors.success;
        break;
    }
    return Container(
      width: 10,
      height: 10,
      decoration: BoxDecoration(color: tint, shape: BoxShape.circle),
    );
  }
}
