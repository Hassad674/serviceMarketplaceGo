import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../domain/entities/job_entity.dart';
import '../providers/job_provider.dart';
import '../widgets/apply_bottom_sheet.dart';

class OpportunityDetailScreen extends ConsumerWidget {
  const OpportunityDetailScreen({super.key, required this.jobId});

  final String jobId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final hasApplied = ref.watch(hasAppliedProvider(jobId));
    final l10n = AppLocalizations.of(context)!;

    return FutureBuilder<JobEntity>(
      future: ref.read(jobRepositoryProvider).getJob(jobId),
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return Scaffold(appBar: AppBar(), body: const Center(child: CircularProgressIndicator()));
        }
        if (snapshot.hasError || !snapshot.hasData) {
          return Scaffold(appBar: AppBar(), body: Center(child: Text(l10n.jobNotFound)));
        }
        final job = snapshot.data!;
        final alreadyApplied = hasApplied.valueOrNull ?? false;

        return Scaffold(
          appBar: AppBar(title: Text(job.title, maxLines: 1, overflow: TextOverflow.ellipsis)),
          body: SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Video player
                if (job.videoUrl != null && job.videoUrl!.isNotEmpty) ...[
                  VideoPlayerWidget(videoUrl: job.videoUrl!),
                  const SizedBox(height: 16),
                ],

                // Budget
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Row(
                      children: [
                        const Icon(Icons.euro, color: Color(0xFFF43F5E)),
                        const SizedBox(width: 12),
                        Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              '${job.minBudget}\u20ac - ${job.maxBudget}\u20ac',
                              style: Theme.of(context).textTheme.titleMedium?.copyWith(
                                fontWeight: FontWeight.bold,
                                color: Theme.of(context).colorScheme.onSurface,
                              ),
                            ),
                            Text(
                              job.budgetType == 'one_shot' ? l10n.budgetTypeOneShot : l10n.budgetTypeLongTerm,
                              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                                color: Theme.of(context).colorScheme.onSurfaceVariant,
                              ),
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 16),
                // Skills
                if (job.skills.isNotEmpty) ...[
                  Text(
                    l10n.requiredSkills,
                    style: Theme.of(context).textTheme.titleSmall?.copyWith(
                      color: Theme.of(context).colorScheme.onSurface,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Wrap(
                    spacing: 8,
                    runSpacing: 4,
                    children: job.skills.map((s) => Chip(
                      label: Text(
                        s,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.onSurface,
                        ),
                      ),
                      backgroundColor: const Color(0xFFFFF1F2),
                    )).toList(),
                  ),
                  const SizedBox(height: 16),
                ],
                // Description
                Text(
                  l10n.jobDescription,
                  style: Theme.of(context).textTheme.titleSmall?.copyWith(
                    color: Theme.of(context).colorScheme.onSurface,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  job.description,
                  style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: Theme.of(context).colorScheme.onSurface,
                  ),
                ),
              ],
            ),
          ),
          bottomNavigationBar: SafeArea(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: FilledButton(
                onPressed: alreadyApplied ? null : () => showApplyBottomSheet(context, ref, jobId),
                style: FilledButton.styleFrom(
                  backgroundColor: alreadyApplied ? Colors.grey : const Color(0xFFF43F5E),
                  minimumSize: const Size.fromHeight(48),
                ),
                child: Text(alreadyApplied ? l10n.alreadyApplied : l10n.applyAction),
              ),
            ),
          ),
        );
      },
    );
  }
}
