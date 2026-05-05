import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../domain/entities/job_entity.dart';
import '../../providers/job_provider.dart';
import '../candidate_card.dart';
import 'job_detail_cards.dart';

/// M-08 description tab — Soleil v2 layout.
///
/// Stacked Soleil cards: status header card, budget hero, optional
/// video, full description body, skills as corail-soft pills, and a
/// dedicated duration card. Editorial calm, no neon.
class JobDetailDetailsTab extends StatelessWidget {
  const JobDetailDetailsTab({super.key, required this.job});

  final JobEntity job;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final soleil = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    final hasVideo = job.videoUrl != null && job.videoUrl!.isNotEmpty;

    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 28),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          JobDetailHeaderCard(job: job),
          const SizedBox(height: 16),
          JobDetailBudgetCard(job: job),
          const SizedBox(height: 16),
          if (hasVideo) ...[
            _SectionCard(
              heading: l10n.jobDetail_m08_videoHeading,
              child: ClipRRect(
                borderRadius: BorderRadius.circular(AppTheme.radiusLg),
                child: AspectRatio(
                  aspectRatio: 16 / 9,
                  child: ColoredBox(
                    color: cs.onSurface,
                    child: const Icon(
                      Icons.play_circle_outline,
                      color: Colors.white,
                      size: 48,
                    ),
                  ),
                ),
              ),
            ),
            const SizedBox(height: 16),
          ],
          _SectionCard(
            heading: l10n.jobDetail_m08_descriptionHeading,
            child: Text(
              job.description,
              style: SoleilTextStyles.bodyLarge.copyWith(
                color: cs.onSurface,
                height: 1.6,
              ),
            ),
          ),
          if (job.skills.isNotEmpty) ...[
            const SizedBox(height: 16),
            _SectionCard(
              heading: l10n.jobDetail_m08_skillsHeading,
              child: Wrap(
                spacing: 8,
                runSpacing: 8,
                children: job.skills
                    .map(
                      (skill) => Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 12,
                          vertical: 6,
                        ),
                        decoration: BoxDecoration(
                          color: soleil.accentSoft,
                          borderRadius:
                              BorderRadius.circular(AppTheme.radiusFull),
                        ),
                        child: Text(
                          skill,
                          style: SoleilTextStyles.caption.copyWith(
                            fontSize: 12,
                            fontWeight: FontWeight.w600,
                            color: soleil.primaryDeep,
                          ),
                        ),
                      ),
                    )
                    .toList(),
              ),
            ),
          ],
          if (job.durationWeeks != null || job.isIndefinite) ...[
            const SizedBox(height: 16),
            _SectionCard(
              heading: l10n.jobDetail_m08_durationLabel,
              child: Text(
                job.isIndefinite
                    ? l10n.jobDetail_m08_durationIndefinite
                    : l10n.jobDetail_m08_durationWeeks(
                        job.durationWeeks ?? 0,
                      ),
                style: SoleilTextStyles.bodyLarge.copyWith(
                  color: cs.onSurface,
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _SectionCard extends StatelessWidget {
  const _SectionCard({required this.heading, required this.child});

  final String heading;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;

    return Container(
      width: double.infinity,
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
          Text(
            heading,
            style: SoleilTextStyles.titleMedium.copyWith(
              color: cs.onSurface,
              fontSize: 16,
            ),
          ),
          const SizedBox(height: 12),
          child,
        ],
      ),
    );
  }
}

/// M-08 candidates tab — list of Soleil candidate cards or a soft empty
/// state. Underlying Riverpod provider is unchanged.
class JobDetailCandidatesTab extends ConsumerWidget {
  const JobDetailCandidatesTab({super.key, required this.jobId});

  final String jobId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final candidates = ref.watch(jobApplicationsProvider(jobId));

    return RefreshIndicator(
      onRefresh: () async => ref.invalidate(jobApplicationsProvider(jobId)),
      child: candidates.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _CandidatesError(jobId: jobId),
        data: (items) {
          if (items.isEmpty) return const _CandidatesEmpty();
          return ListView.separated(
            padding: const EdgeInsets.fromLTRB(20, 16, 20, 28),
            itemCount: items.length,
            separatorBuilder: (_, __) => const SizedBox(height: 12),
            itemBuilder: (context, index) => CandidateCard(
              item: items[index],
              jobId: jobId,
              candidates: items,
              candidateIndex: index,
            ),
          );
        },
      ),
    );
  }
}

class _CandidatesError extends ConsumerWidget {
  const _CandidatesError({required this.jobId});

  final String jobId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 60, 20, 28),
      children: [
        Icon(
          Icons.error_outline,
          size: 48,
          color: theme.colorScheme.onSurfaceVariant,
        ),
        const SizedBox(height: 12),
        Text(
          l10n.somethingWentWrong,
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 8),
        Center(
          child: TextButton(
            onPressed: () => ref.invalidate(jobApplicationsProvider(jobId)),
            child: Text(l10n.retry),
          ),
        ),
      ],
    );
  }
}

class _CandidatesEmpty extends StatelessWidget {
  const _CandidatesEmpty();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final soleil = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 40, 20, 28),
      children: [
        Container(
          padding: const EdgeInsets.all(24),
          decoration: BoxDecoration(
            color: soleil.accentSoft,
            borderRadius: BorderRadius.circular(AppTheme.radius2xl),
            border: Border.all(color: cs.outline),
          ),
          child: Column(
            children: [
              Container(
                width: 56,
                height: 56,
                decoration: BoxDecoration(
                  color: cs.surfaceContainerLowest,
                  borderRadius: BorderRadius.circular(AppTheme.radiusFull),
                ),
                child: Icon(
                  Icons.groups_outlined,
                  color: soleil.primaryDeep,
                  size: 28,
                ),
              ),
              const SizedBox(height: 16),
              Text(
                l10n.jobDetail_m08_emptyTitle,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.titleMedium.copyWith(
                  color: cs.onSurface,
                  fontSize: 18,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                l10n.jobDetail_m08_emptyBody,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.body.copyWith(
                  color: cs.onSurfaceVariant,
                  height: 1.5,
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}
