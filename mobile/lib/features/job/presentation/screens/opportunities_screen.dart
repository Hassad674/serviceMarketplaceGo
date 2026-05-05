import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shimmer/shimmer.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../domain/entities/job_entity.dart';
import '../providers/job_provider.dart';
import '../widgets/opportunity_card.dart';

/// W-12 mobile parity · Opportunities feed.
///
/// Soleil v2 ivoire scaffold, AppBar with Fraunces title (themed),
/// editorial credits chip on top, calm card list. The role-based filter
/// + applied-set logic is preserved exactly — only the chrome changes.
class OpportunitiesScreen extends ConsumerWidget {
  const OpportunitiesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final openJobs = ref.watch(openJobsProvider);
    final credits = ref.watch(creditsProvider);
    final l10n = AppLocalizations.of(context)!;
    final authState = ref.watch(authProvider);
    final userRole = authState.user?['role'] as String?;
    final cs = Theme.of(context).colorScheme;

    return Scaffold(
      appBar: AppBar(
        title: Text(
          l10n.opportunities,
          style: SoleilTextStyles.titleLarge.copyWith(
            color: cs.onSurface,
            fontSize: 20,
          ),
        ),
      ),
      body: RefreshIndicator(
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
// Skeleton shimmer loader (Soleil ivoire surfaces)
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
