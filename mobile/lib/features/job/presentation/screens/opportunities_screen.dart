import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../providers/job_provider.dart';
import '../widgets/opportunity_card.dart';

class OpportunitiesScreen extends ConsumerWidget {
  const OpportunitiesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final openJobs = ref.watch(openJobsProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Opportunit\u00e9s')),
      body: RefreshIndicator(
        onRefresh: () async => ref.invalidate(openJobsProvider),
        child: openJobs.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (e, _) => Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('Erreur: $e', style: Theme.of(context).textTheme.bodyMedium),
                const SizedBox(height: 12),
                FilledButton(
                  onPressed: () => ref.invalidate(openJobsProvider),
                  child: const Text('R\u00e9essayer'),
                ),
              ],
            ),
          ),
          data: (jobs) {
            if (jobs.isEmpty) {
              return ListView(
                children: [
                  SizedBox(height: MediaQuery.of(context).size.height * 0.3),
                  const Icon(Icons.work_off_outlined, size: 48, color: Colors.grey),
                  const SizedBox(height: 12),
                  const Text(
                    'Aucune opportunit\u00e9 pour le moment',
                    textAlign: TextAlign.center,
                    style: TextStyle(color: Colors.grey),
                  ),
                ],
              );
            }
            return ListView.separated(
              padding: const EdgeInsets.all(16),
              itemCount: jobs.length,
              separatorBuilder: (_, __) => const SizedBox(height: 12),
              itemBuilder: (context, index) => OpportunityCard(job: jobs[index]),
            );
          },
        ),
      ),
    );
  }
}
