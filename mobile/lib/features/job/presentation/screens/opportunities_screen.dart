import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shimmer/shimmer.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../domain/entities/job_entity.dart';
import '../providers/job_provider.dart';
import '../widgets/opportunity_card.dart';

/// W-12 mobile parity · Opportunités feed.
///
/// Mobile equivalent of the merged web `/opportunities` surface: two
/// tabs share the same screen — "Toutes les offres" (default) and "Mes
/// candidatures" (lazy-mounted on first activation). The applications
/// view's Riverpod query never fires until the user touches the second
/// tab, mirroring the TanStack `enabled` contract on web.
///
/// Hamburger leading icon is wired explicitly (via [openShellDrawer])
/// so the drawer is always reachable from this primary destination —
/// previously the screen rendered a default AppBar with no leading at
/// all (the user-reported bug).
class OpportunitiesScreen extends ConsumerStatefulWidget {
  const OpportunitiesScreen({super.key});

  @override
  ConsumerState<OpportunitiesScreen> createState() =>
      _OpportunitiesScreenState();
}

class _OpportunitiesScreenState extends ConsumerState<OpportunitiesScreen>
    with SingleTickerProviderStateMixin {
  late final TabController _tabController;
  // Lazy-mount latch: the applications view stays unbuilt until the
  // user touches the second tab. Once flipped to `true` the view stays
  // mounted so toggling back and forth is free.
  bool _applicationsTabEverActivated = false;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 2, vsync: this);
    _tabController.addListener(_handleTabChanged);
  }

  void _handleTabChanged() {
    // `indexIsChanging` fires twice per tap (start + end of animation);
    // we only need the post-change state.
    if (_tabController.indexIsChanging) return;
    if (_tabController.index == 1 && !_applicationsTabEverActivated) {
      setState(() => _applicationsTabEverActivated = true);
    }
  }

  @override
  void dispose() {
    _tabController
      ..removeListener(_handleTabChanged)
      ..dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final cs = theme.colorScheme;

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: Icon(Icons.menu_rounded, color: cs.onSurface, size: 22),
          onPressed: openShellDrawer,
          tooltip: MaterialLocalizations.of(context).openAppDrawerTooltip,
        ),
        title: Text(
          l10n.opportunities,
          style: SoleilTextStyles.titleLarge.copyWith(
            color: cs.onSurface,
            fontSize: 20,
          ),
        ),
        bottom: TabBar(
          controller: _tabController,
          labelColor: cs.primary,
          unselectedLabelColor: cs.onSurfaceVariant,
          indicatorColor: cs.primary,
          tabs: [
            Tab(text: l10n.opportunitiesTabAll),
            Tab(text: l10n.opportunitiesTabApplications),
          ],
        ),
      ),
      body: TabBarView(
        controller: _tabController,
        children: [
          const _AllOffersView(),
          // Lazy-mount: the applications view is only built once the
          // user has activated the tab at least once. Until then we
          // render a tiny placeholder so the TabBarView can keep its
          // page count without firing the underlying provider.
          _applicationsTabEverActivated
              ? const _MyApplicationsView()
              : const SizedBox.shrink(),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Tab 1 — All offers (the legacy opportunities feed)
// ---------------------------------------------------------------------------

class _AllOffersView extends ConsumerWidget {
  const _AllOffersView();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final openJobs = ref.watch(openJobsProvider);
    final credits = ref.watch(creditsProvider);
    final l10n = AppLocalizations.of(context)!;
    final authState = ref.watch(authProvider);
    final userRole = authState.user?['role'] as String?;
    final cs = Theme.of(context).colorScheme;

    return RefreshIndicator(
      color: cs.primary,
      onRefresh: () async {
        ref.invalidate(openJobsProvider);
        ref.invalidate(creditsProvider);
      },
      child: openJobs.when(
        loading: () => const _OpportunitySkeleton(),
        error: (e, _) => _ErrorState(
          onRetry: () => ref.invalidate(openJobsProvider),
          message: l10n.somethingWentWrong,
          retryLabel: l10n.retry,
        ),
        data: (jobs) {
          final userId = authState.user?['id'] as String?;
          final filtered = _filterByRole(
            jobs.where((j) => j.creatorId != userId).toList(),
            userRole,
          );

          if (filtered.isEmpty) {
            return ListView(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
              children: [
                _CreditsHeader(credits: credits, l10n: l10n),
                const SizedBox(height: 32),
                _EmptyState(message: l10n.noOpportunities),
              ],
            );
          }
          final myApps = ref.watch(myApplicationsProvider);
          final appliedJobIds = <String>{};
          myApps.whenData((apps) {
            for (final app in apps) {
              appliedJobIds.add(app.application.jobId);
            }
          });
          return ListView.builder(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
            itemCount: filtered.length + 1,
            itemBuilder: (context, index) {
              if (index == 0) {
                return Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: _CreditsHeader(credits: credits, l10n: l10n),
                );
              }
              final jobIndex = index - 1;
              return Padding(
                padding: EdgeInsets.only(
                  bottom: jobIndex < filtered.length - 1 ? 12 : 0,
                ),
                child: OpportunityCard(
                  job: filtered[jobIndex],
                  hasApplied: appliedJobIds.contains(filtered[jobIndex].id),
                ),
              );
            },
          );
        },
      ),
    );
  }

  /// Returns only jobs whose [applicantType] is compatible with [userRole].
  ///
  /// - provider  -> sees jobs with applicantType "freelancers" or "all"
  /// - agency    -> sees jobs with applicantType "agencies" or "all"
  /// - enterprise / null -> sees all jobs (no filtering)
  List<JobEntity> _filterByRole(List<JobEntity> jobs, String? userRole) {
    if (userRole == null) return jobs;
    switch (userRole) {
      case 'provider':
        return jobs
            .where(
              (j) =>
                  j.applicantType == 'freelancers' ||
                  j.applicantType == 'all',
            )
            .toList();
      case 'agency':
        return jobs
            .where(
              (j) =>
                  j.applicantType == 'agencies' || j.applicantType == 'all',
            )
            .toList();
      default:
        return jobs;
    }
  }
}

// ---------------------------------------------------------------------------
// Tab 2 — Mes candidatures (moved in from the deleted my_applications screen)
// ---------------------------------------------------------------------------

class _MyApplicationsView extends ConsumerWidget {
  const _MyApplicationsView();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final applications = ref.watch(myApplicationsProvider);
    final l10n = AppLocalizations.of(context)!;
    final cs = Theme.of(context).colorScheme;

    return RefreshIndicator(
      color: cs.primary,
      onRefresh: () async => ref.invalidate(myApplicationsProvider),
      child: applications.when(
        loading: () => const _ApplicationSkeleton(),
        error: (e, _) => _ErrorState(
          onRetry: () => ref.invalidate(myApplicationsProvider),
          message: l10n.somethingWentWrong,
          retryLabel: l10n.retry,
        ),
        data: (items) {
          if (items.isEmpty) {
            return ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              children: [
                SizedBox(height: MediaQuery.of(context).size.height * 0.18),
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  child: _EmptyState(message: l10n.noApplications),
                ),
              ],
            );
          }
          return ListView.separated(
            padding: const EdgeInsets.all(16),
            itemCount: items.length,
            separatorBuilder: (_, __) => const SizedBox(height: 12),
            itemBuilder: (context, index) {
              final item = items[index];
              return Card(
                child: ListTile(
                  title: Text(
                    item.job.title,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  subtitle: Text(
                    item.application.message,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                  trailing: IconButton(
                    icon: const Icon(
                      Icons.delete_outline,
                      color: Colors.red,
                    ),
                    onPressed: () => _confirmAndWithdraw(
                      context,
                      ref,
                      l10n,
                      applicationId: item.application.id,
                    ),
                  ),
                ),
              );
            },
          );
        },
      ),
    );
  }

  Future<void> _confirmAndWithdraw(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n, {
    required String applicationId,
  }) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.withdrawApplicationTitle),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(l10n.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(
              l10n.withdrawAction,
              style: const TextStyle(color: Colors.red),
            ),
          ),
        ],
      ),
    );
    if (confirmed == true) {
      await withdrawApplicationAction(ref, applicationId);
    }
  }
}

// ---------------------------------------------------------------------------
// Soleil credits chip (corail-soft pill) — replaces legacy red banner
// ---------------------------------------------------------------------------

class _CreditsHeader extends StatelessWidget {
  const _CreditsHeader({required this.credits, required this.l10n});

  final AsyncValue<int> credits;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    final soleil = Theme.of(context).extension<AppColors>()!;
    final creditCount = credits.valueOrNull ?? 0;
    final isLoading = credits.isLoading;
    final hasNoCredits = !isLoading && creditCount == 0;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          decoration: BoxDecoration(
            color: hasNoCredits
                ? soleil.amberSoft
                : soleil.accentSoft,
            borderRadius: BorderRadius.circular(AppTheme.radiusFull),
            border: Border.all(
              color: hasNoCredits ? soleil.borderStrong : soleil.primaryDeep,
              width: 0.6,
            ),
          ),
          child: Row(
            children: [
              Icon(
                Icons.confirmation_number_rounded,
                color: hasNoCredits ? soleil.warning : soleil.primaryDeep,
                size: 18,
              ),
              const SizedBox(width: 10),
              Expanded(
                child: isLoading
                    ? Text(
                        '...',
                        style: SoleilTextStyles.bodyEmphasis.copyWith(
                          color: cs.onSurface,
                        ),
                      )
                    : Text(
                        l10n.creditsRemaining(creditCount),
                        style: SoleilTextStyles.bodyEmphasis.copyWith(
                          color: hasNoCredits
                              ? cs.onSurface
                              : soleil.primaryDeep,
                          fontSize: 13,
                        ),
                      ),
              ),
              IconButton(
                onPressed: () => _showCreditsExplanation(context),
                icon: const Icon(Icons.help_outline_rounded, size: 18),
                color: soleil.primaryDeep,
                tooltip: l10n.creditsHowItWorks,
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(
                  minWidth: 28,
                  minHeight: 28,
                ),
              ),
            ],
          ),
        ),
        if (hasNoCredits) ...[
          const SizedBox(height: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            decoration: BoxDecoration(
              color: soleil.amberSoft,
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
            child: Row(
              children: [
                Icon(
                  Icons.warning_amber_rounded,
                  size: 18,
                  color: soleil.warning,
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    l10n.noCreditsLeft,
                    style: SoleilTextStyles.caption.copyWith(
                      color: cs.onSurface,
                      fontWeight: FontWeight.w500,
                      fontSize: 12,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ],
    );
  }

  void _showCreditsExplanation(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final cs = Theme.of(context).colorScheme;
    final soleil = Theme.of(context).extension<AppColors>()!;

    showModalBottomSheet<void>(
      context: context,
      backgroundColor: cs.surfaceContainerLowest,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) {
        return SafeArea(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(24, 24, 24, 16),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Center(
                  child: Container(
                    width: 40,
                    height: 4,
                    decoration: BoxDecoration(
                      color: cs.outline,
                      borderRadius: BorderRadius.circular(2),
                    ),
                  ),
                ),
                const SizedBox(height: 20),
                Text(
                  l10n.creditsHowItWorks,
                  style: SoleilTextStyles.headlineMedium.copyWith(
                    color: cs.onSurface,
                    fontSize: 22,
                  ),
                ),
                const SizedBox(height: 20),
                _ExplanationRow(
                  icon: Icons.touch_app_outlined,
                  text: l10n.creditsExplanation1,
                  color: soleil.primaryDeep,
                ),
                const SizedBox(height: 12),
                _ExplanationRow(
                  icon: Icons.calendar_today_outlined,
                  text: l10n.creditsExplanation2,
                  color: soleil.primaryDeep,
                ),
                const SizedBox(height: 12),
                _ExplanationRow(
                  icon: Icons.star_outline,
                  text: l10n.creditsExplanation3,
                  color: soleil.primaryDeep,
                ),
                const SizedBox(height: 12),
                _ExplanationRow(
                  icon: Icons.inventory_2_outlined,
                  text: l10n.creditsExplanation4,
                  color: soleil.primaryDeep,
                ),
                const SizedBox(height: 24),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: () => Navigator.of(context).pop(),
                    style: FilledButton.styleFrom(
                      backgroundColor: cs.primary,
                      foregroundColor: cs.onPrimary,
                      minimumSize: const Size.fromHeight(48),
                      shape: const StadiumBorder(),
                      textStyle: SoleilTextStyles.button,
                    ),
                    child: Text(l10n.cancel),
                  ),
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _ExplanationRow extends StatelessWidget {
  const _ExplanationRow({
    required this.icon,
    required this.text,
    required this.color,
  });

  final IconData icon;
  final String text;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(icon, size: 20, color: color),
        const SizedBox(width: 12),
        Expanded(
          child: Text(
            text,
            style: SoleilTextStyles.body.copyWith(
              color: Theme.of(context).colorScheme.onSurface,
            ),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Empty / error states — Soleil ivoire cards
// ---------------------------------------------------------------------------

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    final soleil = Theme.of(context).extension<AppColors>()!;

    return Container(
      padding: const EdgeInsets.all(32),
      decoration: BoxDecoration(
        color: cs.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(color: cs.outline),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        children: [
          Container(
            width: 56,
            height: 56,
            decoration: BoxDecoration(
              color: soleil.accentSoft,
              borderRadius: BorderRadius.circular(AppTheme.radiusFull),
            ),
            child: Icon(Icons.work_off_rounded, color: cs.primary, size: 24),
          ),
          const SizedBox(height: 14),
          Text(
            message,
            textAlign: TextAlign.center,
            style: SoleilTextStyles.titleMedium.copyWith(
              color: cs.onSurface,
              fontSize: 16,
            ),
          ),
        ],
      ),
    );
  }
}

class _ErrorState extends StatelessWidget {
  const _ErrorState({
    required this.onRetry,
    required this.message,
    required this.retryLabel,
  });

  final VoidCallback onRetry;
  final String message;
  final String retryLabel;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.error_outline_rounded, size: 48, color: cs.error),
          const SizedBox(height: 12),
          Text(
            message,
            style: SoleilTextStyles.body.copyWith(color: cs.onSurfaceVariant),
          ),
          const SizedBox(height: 12),
          TextButton(
            onPressed: onRetry,
            style: TextButton.styleFrom(
              foregroundColor: cs.primary,
              textStyle: SoleilTextStyles.bodyEmphasis,
            ),
            child: Text(retryLabel),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Skeleton shimmer loaders (Soleil ivoire surfaces)
// ---------------------------------------------------------------------------

class _OpportunitySkeleton extends StatelessWidget {
  const _OpportunitySkeleton();

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    return Shimmer.fromColors(
      baseColor: cs.outline,
      highlightColor: cs.surfaceContainerLowest,
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        physics: const NeverScrollableScrollPhysics(),
        itemCount: 3,
        separatorBuilder: (_, __) => const SizedBox(height: 12),
        itemBuilder: (_, __) => Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: cs.surfaceContainerLowest,
            borderRadius: BorderRadius.circular(AppTheme.radiusXl),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 200,
                height: 16,
                decoration: BoxDecoration(
                  color: cs.surfaceContainerLowest,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
              const SizedBox(height: 8),
              Container(
                width: double.infinity,
                height: 12,
                decoration: BoxDecoration(
                  color: cs.surfaceContainerLowest,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
              const SizedBox(height: 4),
              Container(
                width: 160,
                height: 12,
                decoration: BoxDecoration(
                  color: cs.surfaceContainerLowest,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  Container(
                    width: 60,
                    height: 24,
                    decoration: BoxDecoration(
                      color: cs.surfaceContainerLowest,
                      borderRadius: BorderRadius.circular(12),
                    ),
                  ),
                  const SizedBox(width: 6),
                  Container(
                    width: 80,
                    height: 24,
                    decoration: BoxDecoration(
                      color: cs.surfaceContainerLowest,
                      borderRadius: BorderRadius.circular(12),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              Container(
                width: 100,
                height: 12,
                decoration: BoxDecoration(
                  color: cs.surfaceContainerLowest,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ApplicationSkeleton extends StatelessWidget {
  const _ApplicationSkeleton();

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    return Shimmer.fromColors(
      baseColor: cs.outline,
      highlightColor: cs.surfaceContainerLowest,
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        physics: const NeverScrollableScrollPhysics(),
        itemCount: 3,
        separatorBuilder: (_, __) => const SizedBox(height: 12),
        itemBuilder: (_, __) => Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: cs.surfaceContainerLowest,
            borderRadius: BorderRadius.circular(AppTheme.radiusLg),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 180,
                height: 16,
                decoration: BoxDecoration(
                  color: cs.surfaceContainerLowest,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
              const SizedBox(height: 8),
              Container(
                width: double.infinity,
                height: 12,
                decoration: BoxDecoration(
                  color: cs.surfaceContainerLowest,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
              const SizedBox(height: 4),
              Container(
                width: 140,
                height: 12,
                decoration: BoxDecoration(
                  color: cs.surfaceContainerLowest,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
