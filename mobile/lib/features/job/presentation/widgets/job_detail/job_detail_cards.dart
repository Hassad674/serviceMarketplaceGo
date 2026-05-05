import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../domain/entities/job_entity.dart';

/// Soleil v2 header card — status + applicant type pills, posted date.
///
/// Keeps the public test contract intact: still renders the
/// `jobStatusOpen` / `jobStatusClosed` label, the applicant-type
/// label (`jobApplicantFreelancers` / `jobApplicantAgencies`) and the
/// posted date in `DD/MM/YYYY` format.
class JobDetailHeaderCard extends StatelessWidget {
  const JobDetailHeaderCard({super.key, required this.job});

  final JobEntity job;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final soleil = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    final isOpen = job.isOpen;
    final applicantLabel = _applicantTypeLabel(job.applicantType, l10n);
    final accentBg = soleil?.accentSoft ?? cs.primaryContainer;
    final accentFg = soleil?.primaryDeep ?? cs.onPrimaryContainer;

    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: cs.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(color: cs.outline),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              _StatusPill(isOpen: isOpen, label: isOpen ? l10n.jobStatusOpen : l10n.jobStatusClosed),
              const Spacer(),
              _SoftPill(
                label: applicantLabel,
                background: accentBg,
                foreground: accentFg,
              ),
            ],
          ),
          const SizedBox(height: 14),
          Row(
            children: [
              Icon(
                Icons.calendar_today_outlined,
                size: 14,
                color: cs.onSurfaceVariant,
              ),
              const SizedBox(width: 6),
              Text(
                '${l10n.jobPostedOn} ${_formatDate(job.createdAt)}',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: cs.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  String _applicantTypeLabel(String type, AppLocalizations l10n) {
    return switch (type) {
      'freelancers' => l10n.jobApplicantFreelancers,
      'agencies' => l10n.jobApplicantAgencies,
      _ => l10n.jobApplicantAll,
    };
  }

  String _formatDate(String isoDate) {
    try {
      final date = DateTime.parse(isoDate);
      return '${date.day.toString().padLeft(2, '0')}/${date.month.toString().padLeft(2, '0')}/${date.year}';
    } catch (_) {
      return isoDate;
    }
  }
}

class _StatusPill extends StatelessWidget {
  const _StatusPill({required this.isOpen, required this.label});

  final bool isOpen;
  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final soleil = theme.extension<AppColors>();
    final background = isOpen
        ? (soleil?.successSoft ?? cs.tertiaryContainer)
        : (soleil?.border ?? cs.outlineVariant);
    final foreground = isOpen
        ? (soleil?.success ?? cs.tertiary)
        : (soleil?.mutedForeground ?? cs.onSurfaceVariant);
    return _SoftPill(
      label: label,
      background: background,
      foreground: foreground,
    );
  }
}

class _SoftPill extends StatelessWidget {
  const _SoftPill({
    required this.label,
    required this.background,
    required this.foreground,
  });

  final String label;
  final Color background;
  final Color foreground;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        label,
        style: SoleilTextStyles.caption.copyWith(
          color: foreground,
          fontWeight: FontWeight.w700,
          fontSize: 11,
          letterSpacing: 0.4,
        ),
      ),
    );
  }
}

/// Soleil v2 budget card — corail-soft icon disc, mono kind label,
/// Fraunces budget range. Keeps the test contract: label uses
/// `budgetTypeOneShot` / `budgetTypeLongTerm`, budget reads
/// `min€ - max€`, and the euro icon stays present.
class JobDetailBudgetCard extends StatelessWidget {
  const JobDetailBudgetCard({super.key, required this.job});

  final JobEntity job;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final soleil = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final budgetLabel = job.budgetType == 'one_shot'
        ? l10n.budgetTypeOneShot
        : l10n.budgetTypeLongTerm;
    final iconBg = soleil?.accentSoft ?? cs.primaryContainer;

    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: cs.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(color: cs.outline),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Row(
        children: [
          Container(
            width: 44,
            height: 44,
            decoration: BoxDecoration(
              color: iconBg,
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
            child: Icon(
              Icons.euro,
              color: cs.primary,
              size: 22,
            ),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '${job.minBudget}€ - ${job.maxBudget}€',
                  style: SoleilTextStyles.titleMedium.copyWith(
                    color: cs.onSurface,
                    fontSize: 18,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  budgetLabel,
                  style: SoleilTextStyles.mono.copyWith(
                    color: cs.onSurfaceVariant,
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    letterSpacing: 0.6,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
