import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/job_entity.dart';
import '../providers/job_provider.dart';
import '../widgets/job_detail/job_detail_popup_menu.dart';
import '../widgets/job_detail/job_detail_tabs.dart';

/// M-08 — Détail annonce (entreprise). Soleil v2 visual port.
///
/// AppBar: Fraunces title, edit pen icon top-right + popup menu
/// (close/reopen/delete). Body: editorial corail eyebrow, pill tab
/// bar (Description / Candidatures (N)) and a TabBarView whose two
/// tabs are owned by `JobDetailDetailsTab` and
/// `JobDetailCandidatesTab` (Soleil-styled in
/// `widgets/job_detail/job_detail_tabs.dart`).
///
/// The repository / providers / navigation behaviour are unchanged.
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
  int _activeTab = 0;

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
    if (_activeTab != _tabController.index) {
      setState(() => _activeTab = _tabController.index);
    }
    if (_tabController.index == 1 && !_markedViewed) {
      _markedViewed = true;
      markApplicationsViewedAction(ref, widget.jobId);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;
    final canEdit = ref.watch(hasPermissionProvider(OrgPermission.jobsEdit));
    final canDelete =
        ref.watch(hasPermissionProvider(OrgPermission.jobsDelete));
    final candidatesAsync = ref.watch(jobApplicationsProvider(widget.jobId));
    final candidatesCount = candidatesAsync.valueOrNull?.length ?? 0;

    return Scaffold(
      backgroundColor: cs.surface,
      appBar: AppBar(
        backgroundColor: cs.surfaceContainerLowest,
        scrolledUnderElevation: 0,
        elevation: 0,
        title: Text(
          widget.job.title,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: SoleilTextStyles.titleLarge.copyWith(
            color: cs.onSurface,
            fontSize: 18,
            fontWeight: FontWeight.w600,
          ),
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
      ),
      body: SafeArea(
        top: false,
        child: Column(
          children: [
            _DetailEyebrow(isOpen: widget.job.isOpen),
            const SizedBox(height: 8),
            _PillTabs(
              controller: _tabController,
              activeIndex: _activeTab,
              candidatesCount: candidatesCount,
            ),
            const SizedBox(height: 4),
            Expanded(
              child: TabBarView(
                controller: _tabController,
                children: [
                  JobDetailDetailsTab(job: widget.job),
                  JobDetailCandidatesTab(jobId: widget.jobId),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _DetailEyebrow extends StatelessWidget {
  const _DetailEyebrow({required this.isOpen});

  final bool isOpen;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;
    final eyebrow = isOpen
        ? l10n.jobDetail_m08_eyebrowOpen
        : l10n.jobDetail_m08_eyebrowClosed;

    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 0),
      child: Align(
        alignment: Alignment.centerLeft,
        child: Text(
          eyebrow,
          style: SoleilTextStyles.mono.copyWith(
            color: cs.primary,
            fontSize: 10.5,
            fontWeight: FontWeight.w700,
            letterSpacing: 1.5,
          ),
        ),
      ),
    );
  }
}

class _PillTabs extends StatelessWidget {
  const _PillTabs({
    required this.controller,
    required this.activeIndex,
    required this.candidatesCount,
  });

  final TabController controller;
  final int activeIndex;
  final int candidatesCount;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;

    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 12, 20, 0),
      child: Container(
        padding: const EdgeInsets.all(4),
        decoration: BoxDecoration(
          color: cs.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(AppTheme.radiusFull),
          border: Border.all(color: cs.outline),
        ),
        child: Row(
          children: [
            Expanded(
              child: _PillTab(
                label: l10n.jobDetail_m08_tabDescription,
                isActive: activeIndex == 0,
                onTap: () => controller.animateTo(0),
              ),
            ),
            Expanded(
              child: _PillTab(
                label:
                    '${l10n.jobDetail_m08_tabCandidates} ($candidatesCount)',
                isActive: activeIndex == 1,
                onTap: () => controller.animateTo(1),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _PillTab extends StatelessWidget {
  const _PillTab({
    required this.label,
    required this.isActive,
    required this.onTap,
  });

  final String label;
  final bool isActive;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final soleil = theme.extension<AppColors>()!;

    return Material(
      color: isActive ? soleil.accentSoft : Colors.transparent,
      borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 9, horizontal: 14),
          child: Center(
            child: Text(
              label,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: SoleilTextStyles.button.copyWith(
                color: isActive ? soleil.primaryDeep : cs.onSurfaceVariant,
                fontSize: 13,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
