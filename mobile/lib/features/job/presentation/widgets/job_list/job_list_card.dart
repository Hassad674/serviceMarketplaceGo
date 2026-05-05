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

/// M-07 — Single job card on the entreprise listing. Soleil v2 anatomy:
///   - status pill (sapin-soft / mute) + italic mono relative date
///   - Fraunces title + tabac excerpt + skill chips on background
///   - dashed divider, mono budget + duration row
///   - applicants block (corailSoft pill for fresh) when > 0
///   - kebab menu preserved (edit / close-or-reopen / delete)
class JobListCard extends ConsumerWidget {
  const JobListCard({super.key, required this.job});

  final JobEntity job;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final palette = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final canEdit = ref.watch(hasPermissionProvider(OrgPermission.jobsEdit));
    final canDelete =
        ref.watch(hasPermissionProvider(OrgPermission.jobsDelete));

    return Material(
      color: colorScheme.surface,
      borderRadius: BorderRadius.circular(AppTheme.radiusXl),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: () => context.push(RoutePaths.jobDetail, extra: job.id),
        child: DecoratedBox(
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(AppTheme.radiusXl),
            border: Border.all(
              color: palette?.border ?? theme.dividerColor,
              width: 1,
            ),
            boxShadow: AppTheme.cardShadow,
          ),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _CardHeader(
                  job: job,
                  canEdit: canEdit,
                  canDelete: canDelete,
                  onEdit: () =>
                      context.push(RoutePaths.jobEdit, extra: job.id),
                  onClose: () => _handleClose(context, ref, l10n),
                  onReopen: () => _handleReopen(context, ref, l10n),
                  onDelete: () => _handleDelete(context, ref, l10n),
                ),
                const SizedBox(height: 8),
                Text(
                  job.title,
                  style: SoleilTextStyles.titleMedium.copyWith(
                    color: colorScheme.onSurface,
                    fontWeight: FontWeight.w600,
                    fontSize: 16,
                    height: 1.25,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
                if (job.description.isNotEmpty) ...[
                  const SizedBox(height: 6),
                  Text(
                    job.description,
                    style: SoleilTextStyles.body.copyWith(
                      color: palette?.mutedForeground ??
                          colorScheme.onSurfaceVariant,
                      fontSize: 13,
                      height: 1.45,
                    ),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
                if (job.skills.isNotEmpty) ...[
                  const SizedBox(height: 10),
                  _SkillChips(
                    skills: job.skills,
                    palette: palette,
                    background: colorScheme.surfaceContainerLowest,
                    onSurfaceVariant: colorScheme.onSurfaceVariant,
                  ),
                ],
                const SizedBox(height: 12),
                _Divider(color: palette?.border ?? theme.dividerColor),
                const SizedBox(height: 12),
                _Footer(
                  job: job,
                  l10n: l10n,
                  palette: palette,
                  colorScheme: colorScheme,
                ),
                if (job.totalApplicants > 0) ...[
                  const SizedBox(height: 10),
                  _ApplicantsBlock(
                    job: job,
                    l10n: l10n,
                    palette: palette,
                    colorScheme: colorScheme,
                  ),
                ],
              ],
            ),
          ),
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
    final colorScheme = Theme.of(context).colorScheme;
    final palette = Theme.of(context).extension<AppColors>();
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(ok ? l10n.jobReopenSuccess : l10n.unexpectedError),
        backgroundColor: ok
            ? (palette?.success ?? colorScheme.primary)
            : null,
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
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            child: Text(l10n.jobDelete),
          ),
        ],
      ),
    );
    if (confirmed != true || !context.mounted) return;

    final ok = await deleteJobAction(ref, job.id);
    if (!context.mounted) return;
    final colorScheme = Theme.of(context).colorScheme;
    final palette = Theme.of(context).extension<AppColors>();
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(ok ? l10n.jobDeleteSuccess : l10n.unexpectedError),
        backgroundColor: ok
            ? (palette?.success ?? colorScheme.primary)
            : null,
      ),
    );
  }
}

/// Card top row — Soleil status pill + mono italic relative date,
/// then the kebab menu pinned right.
class _CardHeader extends StatelessWidget {
  const _CardHeader({
    required this.job,
    required this.canEdit,
    required this.canDelete,
    required this.onEdit,
    required this.onClose,
    required this.onReopen,
    required this.onDelete,
  });

  final JobEntity job;
  final bool canEdit;
  final bool canDelete;
  final VoidCallback onEdit;
  final VoidCallback onClose;
  final VoidCallback onReopen;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final palette = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    final dateLabel = _formatRelative(
      l10n: l10n,
      from:
          !job.isOpen && job.closedAt != null ? job.closedAt! : job.createdAt,
    );

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          child: Wrap(
            crossAxisAlignment: WrapCrossAlignment.center,
            spacing: 8,
            runSpacing: 4,
            children: [
              _SoleilStatusPill(isOpen: job.isOpen, palette: palette),
              Text(
                !job.isOpen && job.closedAt != null
                    ? l10n.jobsClosedRelative(dateLabel)
                    : l10n.jobsPublishedRelative(dateLabel),
                style: SoleilTextStyles.mono.copyWith(
                  fontSize: 10.5,
                  fontWeight: FontWeight.w500,
                  letterSpacing: 0.4,
                  color: palette?.subtleForeground ??
                      theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ),
        if (canEdit || canDelete)
          JobListPopupMenu(
            job: job,
            canEdit: canEdit,
            canDelete: canDelete,
            onEdit: onEdit,
            onClose: onClose,
            onReopen: onReopen,
            onDelete: onDelete,
          ),
      ],
    );
  }
}

/// Soleil status pill — sapin-soft for open, neutral border-tabac for
/// closed. Uses theme-aware tokens, never hardcoded hex.
class _SoleilStatusPill extends StatelessWidget {
  const _SoleilStatusPill({required this.isOpen, required this.palette});

  final bool isOpen;
  final AppColors? palette;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final bg = isOpen
        ? (palette?.successSoft ?? theme.colorScheme.surfaceContainerLowest)
        : (palette?.border ?? theme.dividerColor);
    final fg = isOpen
        ? (palette?.success ?? theme.colorScheme.primary)
        : (palette?.mutedForeground ?? theme.colorScheme.onSurfaceVariant);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        isOpen ? l10n.jobStatusOpen : l10n.jobStatusClosed,
        style: SoleilTextStyles.caption.copyWith(
          color: fg,
          fontSize: 10.5,
          fontWeight: FontWeight.w700,
          height: 1.1,
        ),
      ),
    );
  }
}

/// Skill chips — small pill chips on the page background, capped at 6
/// with an overflow counter to avoid wrapping multiple lines on phone
/// widths.
class _SkillChips extends StatelessWidget {
  const _SkillChips({
    required this.skills,
    required this.palette,
    required this.background,
    required this.onSurfaceVariant,
  });

  final List<String> skills;
  final AppColors? palette;
  final Color background;
  final Color onSurfaceVariant;

  @override
  Widget build(BuildContext context) {
    final visible = skills.length > 6 ? skills.sublist(0, 6) : skills;
    final remaining = skills.length - visible.length;
    return Wrap(
      spacing: 6,
      runSpacing: 4,
      children: [
        for (final s in visible)
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
            decoration: BoxDecoration(
              color: background,
              borderRadius: BorderRadius.circular(AppTheme.radiusFull),
            ),
            child: Text(
              s,
              style: SoleilTextStyles.caption.copyWith(
                fontSize: 11,
                color: palette?.mutedForeground ?? onSurfaceVariant,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
        if (remaining > 0)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
            child: Text(
              '+$remaining',
              style: SoleilTextStyles.caption.copyWith(
                fontSize: 11,
                color: palette?.subtleForeground ?? onSurfaceVariant,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
      ],
    );
  }
}

/// Subtle horizontal separator (1px line in the border tone).
class _Divider extends StatelessWidget {
  const _Divider({required this.color});

  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(height: 1, color: color);
  }
}

/// Footer — mono budget, then duration with clock icon.
class _Footer extends StatelessWidget {
  const _Footer({
    required this.job,
    required this.l10n,
    required this.palette,
    required this.colorScheme,
  });

  final JobEntity job;
  final AppLocalizations l10n;
  final AppColors? palette;
  final ColorScheme colorScheme;

  @override
  Widget build(BuildContext context) {
    final mutedFg = palette?.mutedForeground ?? colorScheme.onSurfaceVariant;
    return Wrap(
      spacing: 14,
      runSpacing: 6,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.euro_rounded, size: 13, color: mutedFg),
            const SizedBox(width: 4),
            Text(
              '${job.minBudget} – ${job.maxBudget} €',
              style: SoleilTextStyles.mono.copyWith(
                fontSize: 12,
                fontWeight: FontWeight.w600,
                color: colorScheme.onSurface,
              ),
            ),
          ],
        ),
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.schedule_rounded, size: 13, color: mutedFg),
            const SizedBox(width: 4),
            Text(
              job.budgetType == 'one_shot' ? l10n.jobsOneShot : l10n.jobsLongTerm,
              style: SoleilTextStyles.caption.copyWith(
                fontSize: 12,
                color: mutedFg,
              ),
            ),
          ],
        ),
      ],
    );
  }
}

/// Applicants block — only rendered when totalApplicants > 0. Soft
/// background panel with people icon + count, plus corail-soft "X new"
/// pill when newApplicants > 0.
class _ApplicantsBlock extends StatelessWidget {
  const _ApplicantsBlock({
    required this.job,
    required this.l10n,
    required this.palette,
    required this.colorScheme,
  });

  final JobEntity job;
  final AppLocalizations l10n;
  final AppColors? palette;
  final ColorScheme colorScheme;

  @override
  Widget build(BuildContext context) {
    final mutedFg = palette?.mutedForeground ?? colorScheme.onSurfaceVariant;
    final corail = colorScheme.primary;
    final corailSoft = palette?.accentSoft ?? colorScheme.surfaceContainerLowest;
    final ivoire = colorScheme.surfaceContainerLowest;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: ivoire,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Row(
        children: [
          Icon(Icons.people_outline_rounded, size: 15, color: mutedFg),
          const SizedBox(width: 6),
          Text(
            '${job.totalApplicants}',
            style: SoleilTextStyles.bodyEmphasis.copyWith(
              fontSize: 13,
              color: colorScheme.onSurface,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(width: 4),
          Text(
            job.totalApplicants == 1
                ? l10n.jobsApplicantsOne
                : l10n.jobsApplicants,
            style: SoleilTextStyles.caption.copyWith(
              fontSize: 12,
              color: mutedFg,
            ),
          ),
          const SizedBox(width: 8),
          if (job.newApplicants > 0)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
              decoration: BoxDecoration(
                color: corailSoft,
                borderRadius: BorderRadius.circular(AppTheme.radiusFull),
              ),
              child: Text(
                l10n.jobsApplicantsNew(job.newApplicants),
                style: SoleilTextStyles.caption.copyWith(
                  fontSize: 10.5,
                  fontWeight: FontWeight.w700,
                  color: corail,
                ),
              ),
            ),
          const Spacer(),
          Text(
            l10n.jobsViewArrow,
            style: SoleilTextStyles.caption.copyWith(
              color: corail,
              fontWeight: FontWeight.w700,
              fontSize: 12,
            ),
          ),
        ],
      ),
    );
  }
}

// ─── Helpers ────────────────────────────────────────────────────────

String _formatRelative({
  required AppLocalizations l10n,
  required String from,
}) {
  final parsed = DateTime.tryParse(from);
  if (parsed == null) return '';
  final diff = DateTime.now().difference(parsed);
  if (diff.inMinutes < 1) return l10n.jobsJustNow;
  if (diff.inMinutes < 60) return l10n.jobsMinutesAgo(diff.inMinutes);
  if (diff.inHours < 24) return l10n.jobsHoursAgo(diff.inHours);
  if (diff.inDays < 7) return l10n.jobsDaysAgo(diff.inDays);
  final weeks = (diff.inDays / 7).floor();
  return l10n.jobsWeeksAgo(weeks);
}
