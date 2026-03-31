import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../providers/job_provider.dart';
import '../widgets/candidate_card.dart';

class CandidatesScreen extends ConsumerWidget {
  const CandidatesScreen({super.key, required this.jobId});

  final String jobId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final candidates = ref.watch(jobApplicationsProvider(jobId));

    return Scaffold(
      appBar: AppBar(title: const Text('Candidatures')),
      body: RefreshIndicator(
        onRefresh: () async => ref.invalidate(jobApplicationsProvider(jobId)),
        child: candidates.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (e, _) => Center(child: Text('Erreur: $e')),
          data: (items) {
            if (items.isEmpty) {
              return ListView(
                children: [
                  SizedBox(height: MediaQuery.of(context).size.height * 0.3),
                  const Icon(Icons.people_outline, size: 48, color: Colors.grey),
                  const SizedBox(height: 12),
                  const Text(
                    'Aucune candidature pour le moment',
                    textAlign: TextAlign.center,
                    style: TextStyle(color: Colors.grey),
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
                onContact: (conversationId) {
                  context.push('/messaging?conversation=$conversationId');
                },
              ),
            );
          },
        ),
      ),
    );
  }
}
