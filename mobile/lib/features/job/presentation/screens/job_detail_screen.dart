import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../domain/entities/job_entity.dart';
import '../providers/job_provider.dart';
import '../widgets/candidate_card.dart';

/// Detail screen for a job the current user owns (two tabs: details + candidates).
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

    return Scaffold(
      appBar: AppBar(
        title: Text(
          widget.job.title,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.edit_outlined),
            tooltip: l10n.jobEditJob,
            onPressed: () =>
                context.push(RoutePaths.jobEdit, extra: widget.jobId),
          ),
          _JobPopupMenu(
            job: widget.job,
            jobId: widget.jobId,
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
          _DetailsTab(job: widget.job),
          _CandidatesTab(jobId: widget.jobId),
        ],
      ),
    );
  }
}

class _JobPopupMenu extends ConsumerWidget {
  const _JobPopupMenu({
    required this.job,
    required this.jobId,
    required this.onRefresh,
  });

  final JobEntity job;
  final String jobId;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return PopupMenuButton<String>(
      onSelected: (value) => _onSelected(context, ref, value),
      itemBuilder: (context) => [
        if (job.isOpen)
          PopupMenuItem(
            value: 'close',
            child: Row(
              children: [
                const Icon(Icons.block, size: 18),
                const SizedBox(width: 8),
                Text(l10n.jobClose),
              ],
            ),
          )
        else
          PopupMenuItem(
            value: 'reopen',
            child: Row(
              children: [
                const Icon(Icons.refresh, size: 18),
                const SizedBox(width: 8),
                Text(l10n.jobReopen),
              ],
            ),
          ),
        PopupMenuItem(
          value: 'delete',
          child: Row(
            children: [
              Icon(Icons.delete_outline, size: 18, color: theme.colorScheme.error),
              const SizedBox(width: 8),
              Text(l10n.jobDelete, style: TextStyle(color: theme.colorScheme.error)),
            ],
          ),
        ),
      ],
    );
  }

  Future<void> _onSelected(
    BuildContext context,
    WidgetRef ref,
    String value,
  ) async {
    final l10n = AppLocalizations.of(context)!;

    switch (value) {
      case 'close':
        final ok = await closeJobAction(ref, jobId);
        if (!context.mounted) return;
        if (ok) {
          onRefresh();
        } else {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(l10n.unexpectedError)),
          );
        }
      case 'reopen':
        final ok = await reopenJobAction(ref, jobId);
        if (!context.mounted) return;
        if (ok) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(l10n.jobReopenSuccess)),
          );
          onRefresh();
        } else {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(l10n.unexpectedError)),
          );
        }
      case 'delete':
        await _confirmDelete(context, ref);
    }
  }

  Future<void> _confirmDelete(BuildContext context, WidgetRef ref) async {
    final l10n = AppLocalizations.of(context)!;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.jobDelete),
        content: Text(l10n.jobDeleteConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(l10n.jobCancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            child: Text(l10n.jobDelete),
          ),
        ],
      ),
    );

    if (confirmed != true || !context.mounted) return;

    final ok = await deleteJobAction(ref, jobId);
    if (!context.mounted) return;

    if (ok) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.jobDeleteSuccess)),
      );
      context.pop();
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.unexpectedError)),
      );
    }
  }
}

class _DetailsTab extends StatelessWidget {
  const _DetailsTab({required this.job});

  final JobEntity job;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Header: status + date + applicant type
          _JobHeaderCard(job: job),
          const SizedBox(height: 16),

          // Video player
          if (_hasVideo) ...[
            VideoPlayerWidget(videoUrl: job.videoUrl!),
            const SizedBox(height: 16),
          ],

          // Budget card
          _BudgetCard(job: job),
          const SizedBox(height: 16),

          // Skills
          if (job.skills.isNotEmpty) ...[
            Text(
              l10n.jobSkills,
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 4,
              children: job.skills
                  .map((s) => Chip(
                        label: Text(s, style: const TextStyle(fontSize: 12)),
                        backgroundColor: theme.colorScheme.primary
                            .withValues(alpha: 0.08),
                        side: BorderSide.none,
                        visualDensity: VisualDensity.compact,
                        materialTapTargetSize:
                            MaterialTapTargetSize.shrinkWrap,
                      ),)
                  .toList(),
            ),
            const SizedBox(height: 16),
          ],

          // Description
          Text(
            l10n.jobDescription,
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w600,
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

  bool get _hasVideo =>
      job.videoUrl != null && job.videoUrl!.isNotEmpty;
}

class _JobHeaderCard extends StatelessWidget {
  const _JobHeaderCard({required this.job});

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
              // Status badge
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
              // Applicant type
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
          // Date
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

class _BudgetCard extends StatelessWidget {
  const _BudgetCard({required this.job});

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
                '${job.minBudget}\u20ac - ${job.maxBudget}\u20ac',
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.bold,
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

class _CandidatesTab extends ConsumerWidget {
  const _CandidatesTab({required this.jobId});
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
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            const Icon(Icons.error_outline, size: 48, color: Colors.grey),
            const SizedBox(height: 12),
            Text(l10n.somethingWentWrong, style: const TextStyle(color: Colors.grey)),
            const SizedBox(height: 8),
            TextButton(
              onPressed: () => ref.invalidate(jobApplicationsProvider(jobId)),
              child: Text(l10n.retry),
            ),
          ]),
        ),
        data: (items) {
          if (items.isEmpty) {
            return ListView(children: [
              SizedBox(height: MediaQuery.of(context).size.height * 0.25),
              Icon(Icons.people_outline, size: 48, color: theme.colorScheme.onSurface.withValues(alpha: 0.3)),
              const SizedBox(height: 12),
              Text(l10n.jobNoCandidates, textAlign: TextAlign.center, style: theme.textTheme.titleMedium),
              const SizedBox(height: 4),
              Text(l10n.jobNoCandidatesDesc, textAlign: TextAlign.center,
                style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurface.withValues(alpha: 0.5)),
              ),
            ]);
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
