import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../domain/entities/job_entity.dart';

/// Header card showing status pill, applicant type pill and posted date.
class JobDetailHeaderCard extends StatelessWidget {
  const JobDetailHeaderCard({super.key, required this.job});

  final JobEntity job;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final statusColor = job.isOpen ? Colors.green : Colors.grey;
    final applicantLabel = _applicantTypeLabel(job.applicantType, l10n);

    return Container(
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
              Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 4,
                ),
                decoration: BoxDecoration(
                  color: statusColor.withValues(alpha: 0.12),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Text(
                  job.isOpen ? l10n.jobStatusOpen : l10n.jobStatusClosed,
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: statusColor,
                  ),
                ),
              ),
              const Spacer(),
              Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 4,
                ),
                decoration: BoxDecoration(
                  color: theme.colorScheme.primary.withValues(alpha: 0.08),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Text(
                  applicantLabel,
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    color: theme.colorScheme.primary,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Icon(
                Icons.calendar_today_outlined,
                size: 14,
                color: appColors?.mutedForeground,
              ),
              const SizedBox(width: 6),
              Text(
                '${l10n.jobPostedOn} ${_formatDate(job.createdAt)}',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: appColors?.mutedForeground,
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

/// Card showing the budget range and one-shot vs long-term flag.
class JobDetailBudgetCard extends StatelessWidget {
  const JobDetailBudgetCard({super.key, required this.job});

  final JobEntity job;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final budgetLabel = job.budgetType == 'one_shot'
        ? l10n.budgetTypeOneShot
        : l10n.budgetTypeLongTerm;

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
      ),
      child: Row(
        children: [
          Container(
            width: 44,
            height: 44,
            decoration: BoxDecoration(
              color: theme.colorScheme.primary.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
            child: Icon(
              Icons.euro,
              color: theme.colorScheme.primary,
              size: 22,
            ),
          ),
          const SizedBox(width: 12),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                '${job.minBudget}€ - ${job.maxBudget}€',
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.bold,
                  color: theme.colorScheme.onSurface,
                ),
              ),
              Text(
                budgetLabel,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: appColors?.mutedForeground,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
