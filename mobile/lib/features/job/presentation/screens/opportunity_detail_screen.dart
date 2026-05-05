import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/video_player_widget.dart';
import '../../../reporting/presentation/widgets/report_bottom_sheet.dart';
import '../../domain/entities/job_entity.dart';
import '../providers/job_provider.dart';
import '../widgets/apply_bottom_sheet.dart';

/// W-13 mobile parity · Opportunity detail.
///
/// AppBar with Fraunces title, Soleil ivoire body, calm card sections
/// (budget hero, optional video, skills as corail-soft pills, full
/// description). Sticky bottom corail FilledButton "Postuler" — disabled
/// state turns sable to keep the warm palette consistent.
class OpportunityDetailScreen extends ConsumerWidget {
  const OpportunityDetailScreen({super.key, required this.jobId});

  final String jobId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final hasApplied = ref.watch(hasAppliedProvider(jobId));
    final credits = ref.watch(creditsProvider);
    final canApply = ref.watch(
      hasPermissionProvider(OrgPermission.proposalsCreate),
    );
    final l10n = AppLocalizations.of(context)!;

    return FutureBuilder<JobEntity>(
      future: ref.read(jobRepositoryProvider).getJob(jobId),
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return Scaffold(
            appBar: AppBar(),
            body: const Center(child: CircularProgressIndicator()),
          );
        }
        if (snapshot.hasError || !snapshot.hasData) {
          return Scaffold(
            appBar: AppBar(),
            body: Center(child: Text(l10n.jobNotFound)),
          );
        }
        final job = snapshot.data!;
        final alreadyApplied = hasApplied.valueOrNull ?? false;
        final creditCount = credits.valueOrNull ?? 0;
        final noCredits = !credits.isLoading && creditCount == 0;
        final isDisabled = alreadyApplied || noCredits || !canApply;

        return Scaffold(
          appBar: AppBar(
            title: Text(
              job.title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: SoleilTextStyles.titleLarge.copyWith(
                color: Theme.of(context).colorScheme.onSurface,
                fontSize: 18,
              ),
            ),
            actions: [
              PopupMenuButton<String>(
                onSelected: (value) {
                  if (value == 'report') {
                    showReportBottomSheet(
                      context,
                      ref,
                      targetType: 'job',
                      targetId: jobId,
                      conversationId: '',
                    );
                  }
                },
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
                itemBuilder: (_) => [
                  PopupMenuItem(
                    value: 'report',
                    child: Row(
                      children: [
                        Icon(
                          Icons.flag_outlined,
                          size: 18,
                          color: Theme.of(context).colorScheme.error,
                        ),
                        const SizedBox(width: 8),
                        Text(l10n.reportJob),
                      ],
                    ),
                  ),
                ],
              ),
            ],
          ),
          body: SafeArea(
            child: SingleChildScrollView(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Eyebrow + Fraunces title
                  _SoleilEyebrow(
                    text: job.budgetType == 'one_shot'
                        ? l10n.budgetTypeOneShot
                        : l10n.budgetTypeLongTerm,
                  ),
                  const SizedBox(height: 8),
                  Text(
                    job.title,
                    style: SoleilTextStyles.headlineMedium.copyWith(
                      color: Theme.of(context).colorScheme.onSurface,
                      fontSize: 24,
                    ),
                  ),
                  const SizedBox(height: 16),
                  if (job.videoUrl != null && job.videoUrl!.isNotEmpty) ...[
                    ClipRRect(
                      borderRadius: BorderRadius.circular(AppTheme.radiusXl),
                      child: VideoPlayerWidget(videoUrl: job.videoUrl!),
                    ),
                    const SizedBox(height: 16),
                  ],
                  _BudgetCard(
                    job: job,
                    typeLabel: job.budgetType == 'one_shot'
                        ? l10n.budgetTypeOneShot
                        : l10n.budgetTypeLongTerm,
                  ),
                  const SizedBox(height: 16),
                  if (job.skills.isNotEmpty) ...[
                    _SoleilSection(
                      title: l10n.requiredSkills,
                      child: Wrap(
                        spacing: 8,
                        runSpacing: 8,
                        children: job.skills.map((s) {
                          final soleil = Theme.of(
                            context,
                          ).extension<AppColors>()!;
                          return Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 12,
                              vertical: 6,
                            ),
                            decoration: BoxDecoration(
                              color: soleil.accentSoft,
                              borderRadius: BorderRadius.circular(
                                AppTheme.radiusFull,
                              ),
                            ),
                            child: Text(
                              s,
                              style: SoleilTextStyles.caption.copyWith(
                                fontSize: 12,
                                fontWeight: FontWeight.w600,
                                color: soleil.primaryDeep,
                              ),
                            ),
                          );
                        }).toList(),
                      ),
                    ),
                    const SizedBox(height: 16),
                  ],
                  _SoleilSection(
                    title: l10n.jobDescription,
                    child: Text(
                      job.description,
                      style: SoleilTextStyles.bodyLarge.copyWith(
                        color: Theme.of(context).colorScheme.onSurface,
                        height: 1.6,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
          bottomNavigationBar: SafeArea(
            child: _ApplyBottomBar(
              isDisabled: isDisabled,
              alreadyApplied: alreadyApplied,
              noCredits: noCredits,
              canApply: canApply,
              jobId: jobId,
            ),
          ),
        );
      },
    );
  }
}

class _SoleilEyebrow extends StatelessWidget {
  const _SoleilEyebrow({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Text(
      text.toUpperCase(),
      style: SoleilTextStyles.mono.copyWith(
        color: Theme.of(context).colorScheme.onSurfaceVariant,
        fontSize: 11,
        letterSpacing: 0.96,
        fontWeight: FontWeight.w700,
      ),
    );
  }
}

class _BudgetCard extends StatelessWidget {
  const _BudgetCard({required this.job, required this.typeLabel});

  final JobEntity job;
  final String typeLabel;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    final soleil = Theme.of(context).extension<AppColors>()!;

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: cs.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(color: cs.outline),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Row(
        children: [
          Container(
            width: 44,
            height: 44,
            decoration: BoxDecoration(
              color: soleil.accentSoft,
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
            child: Icon(Icons.work_rounded, color: cs.primary, size: 22),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  typeLabel.toUpperCase(),
                  style: SoleilTextStyles.mono.copyWith(
                    color: cs.onSurfaceVariant,
                    fontSize: 11,
                    letterSpacing: 0.96,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  '${job.minBudget} € — ${job.maxBudget} €',
                  style: SoleilTextStyles.titleMedium.copyWith(
                    color: cs.onSurface,
                    fontSize: 20,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _SoleilSection extends StatelessWidget {
  const _SoleilSection({required this.title, required this.child});

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: cs.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(color: cs.outline),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            title,
            style: SoleilTextStyles.titleMedium.copyWith(
              color: cs.onSurface,
              fontSize: 16,
            ),
          ),
          const SizedBox(height: 12),
          child,
        ],
      ),
    );
  }
}

class _ApplyBottomBar extends ConsumerWidget {
  const _ApplyBottomBar({
    required this.isDisabled,
    required this.alreadyApplied,
    required this.noCredits,
    required this.canApply,
    required this.jobId,
  });

  final bool isDisabled;
  final bool alreadyApplied;
  final bool noCredits;
  final bool canApply;
  final String jobId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final cs = Theme.of(context).colorScheme;
    final soleil = Theme.of(context).extension<AppColors>()!;

    return Container(
      decoration: BoxDecoration(
        color: cs.surfaceContainerLowest,
        border: Border(top: BorderSide(color: cs.outline)),
      ),
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (!canApply && !alreadyApplied) ...[
            Text(
              l10n.permissionDenied,
              style: SoleilTextStyles.caption.copyWith(color: cs.error),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
          ] else if (noCredits && !alreadyApplied) ...[
            Text(
              l10n.noCreditsCannotApply,
              style: SoleilTextStyles.caption.copyWith(color: cs.error),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
          ],
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: isDisabled
                  ? null
                  : () => showApplyBottomSheet(context, ref, jobId),
              style: FilledButton.styleFrom(
                backgroundColor: cs.primary,
                disabledBackgroundColor: soleil.borderStrong,
                disabledForegroundColor: cs.onSurfaceVariant,
                foregroundColor: cs.onPrimary,
                minimumSize: const Size.fromHeight(48),
                shape: const StadiumBorder(),
                textStyle: SoleilTextStyles.button.copyWith(fontSize: 14),
              ),
              child: Text(
                alreadyApplied ? l10n.alreadyApplied : l10n.applyAction,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
