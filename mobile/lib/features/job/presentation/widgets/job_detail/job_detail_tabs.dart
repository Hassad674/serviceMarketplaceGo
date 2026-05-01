import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../../../shared/widgets/video_player_widget.dart';
import '../../../domain/entities/job_entity.dart';
import '../../providers/job_provider.dart';
import '../candidate_card.dart';
import 'job_detail_cards.dart';

/// Read-only "Details" tab content.
class JobDetailDetailsTab extends StatelessWidget {
  const JobDetailDetailsTab({super.key, required this.job});

  final JobEntity job;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final hasVideo = job.videoUrl != null && job.videoUrl!.isNotEmpty;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          JobDetailHeaderCard(job: job),
          const SizedBox(height: 16),
          if (hasVideo) ...[
            VideoPlayerWidget(videoUrl: job.videoUrl!),
            const SizedBox(height: 16),
          ],
          JobDetailBudgetCard(job: job),
          const SizedBox(height: 16),
          if (job.skills.isNotEmpty) ...[
            Text(
              l10n.jobSkills,
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.w600,
                color: theme.colorScheme.onSurface,
              ),
            ),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 4,
              children: job.skills
                  .map(
                    (s) => Chip(
                      label: Text(
                        s,
                        style: TextStyle(
                          fontSize: 12,
                          color: theme.colorScheme.onSurface,
                        ),
                      ),
                      backgroundColor:
                          theme.colorScheme.primary.withValues(alpha: 0.08),
                      side: BorderSide.none,
                      visualDensity: VisualDensity.compact,
                      materialTapTargetSize:
                          MaterialTapTargetSize.shrinkWrap,
                    ),
                  )
                  .toList(),
            ),
            const SizedBox(height: 16),
          ],
          Text(
            l10n.jobDescription,
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w600,
              color: theme.colorScheme.onSurface,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            job.description,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: appColors?.mutedForeground,
              height: 1.5,
            ),
          ),
          const SizedBox(height: 32),
        ],
      ),
    );
  }
}

/// "Candidates" tab content — list of applicants.
class JobDetailCandidatesTab extends ConsumerWidget {
  const JobDetailCandidatesTab({super.key, required this.jobId});

  final String jobId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final candidates = ref.watch(jobApplicationsProvider(jobId));
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return RefreshIndicator(
      onRefresh: () async => ref.invalidate(jobApplicationsProvider(jobId)),
      child: candidates.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.error_outline, size: 48, color: Colors.grey),
              const SizedBox(height: 12),
              Text(
                l10n.somethingWentWrong,
                style: const TextStyle(color: Colors.grey),
              ),
              const SizedBox(height: 8),
              TextButton(
                onPressed: () =>
                    ref.invalidate(jobApplicationsProvider(jobId)),
                child: Text(l10n.retry),
              ),
            ],
          ),
        ),
        data: (items) {
          if (items.isEmpty) {
            return ListView(
              children: [
                SizedBox(height: MediaQuery.of(context).size.height * 0.25),
                Icon(
                  Icons.people_outline,
                  size: 48,
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.3),
                ),
                const SizedBox(height: 12),
                Text(
                  l10n.jobNoCandidates,
                  textAlign: TextAlign.center,
                  style: theme.textTheme.titleMedium,
                ),
                const SizedBox(height: 4),
                Text(
                  l10n.jobNoCandidatesDesc,
                  textAlign: TextAlign.center,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
                  ),
                ),
              ],
            );
          }
          return ListView.separated(
            padding: const EdgeInsets.all(16),
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
