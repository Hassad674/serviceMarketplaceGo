import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shimmer/shimmer.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../domain/entities/job_entity.dart';
import '../providers/job_provider.dart';
import '../widgets/opportunity_card.dart';
import '../../../../core/theme/app_palette.dart';

class OpportunitiesScreen extends ConsumerWidget {
  const OpportunitiesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final openJobs = ref.watch(openJobsProvider);
    final credits = ref.watch(creditsProvider);
    final l10n = AppLocalizations.of(context)!;
    final authState = ref.watch(authProvider);
    final userRole = authState.user?['role'] as String?;

    return Scaffold(
      appBar: AppBar(title: Text(l10n.opportunities)),
      body: RefreshIndicator(
        onRefresh: () async {
          ref.invalidate(openJobsProvider);
          ref.invalidate(creditsProvider);
        },
        child: openJobs.when(
          loading: () => const _OpportunitySkeleton(),
          error: (e, _) => Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Icon(Icons.error_outline, size: 48, color: Colors.grey),
                const SizedBox(height: 12),
                Text(l10n.somethingWentWrong, style: const TextStyle(color: Colors.grey)),
                const SizedBox(height: 8),
                TextButton(
                  onPressed: () => ref.invalidate(openJobsProvider),
                  child: Text(l10n.retry),
                ),
              ],
            ),
          ),
          data: (jobs) {
            final userId = authState.user?['id'] as String?;
            final filtered = _filterByRole(
              jobs.where((j) => j.creatorId != userId).toList(),
              userRole,
            );

            if (filtered.isEmpty) {
              return ListView(
                children: [
                  _CreditsHeader(credits: credits, l10n: l10n),
                  SizedBox(height: MediaQuery.of(context).size.height * 0.2),
                  const Icon(Icons.work_off_outlined, size: 48, color: Colors.grey),
                  const SizedBox(height: 12),
                  Text(
                    l10n.noOpportunities,
                    textAlign: TextAlign.center,
                    style: const TextStyle(color: Colors.grey),
                  ),
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
              padding: const EdgeInsets.all(16),
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
                  padding: EdgeInsets.only(bottom: jobIndex < filtered.length - 1 ? 12 : 0),
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
            .where((j) =>
                j.applicantType == 'freelancers' || j.applicantType == 'all')
            .toList();
      case 'agency':
        return jobs
            .where((j) =>
                j.applicantType == 'agencies' || j.applicantType == 'all')
            .toList();
      default:
        return jobs;
    }
  }
}

// ---------------------------------------------------------------------------
// Credits header + explanation modal
// ---------------------------------------------------------------------------

class _CreditsHeader extends StatelessWidget {
  const _CreditsHeader({required this.credits, required this.l10n});

  final AsyncValue<int> credits;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final creditCount = credits.valueOrNull ?? 0;
    final isLoading = credits.isLoading;
    final hasNoCredits = !isLoading && creditCount == 0;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: BoxDecoration(
            color: hasNoCredits
                ? AppPalette.red50
                : AppPalette.rose50,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: hasNoCredits
                  ? AppPalette.red200
                  : AppPalette.rose300,
            ),
          ),
          child: Row(
            children: [
              Icon(
                Icons.confirmation_number_outlined,
                color: hasNoCredits
                    ? AppPalette.red500
                    : AppPalette.rose500,
                size: 22,
              ),
              const SizedBox(width: 10),
              Expanded(
                child: isLoading
                    ? Text(
                        '...',
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: theme.colorScheme.onSurface,
                        ),
                      )
                    : Text(
                        l10n.creditsRemaining(creditCount),
                        style: theme.textTheme.bodyMedium?.copyWith(
                          fontWeight: FontWeight.w600,
                          color: hasNoCredits
                              ? AppPalette.red500
                              : theme.colorScheme.onSurface,
                        ),
                      ),
              ),
              IconButton(
                onPressed: () => _showCreditsExplanation(context),
                icon: const Icon(Icons.help_outline, size: 20),
                color: AppPalette.rose500,
                tooltip: l10n.creditsHowItWorks,
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
              ),
            ],
          ),
        ),
        if (hasNoCredits) ...[
          const SizedBox(height: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            decoration: BoxDecoration(
              color: AppPalette.red50,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Row(
              children: [
                const Icon(Icons.warning_amber_rounded, size: 18, color: AppPalette.red500),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    l10n.noCreditsLeft,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: AppPalette.red500,
                      fontWeight: FontWeight.w500,
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
    final theme = Theme.of(context);

    showModalBottomSheet<void>(
      context: context,
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
                      color: Colors.grey.shade300,
                      borderRadius: BorderRadius.circular(2),
                    ),
                  ),
                ),
                const SizedBox(height: 20),
                Text(
                  l10n.creditsHowItWorks,
                  style: theme.textTheme.titleLarge?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 20),
                _ExplanationRow(
                  icon: Icons.touch_app_outlined,
                  text: l10n.creditsExplanation1,
                ),
                const SizedBox(height: 12),
                _ExplanationRow(
                  icon: Icons.calendar_today_outlined,
                  text: l10n.creditsExplanation2,
                ),
                const SizedBox(height: 12),
                _ExplanationRow(
                  icon: Icons.star_outline,
                  text: l10n.creditsExplanation3,
                ),
                const SizedBox(height: 12),
                _ExplanationRow(
                  icon: Icons.inventory_2_outlined,
                  text: l10n.creditsExplanation4,
                ),
                const SizedBox(height: 24),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: () => Navigator.of(context).pop(),
                    style: FilledButton.styleFrom(
                      backgroundColor: AppPalette.rose500,
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
  const _ExplanationRow({required this.icon, required this.text});

  final IconData icon;
  final String text;

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(icon, size: 20, color: AppPalette.rose500),
        const SizedBox(width: 12),
        Expanded(
          child: Text(
            text,
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
              color: Theme.of(context).colorScheme.onSurface,
            ),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Skeleton shimmer loader
// ---------------------------------------------------------------------------

class _OpportunitySkeleton extends StatelessWidget {
  const _OpportunitySkeleton();

  @override
  Widget build(BuildContext context) {
    return Shimmer.fromColors(
      baseColor: Colors.grey.shade200,
      highlightColor: Colors.grey.shade50,
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        physics: const NeverScrollableScrollPhysics(),
        itemCount: 3,
        separatorBuilder: (_, __) => const SizedBox(height: 12),
        itemBuilder: (_, __) => Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(AppTheme.radiusLg),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 200,
                height: 16,
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
              const SizedBox(height: 8),
              Container(
                width: double.infinity,
                height: 12,
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(4),
                ),
              ),
              const SizedBox(height: 4),
              Container(
                width: 160,
                height: 12,
                decoration: BoxDecoration(
                  color: Colors.white,
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
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(12),
                    ),
                  ),
                  const SizedBox(width: 6),
                  Container(
                    width: 80,
                    height: 24,
                    decoration: BoxDecoration(
                      color: Colors.white,
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
                  color: Colors.white,
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
