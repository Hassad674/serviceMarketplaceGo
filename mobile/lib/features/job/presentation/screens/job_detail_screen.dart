import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/job_entity.dart';
import '../providers/job_provider.dart';
import '../widgets/job_detail/job_detail_popup_menu.dart';
import '../widgets/job_detail/job_detail_tabs.dart';

/// Detail screen for a job the current user owns (two tabs: details +
/// candidates).
class JobDetailScreen extends ConsumerStatefulWidget {
  const JobDetailScreen({super.key, required this.jobId});

  final String jobId;

  @override
  ConsumerState<JobDetailScreen> createState() => _JobDetailScreenState();
}

class _JobDetailScreenState extends ConsumerState<JobDetailScreen> {
  late Future<JobEntity> _jobFuture;

  @override
  void initState() {
    super.initState();
    _jobFuture = ref.read(jobRepositoryProvider).getJob(widget.jobId);
  }

  void _refreshJob() {
    setState(() {
      _jobFuture = ref.read(jobRepositoryProvider).getJob(widget.jobId);
    });
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<JobEntity>(
      future: _jobFuture,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return Scaffold(
            appBar: AppBar(),
            body: const Center(child: CircularProgressIndicator()),
          );
        }
        if (snapshot.hasError || !snapshot.hasData) {
          final l10n = AppLocalizations.of(context)!;
          return Scaffold(
            appBar: AppBar(),
            body: Center(child: Text(l10n.jobNotFound)),
          );
        }
        return _JobDetailBody(
          job: snapshot.data!,
          jobId: widget.jobId,
          onRefresh: _refreshJob,
        );
      },
    );
  }
}

class _JobDetailBody extends ConsumerStatefulWidget {
  const _JobDetailBody({
    required this.job,
    required this.jobId,
    required this.onRefresh,
  });

  final JobEntity job;
  final String jobId;
  final VoidCallback onRefresh;

  @override
  ConsumerState<_JobDetailBody> createState() => _JobDetailBodyState();
}

class _JobDetailBodyState extends ConsumerState<_JobDetailBody>
    with SingleTickerProviderStateMixin {
  late final TabController _tabController;
  bool _markedViewed = false;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _tabController.addListener(_onTabChanged);
  }

  @override
  void dispose() {
    _tabController.removeListener(_onTabChanged);
    _tabController.dispose();
    super.dispose();
  }

  void _onTabChanged() {
    if (_tabController.index == 1 && !_markedViewed) {
      _markedViewed = true;
      markApplicationsViewedAction(ref, widget.jobId);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final canEdit = ref.watch(hasPermissionProvider(OrgPermission.jobsEdit));
    final canDelete = ref.watch(hasPermissionProvider(OrgPermission.jobsDelete));

    return Scaffold(
      appBar: AppBar(
        title: Text(
          widget.job.title,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        actions: [
          if (canEdit)
            IconButton(
              icon: const Icon(Icons.edit_outlined),
              tooltip: l10n.jobEditJob,
              onPressed: () =>
                  context.push(RoutePaths.jobEdit, extra: widget.jobId),
            ),
          if (canEdit || canDelete)
            JobDetailPopupMenu(
              job: widget.job,
              jobId: widget.jobId,
              canEdit: canEdit,
              canDelete: canDelete,
              onRefresh: widget.onRefresh,
            ),
        ],
        bottom: TabBar(
          controller: _tabController,
          tabs: [
            Tab(text: l10n.jobOfferDetails),
            Tab(text: l10n.jobCandidates),
          ],
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          JobDetailDetailsTab(job: widget.job),
          JobDetailCandidatesTab(jobId: widget.jobId),
        ],
      ),
    );
  }
}
