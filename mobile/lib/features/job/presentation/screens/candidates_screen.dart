import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../providers/job_provider.dart';
import '../widgets/candidate_card.dart';

/// Standalone candidates screen — Soleil v2 visual port.
///
/// Reachable from the legacy "/candidates" route (now superseded by
/// the M-08 candidates tab inside `JobDetailScreen`). Keeps a Soleil
/// AppBar + the same Soleil-styled empty/loading/error states used in
/// the tabbed flow.
class CandidatesScreen extends ConsumerWidget {
  const CandidatesScreen({super.key, required this.jobId});

  final String jobId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final candidates = ref.watch(jobApplicationsProvider(jobId));
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      backgroundColor: cs.surface,
      appBar: AppBar(
        backgroundColor: cs.surfaceContainerLowest,
        scrolledUnderElevation: 0,
        elevation: 0,
        title: Text(
          l10n.applications,
          style: SoleilTextStyles.titleLarge.copyWith(
            color: cs.onSurface,
            fontSize: 18,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      body: SafeArea(
        top: false,
        child: RefreshIndicator(
          onRefresh: () async => ref.invalidate(jobApplicationsProvider(jobId)),
          child: candidates.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => _ErrorView(
              jobId: jobId,
              message: l10n.somethingWentWrong,
              retryLabel: l10n.retry,
            ),
            data: (items) {
              if (items.isEmpty) {
                return _EmptyView(
                  title: l10n.jobDetail_m08_emptyTitle,
                  body: l10n.jobDetail_m08_emptyBody,
                );
              }
              return ListView.separated(
                padding: const EdgeInsets.fromLTRB(20, 16, 20, 28),
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
        ),
      ),
    );
  }
}

class _ErrorView extends ConsumerWidget {
  const _ErrorView({
    required this.jobId,
    required this.message,
    required this.retryLabel,
  });

  final String jobId;
  final String message;
  final String retryLabel;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 60, 20, 28),
      children: [
        Icon(
          Icons.error_outline,
          size: 48,
          color: theme.colorScheme.onSurfaceVariant,
        ),
        const SizedBox(height: 12),
        Text(
          message,
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 8),
        Center(
          child: TextButton(
            onPressed: () => ref.invalidate(jobApplicationsProvider(jobId)),
            child: Text(retryLabel),
          ),
        ),
      ],
    );
  }
}

class _EmptyView extends StatelessWidget {
  const _EmptyView({required this.title, required this.body});

  final String title;
  final String body;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;
    final soleil = theme.extension<AppColors>()!;

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 40, 20, 28),
      children: [
        Container(
          padding: const EdgeInsets.all(24),
          decoration: BoxDecoration(
            color: soleil.accentSoft,
            borderRadius: BorderRadius.circular(AppTheme.radius2xl),
            border: Border.all(color: cs.outline),
          ),
          child: Column(
            children: [
              Container(
                width: 56,
                height: 56,
                decoration: BoxDecoration(
                  color: cs.surfaceContainerLowest,
                  borderRadius: BorderRadius.circular(AppTheme.radiusFull),
                ),
                child: Icon(
                  Icons.groups_outlined,
                  color: soleil.primaryDeep,
                  size: 28,
                ),
              ),
              const SizedBox(height: 16),
              Text(
                title,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.titleMedium.copyWith(
                  color: cs.onSurface,
                  fontSize: 18,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                body,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.body.copyWith(
                  color: cs.onSurfaceVariant,
                  height: 1.5,
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}
