import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/job_entity.dart';
import '../providers/job_provider.dart';
import '../widgets/job_list/job_list_card.dart';
import '../widgets/job_list/job_list_states.dart';

/// Lists the current user's jobs, fetched from the backend.
///
/// Displays an empty state with a clear CTA when no jobs exist.
/// A FAB navigates to the Create Job screen.
class JobsScreen extends ConsumerWidget {
  const JobsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final jobsAsync = ref.watch(myJobsProvider);
    final canCreate = ref.watch(hasPermissionProvider(OrgPermission.jobsCreate));

    return Scaffold(
      appBar: AppBar(
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.jobMyJobs),
      ),
      body: SafeArea(
        child: jobsAsync.when(
          loading: () => const JobListSkeleton(),
          error: (error, _) => JobListErrorState(
            message: l10n.unexpectedError,
            onRetry: () => ref.invalidate(myJobsProvider),
          ),
          data: (jobs) => jobs.isEmpty
              ? const JobListEmptyState()
              : _JobListView(jobs: jobs),
        ),
      ),
      floatingActionButton: canCreate
          ? FloatingActionButton(
              onPressed: () => context.push(RoutePaths.jobsCreate),
              backgroundColor: theme.colorScheme.primary,
              foregroundColor: Colors.white,
              child: const Icon(Icons.add),
            )
          : null,
    );
  }
}

class _JobListView extends StatelessWidget {
  const _JobListView({required this.jobs});

  final List<JobEntity> jobs;

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: jobs.length,
      separatorBuilder: (_, __) => const SizedBox(height: 12),
      itemBuilder: (context, index) => JobListCard(job: jobs[index]),
    );
  }
}
