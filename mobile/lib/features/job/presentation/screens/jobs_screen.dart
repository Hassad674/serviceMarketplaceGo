import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/job_entity.dart';
import '../providers/job_provider.dart';
import '../widgets/job_list/job_list_card.dart';
import '../widgets/job_list/job_list_states.dart';

/// M-07 — Mes annonces (entreprise listing). Soleil v2 visual port.
///
/// Editorial header (Fraunces title with italic corail accent + tabac
/// subtitle) inside a [SliverAppBar], a corail FilledButton "Publier
/// une annonce" pill in the body, then a vertical list of job cards.
/// Empty / loading / error states delegate to the shared widgets in
/// [JobListEmptyState] / [JobListSkeleton] / [JobListErrorState] — the
/// public surface of those widgets is preserved so the existing widget
/// tests still pass.
class JobsScreen extends ConsumerWidget {
  const JobsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final jobsAsync = ref.watch(myJobsProvider);
    final canCreate = ref.watch(hasPermissionProvider(OrgPermission.jobsCreate));

    return Scaffold(
      backgroundColor: theme.colorScheme.surfaceContainerLowest,
      appBar: AppBar(
        backgroundColor: theme.colorScheme.surfaceContainerLowest,
        scrolledUnderElevation: 0,
        elevation: 0,
        leading: IconButton(
          icon: Icon(
            Icons.menu_rounded,
            color: theme.colorScheme.onSurface,
            size: 22,
          ),
          onPressed: openShellDrawer,
        ),
        title: Text(
          l10n.jobMyJobs,
          style: SoleilTextStyles.titleLarge.copyWith(
            color: theme.colorScheme.onSurface,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      body: SafeArea(
        top: false,
        child: jobsAsync.when(
          loading: () => const _LoadingView(),
          error: (error, _) => _ErrorView(
            message: l10n.unexpectedError,
            onRetry: () => ref.invalidate(myJobsProvider),
          ),
          data: (jobs) => jobs.isEmpty
              ? _EmptyView(canCreate: canCreate)
              : _JobsListView(jobs: jobs, canCreate: canCreate),
        ),
      ),
    );
  }
}

/// Editorial header — corail mono eyebrow + Fraunces title with italic
/// corail accent + tabac subtitle. Reused by every state branch (loading,
/// error, empty, list) so the page's visual identity stays consistent.
class _PageHeader extends StatelessWidget {
  const _PageHeader();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;

    final base = SoleilTextStyles.headlineLarge.copyWith(
      color: colorScheme.onSurface,
      fontWeight: FontWeight.w500,
      letterSpacing: -0.5,
      height: 1.1,
    );

    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 4, 20, 14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.jobsEyebrow,
            style: SoleilTextStyles.mono.copyWith(
              color: colorScheme.primary,
              fontSize: 10.5,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.5,
            ),
          ),
          const SizedBox(height: 6),
          RichText(
            text: TextSpan(
              style: base,
              children: [
                TextSpan(text: '${l10n.jobsTitlePrefix} '),
                TextSpan(
                  text: l10n.jobsTitleAccent,
                  style: base.copyWith(
                    fontStyle: FontStyle.italic,
                    color: colorScheme.primary,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 6),
          Text(
            l10n.jobsSubtitle,
            style: SoleilTextStyles.body.copyWith(
              color: colorScheme.onSurfaceVariant,
              fontSize: 13.5,
              height: 1.5,
            ),
          ),
        ],
      ),
    );
  }
}

/// Corail "Publier une annonce" pill button. Anchored under the header
/// when the user can create.
class _PublishPill extends StatelessWidget {
  const _PublishPill();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;

    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 0, 20, 12),
      child: Align(
        alignment: Alignment.centerLeft,
        child: FilledButton.icon(
          onPressed: () => GoRouter.of(context).push(RoutePaths.jobsCreate),
          style: FilledButton.styleFrom(
            backgroundColor: colorScheme.primary,
            foregroundColor: colorScheme.onPrimary,
            padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 12),
            shape: const StadiumBorder(),
            elevation: 0,
            textStyle: SoleilTextStyles.button.copyWith(
              fontWeight: FontWeight.w700,
              fontSize: 13.5,
            ),
          ),
          icon: const Icon(Icons.add_rounded, size: 18),
          label: Text(l10n.jobCreateJob),
        ),
      ),
    );
  }
}

/// List view — header on top, optional publish pill, then cards.
class _JobsListView extends StatelessWidget {
  const _JobsListView({required this.jobs, required this.canCreate});

  final List<JobEntity> jobs;
  final bool canCreate;

  @override
  Widget build(BuildContext context) {
    return CustomScrollView(
      slivers: [
        const SliverToBoxAdapter(child: _PageHeader()),
        if (canCreate) const SliverToBoxAdapter(child: _PublishPill()),
        SliverPadding(
          padding: const EdgeInsets.fromLTRB(16, 4, 16, 24),
          sliver: SliverList.separated(
            itemCount: jobs.length,
            separatorBuilder: (_, __) => const SizedBox(height: 10),
            itemBuilder: (context, index) => JobListCard(job: jobs[index]),
          ),
        ),
      ],
    );
  }
}

/// Loading view — preserves the JobListSkeleton public surface to keep
/// the existing widget tests passing, but anchors it under the editorial
/// header so the visual identity stays consistent.
class _LoadingView extends StatelessWidget {
  const _LoadingView();

  @override
  Widget build(BuildContext context) {
    return const CustomScrollView(
      slivers: [
        SliverToBoxAdapter(child: _PageHeader()),
        SliverFillRemaining(
          hasScrollBody: false,
          child: JobListSkeleton(),
        ),
      ],
    );
  }
}

/// Empty view — anchors [JobListEmptyState] (test-locked widget) under
/// the editorial header and adds the corail "Publier ta première annonce"
/// pill below it when the user can create.
class _EmptyView extends StatelessWidget {
  const _EmptyView({required this.canCreate});

  final bool canCreate;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final colorScheme = Theme.of(context).colorScheme;

    return CustomScrollView(
      slivers: [
        const SliverToBoxAdapter(child: _PageHeader()),
        SliverFillRemaining(
          hasScrollBody: false,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const JobListEmptyState(),
                if (canCreate) ...[
                  const SizedBox(height: 16),
                  FilledButton.icon(
                    onPressed: () =>
                        GoRouter.of(context).push(RoutePaths.jobsCreate),
                    style: FilledButton.styleFrom(
                      backgroundColor: colorScheme.primary,
                      foregroundColor: colorScheme.onPrimary,
                      padding: const EdgeInsets.symmetric(
                        horizontal: 18,
                        vertical: 12,
                      ),
                      shape: const StadiumBorder(),
                      elevation: 0,
                      textStyle: SoleilTextStyles.button.copyWith(
                        fontWeight: FontWeight.w700,
                        fontSize: 13.5,
                      ),
                    ),
                    icon: const Icon(Icons.add_rounded, size: 18),
                    label: Text(l10n.jobsEmptyCta),
                  ),
                ],
              ],
            ),
          ),
        ),
      ],
    );
  }
}

/// Error view — wraps [JobListErrorState] under the editorial header.
class _ErrorView extends StatelessWidget {
  const _ErrorView({
    required this.message,
    required this.onRetry,
  });

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return CustomScrollView(
      slivers: [
        const SliverToBoxAdapter(child: _PageHeader()),
        SliverFillRemaining(
          hasScrollBody: false,
          child: JobListErrorState(message: message, onRetry: onRetry),
        ),
      ],
    );
  }
}
