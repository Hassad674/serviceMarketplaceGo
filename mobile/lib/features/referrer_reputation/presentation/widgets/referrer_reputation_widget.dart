import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/profile_display_card_shell.dart';
import '../../domain/entities/referrer_reputation.dart';
import '../providers/referrer_reputation_provider.dart';

/// Apporteur reputation surface: dedicated rating (distinct from the
/// freelance rating) + history of attributed missions. Visual grammar
/// mirrors the freelance project-history card so users find it
/// familiar, with labels that stay unambiguous about scope —
/// "Projets apportés" / "Avis des clients sur les prestataires
/// recommandés". Client identity is intentionally absent.
class ReferrerReputationWidget extends ConsumerWidget {
  final String orgId;

  const ReferrerReputationWidget({super.key, required this.orgId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final async = ref.watch(referrerReputationProvider(orgId));

    return ProfileDisplayCardShell(
      title: l10n.reputationSectionTitle,
      icon: Icons.history_edu_outlined,
      child: async.when(
        loading: () => const _LoadingBody(),
        error: (_, __) => _ErrorBody(message: l10n.reputationLoadError),
        data: (rep) => _Body(reputation: rep),
      ),
    );
  }
}

class _LoadingBody extends StatelessWidget {
  const _LoadingBody();

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: List.generate(
        2,
        (i) => Container(
          margin: const EdgeInsets.symmetric(vertical: 6),
          height: 96,
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surfaceContainerHighest,
            borderRadius: BorderRadius.circular(16),
          ),
        ),
      ),
    );
  }
}

class _ErrorBody extends StatelessWidget {
  const _ErrorBody({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Text(
        message,
        style: theme.textTheme.bodyMedium?.copyWith(
          color: theme.colorScheme.error,
        ),
      ),
    );
  }
}

class _Body extends StatelessWidget {
  const _Body({required this.reputation});

  final ReferrerReputation reputation;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final entries = reputation.history;

    if (entries.isEmpty) {
      return _EmptyState();
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _ReputationHeader(
          ratingAvg: reputation.ratingAvg,
          reviewCount: reputation.reviewCount,
          subtitle: l10n.reputationSectionSubtitle,
          ratingLabel: l10n.reputationRatingLabel,
        ),
        const SizedBox(height: 12),
        for (final entry in entries) ...[
          _EntryCard(entry: entry),
          const SizedBox(height: 8),
        ],
        if (reputation.hasMore) ...[
          const SizedBox(height: 4),
          Text(
            l10n.reputationLoadMore,
            style: theme.textTheme.labelMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
              fontStyle: FontStyle.italic,
            ),
          ),
        ],
      ],
    );
  }
}

class _EmptyState extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 20),
      child: Column(
        children: [
          Icon(
            Icons.insert_drive_file_outlined,
            size: 36,
            color: theme.colorScheme.onSurfaceVariant,
          ),
          const SizedBox(height: 12),
          Text(
            l10n.reputationEmptyTitle,
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            l10n.reputationEmptyDescription,
            textAlign: TextAlign.center,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
              fontStyle: FontStyle.italic,
            ),
          ),
        ],
      ),
    );
  }
}

class _ReputationHeader extends StatelessWidget {
  const _ReputationHeader({
    required this.ratingAvg,
    required this.reviewCount,
    required this.subtitle,
    required this.ratingLabel,
  });

  final double ratingAvg;
  final int reviewCount;
  final String subtitle;
  final String ratingLabel;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            const Icon(
              Icons.star_rounded,
              size: 18,
              color: Color(0xFFF59E0B),
            ),
            const SizedBox(width: 6),
            Text(
              '$ratingLabel · ${ratingAvg.toStringAsFixed(1)} / 5',
              style: theme.textTheme.labelLarge?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(width: 6),
            Text(
              '($reviewCount)',
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
        const SizedBox(height: 4),
        Text(
          subtitle,
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}

class _EntryCard extends StatelessWidget {
  const _EntryCard({required this.entry});

  final ReferrerProjectHistoryEntry entry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final hasReview = entry.rating != null;

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: hasReview
            ? theme.colorScheme.surface
            : theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(
          color: theme.colorScheme.outlineVariant.withValues(alpha: 0.5),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              _StatusBadge(status: entry.proposalStatus),
              const Spacer(),
              Text(
                _formatDate(entry.completedAt ?? entry.attributedAt),
                style: theme.textTheme.labelSmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ),
          if (entry.proposalTitle.isNotEmpty) ...[
            const SizedBox(height: 6),
            Text(
              entry.proposalTitle,
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
          ],
          const SizedBox(height: 4),
          Text(
            entry.providerName.isNotEmpty
                ? entry.providerName
                : entry.providerId,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 8),
          if (hasReview)
            _ReviewBlock(
              rating: entry.rating!,
              comment: entry.comment,
              reviewedAt: entry.reviewedAt,
            )
          else
            Container(
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(10),
                color: theme.colorScheme.surfaceContainerHigh,
              ),
              child: Row(
                children: [
                  Icon(
                    Icons.hourglass_empty,
                    size: 14,
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                  const SizedBox(width: 8),
                  Text(
                    l10n.reputationNoReviewBadge,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }

  String _formatDate(DateTime d) {
    final local = d.toLocal();
    final month = local.month.toString().padLeft(2, '0');
    final day = local.day.toString().padLeft(2, '0');
    return '${local.year}-$month-$day';
  }
}

class _ReviewBlock extends StatelessWidget {
  const _ReviewBlock({
    required this.rating,
    required this.comment,
    required this.reviewedAt,
  });

  final int rating;
  final String comment;
  final DateTime? reviewedAt;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            for (var i = 1; i <= 5; i++)
              Icon(
                Icons.star_rounded,
                size: 16,
                color: i <= rating
                    ? const Color(0xFFF59E0B)
                    : theme.colorScheme.onSurfaceVariant
                        .withValues(alpha: 0.4),
              ),
            if (reviewedAt != null) ...[
              const Spacer(),
              Text(
                _formatDate(reviewedAt!),
                style: theme.textTheme.labelSmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ],
        ),
        if (comment.isNotEmpty) ...[
          const SizedBox(height: 6),
          Text(
            comment,
            style: theme.textTheme.bodyMedium?.copyWith(height: 1.4),
          ),
        ],
      ],
    );
  }

  String _formatDate(DateTime d) {
    final local = d.toLocal();
    final month = local.month.toString().padLeft(2, '0');
    final day = local.day.toString().padLeft(2, '0');
    return '${local.year}-$month-$day';
  }
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final (label, bg, fg) = _styleForStatus(status, theme, l10n);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(
          color: fg,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  (String, Color, Color) _styleForStatus(
    String status,
    ThemeData theme,
    AppLocalizations l10n,
  ) {
    switch (status) {
      case 'completed':
        return (
          l10n.reputationStatusCompleted,
          const Color(0xFFD1FAE5),
          const Color(0xFF047857),
        );
      case 'disputed':
        return (
          l10n.reputationStatusDisputed,
          theme.colorScheme.errorContainer,
          theme.colorScheme.onErrorContainer,
        );
      case 'active':
      case 'paid':
      case 'accepted':
      case 'completion_requested':
        return (
          l10n.reputationStatusActive,
          const Color(0xFFDBEAFE),
          const Color(0xFF1D4ED8),
        );
      case 'pending':
        return (
          l10n.reputationStatusPending,
          theme.colorScheme.surfaceContainerHighest,
          theme.colorScheme.onSurfaceVariant,
        );
      default:
        return (
          l10n.reputationStatusOther(status),
          theme.colorScheme.surfaceContainerHighest,
          theme.colorScheme.onSurfaceVariant,
        );
    }
  }
}
