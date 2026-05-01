import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../../core/router/app_router.dart';
import '../../../../../core/theme/app_theme.dart';
import '../../../../../core/utils/permissions.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../domain/entities/job_entity.dart';
import '../../providers/job_provider.dart';
import 'job_list_popup_menu.dart';
import 'job_list_status_badge.dart';

/// Compact card displayed for each job in the "My jobs" list.
class JobListCard extends ConsumerWidget {
  const JobListCard({super.key, required this.job});

  final JobEntity job;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();
    final canEdit = ref.watch(hasPermissionProvider(OrgPermission.jobsEdit));
    final canDelete =
        ref.watch(hasPermissionProvider(OrgPermission.jobsDelete));

    return GestureDetector(
      onTap: () => context.push(RoutePaths.jobDetail, extra: job.id),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(AppTheme.radiusLg),
          border: Border.all(
            color: appColors?.border ?? theme.dividerColor,
          ),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    job.title,
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                      color: theme.colorScheme.onSurface,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                const SizedBox(width: 4),
                JobListStatusBadge(isOpen: job.isOpen),
                if (canEdit || canDelete)
                  JobListPopupMenu(
                    job: job,
                    canEdit: canEdit,
                    canDelete: canDelete,
                    onEdit: () =>
                        context.push(RoutePaths.jobEdit, extra: job.id),
                    onClose: () => _handleClose(context, ref, l10n),
                    onReopen: () => _handleReopen(context, ref, l10n),
                    onDelete: () => _handleDelete(context, ref, l10n),
                  ),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              job.description,
              style: theme.textTheme.bodySmall?.copyWith(
                color: appColors?.mutedForeground,
              ),
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
            if (job.skills.isNotEmpty) ...[
              const SizedBox(height: 8),
              Wrap(
                spacing: 6,
                runSpacing: 4,
                children: job.skills
                    .map(
                      (s) => Chip(
                        label: Text(
                          s,
                          style: TextStyle(
                            fontSize: 11,
                            color: theme.colorScheme.onSurface,
                          ),
                        ),
                        materialTapTargetSize:
                            MaterialTapTargetSize.shrinkWrap,
                        visualDensity: VisualDensity.compact,
                        backgroundColor: theme.colorScheme.primary
                            .withValues(alpha: 0.08),
                        side: BorderSide.none,
                      ),
                    )
                    .toList(),
              ),
            ],
            const SizedBox(height: 12),
            _Footer(job: job, mutedFg: appColors?.mutedForeground),
          ],
        ),
      ),
    );
  }

  Future<void> _handleClose(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
  ) async {
    final ok = await closeJobAction(ref, job.id);
    if (!context.mounted) return;
    if (!ok) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.unexpectedError)),
      );
    }
  }

  Future<void> _handleReopen(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
  ) async {
    final ok = await reopenJobAction(ref, job.id);
    if (!context.mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(ok ? l10n.jobReopenSuccess : l10n.unexpectedError),
        backgroundColor: ok ? const Color(0xFF22C55E) : null,
      ),
    );
  }

  Future<void> _handleDelete(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.jobDelete),
        content: Text(l10n.jobDeleteConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(l10n.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            child: Text(l10n.jobDelete),
          ),
        ],
      ),
    );
    if (confirmed != true || !context.mounted) return;

    final ok = await deleteJobAction(ref, job.id);
    if (!context.mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(ok ? l10n.jobDeleteSuccess : l10n.unexpectedError),
        backgroundColor: ok ? const Color(0xFF22C55E) : null,
      ),
    );
  }
}

class _Footer extends StatelessWidget {
  const _Footer({required this.job, this.mutedFg});

  final JobEntity job;
  final Color? mutedFg;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Row(
      children: [
        Text(
          '${job.minBudget}€ - ${job.maxBudget}€',
          style: theme.textTheme.bodySmall?.copyWith(color: mutedFg),
        ),
        const Spacer(),
        if (job.totalApplicants > 0) ...[
          Icon(Icons.people_outline, size: 14, color: mutedFg),
          const SizedBox(width: 4),
          Text(
            l10n.jobTotalApplicants(job.totalApplicants),
            style: theme.textTheme.bodySmall?.copyWith(
              color: mutedFg,
              fontSize: 11,
            ),
          ),
        ],
        if (job.newApplicants > 0) ...[
          const SizedBox(width: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
            decoration: BoxDecoration(
              color: const Color(0xFFF43F5E),
              borderRadius: BorderRadius.circular(10),
            ),
            child: Text(
              l10n.jobNewApplicants(job.newApplicants),
              style: const TextStyle(
                fontSize: 10,
                fontWeight: FontWeight.w600,
                color: Colors.white,
              ),
            ),
          ),
        ],
      ],
    );
  }
}
