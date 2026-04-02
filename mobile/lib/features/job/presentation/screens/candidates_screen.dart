import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../providers/job_provider.dart';
import '../widgets/candidate_card.dart';

class CandidatesScreen extends ConsumerWidget {
  const CandidatesScreen({super.key, required this.jobId});

  final String jobId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final candidates = ref.watch(jobApplicationsProvider(jobId));
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.applications)),
      body: RefreshIndicator(
        onRefresh: () async => ref.invalidate(jobApplicationsProvider(jobId)),
        child: candidates.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (e, _) => Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Icon(Icons.error_outline, size: 48, color: Colors.grey),
                const SizedBox(height: 12),
                Text(l10n.somethingWentWrong, style: const TextStyle(color: Colors.grey)),
                const SizedBox(height: 8),
                TextButton(
                  onPressed: () => ref.invalidate(jobApplicationsProvider(jobId)),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
          data: (items) {
            if (items.isEmpty) {
              return ListView(
                children: [
                  SizedBox(height: MediaQuery.of(context).size.height * 0.3),
                  const Icon(Icons.people_outline, size: 48, color: Colors.grey),
                  const SizedBox(height: 12),
                  Text(
                    l10n.noApplicationsYet,
                    textAlign: TextAlign.center,
                    style: const TextStyle(color: Colors.grey),
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
              ),
            );
          },
        ),
      ),
    );
  }
}
