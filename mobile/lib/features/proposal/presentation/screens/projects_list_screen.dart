import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/proposal_entity.dart';
import '../providers/proposal_provider.dart';

/// Soleil v2 — Projects (active missions) list.
///
/// Editorial header (corail mono eyebrow + Fraunces italic-corail title +
/// tabac subtitle), Soleil project cards with status pill + Geist Mono
/// budget. Pulls data from `GET /api/v1/projects` via [projectsProvider].
class ProjectsListScreen extends ConsumerWidget {
  const ProjectsListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final projectsAsync = ref.watch(projectsProvider);

    return Scaffold(
      backgroundColor: theme.colorScheme.surface,
      appBar: AppBar(
        backgroundColor: theme.colorScheme.surface,
        elevation: 0,
        scrolledUnderElevation: 0,
        leading: IconButton(
          icon: const Icon(Icons.menu_rounded),
          color: theme.colorScheme.onSurface,
          onPressed: openShellDrawer,
        ),
        title: Text(
          l10n.activeProjects,
          style: SoleilTextStyles.titleMedium.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
      ),
      body: SafeArea(
        child: projectsAsync.when(
          loading: () => const _ListSkeleton(),
          error: (error, _) =>
              _ErrorBlock(onRetry: () => ref.invalidate(projectsProvider)),
          data: (projects) => projects.isEmpty
              ? _EmptyState()
              : _ProjectsList(
                  projects: projects,
                  onRefresh: () async => ref.invalidate(projectsProvider),
                ),
        ),
      ),
    );
  }
}

class _ProjectsList extends StatelessWidget {
  const _ProjectsList({required this.projects, required this.onRefresh});

  final List<ProposalEntity> projects;
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    return RefreshIndicator(
      onRefresh: onRefresh,
      color: Theme.of(context).colorScheme.primary,
      child: ListView.separated(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
        itemCount: projects.length + 1,
        separatorBuilder: (_, index) =>
            SizedBox(height: index == 0 ? 20 : 12),
        itemBuilder: (context, index) {
          if (index == 0) return const _Header();
          return _ProjectCard(proposal: projects[index - 1]);
        },
      ),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final primary = theme.colorScheme.primary;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.proposalFlow_list_eyebrow,
          style: SoleilTextStyles.mono.copyWith(
            color: primary,
            fontSize: 11,
            fontWeight: FontWeight.w700,
            letterSpacing: 1.4,
          ),
        ),
        const SizedBox(height: 8),
        RichText(
          text: TextSpan(
            style: SoleilTextStyles.headlineLarge.copyWith(
              color: theme.colorScheme.onSurface,
            ),
            children: [
              TextSpan(text: '${l10n.proposalFlow_list_titlePrefix} '),
              TextSpan(
                text: l10n.proposalFlow_list_titleAccent,
                style: SoleilTextStyles.headlineLarge.copyWith(
                  color: primary,
                  fontStyle: FontStyle.italic,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),
        Text(
          l10n.proposalFlow_list_subtitle,
          style: SoleilTextStyles.bodyLarge.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}

class _ProjectCard extends StatelessWidget {
  const _ProjectCard({required this.proposal});

  final ProposalEntity proposal;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusXl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  proposal.title,
                  style: SoleilTextStyles.titleMedium.copyWith(
                    color: theme.colorScheme.onSurface,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              const SizedBox(width: 8),
              _StatusPill(status: proposal.status),
            ],
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Icon(
                Icons.euro_rounded,
                size: 16,
                color: theme.colorScheme.onSurfaceVariant,
              ),
              const SizedBox(width: 6),
              Text(
                '€ ${proposal.amountInEuros.toStringAsFixed(2)}',
                style: SoleilTextStyles.monoLarge.copyWith(
                  color: theme.colorScheme.onSurface,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ),
          if (proposal.deadline != null) ...[
            const SizedBox(height: 6),
            Row(
              children: [
                Icon(
                  Icons.calendar_today_rounded,
                  size: 14,
                  color: theme.colorScheme.onSurfaceVariant,
                ),
                const SizedBox(width: 6),
                Text(
                  _formatDeadline(proposal.deadline!),
                  style: SoleilTextStyles.mono.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ],
        ],
      ),
    );
  }

  String _formatDeadline(String isoDate) {
    try {
      final dt = DateTime.parse(isoDate);
      final d = dt.day.toString().padLeft(2, '0');
      final m = dt.month.toString().padLeft(2, '0');
      return '$d/$m/${dt.year}';
    } catch (_) {
      return isoDate;
    }
  }
}

class _StatusPill extends StatelessWidget {
  const _StatusPill({required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    final (label, bgColor, fgColor) = switch (status) {
      'paid' || 'active' => (
          l10n.projectStatusActive,
          appColors?.successSoft ?? theme.colorScheme.primaryContainer,
          appColors?.success ?? theme.colorScheme.primary,
        ),
      'disputed' => (
          l10n.projectStatusDisputed,
          appColors?.amberSoft ?? theme.colorScheme.primaryContainer,
          appColors?.warning ?? theme.colorScheme.error,
        ),
      'completed' => (
          l10n.projectStatusCompleted,
          theme.colorScheme.outline.withValues(alpha: 0.2),
          theme.colorScheme.onSurfaceVariant,
        ),
      'accepted' => (
          l10n.proposalAccepted,
          appColors?.amberSoft ?? theme.colorScheme.primaryContainer,
          appColors?.warning ?? theme.colorScheme.primary,
        ),
      _ => (
          status,
          theme.colorScheme.outline.withValues(alpha: 0.2),
          theme.colorScheme.onSurfaceVariant,
        ),
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        label,
        style: SoleilTextStyles.mono.copyWith(
          color: fgColor,
          fontSize: 10.5,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.6,
        ),
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;

    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 24, 16, 24),
      children: [
        const _Header(),
        const SizedBox(height: 24),
        Container(
          padding: const EdgeInsets.all(32),
          decoration: BoxDecoration(
            color: theme.colorScheme.surface,
            borderRadius: BorderRadius.circular(AppTheme.radius2xl),
            border: Border.all(
              color: appColors?.borderStrong ??
                  theme.colorScheme.outline.withValues(alpha: 0.6),
              style: BorderStyle.solid,
            ),
          ),
          child: Column(
            children: [
              Container(
                width: 56,
                height: 56,
                decoration: BoxDecoration(
                  color: theme.colorScheme.primaryContainer,
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.folder_open_rounded,
                  size: 28,
                  color: primary,
                ),
              ),
              const SizedBox(height: 20),
              Text(
                l10n.proposalFlow_list_emptyTitle,
                style: SoleilTextStyles.titleLarge.copyWith(
                  color: theme.colorScheme.onSurface,
                ),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 8),
              Text(
                l10n.proposalFlow_list_emptyBody,
                style: SoleilTextStyles.body.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
                textAlign: TextAlign.center,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _ErrorBlock extends StatelessWidget {
  const _ErrorBlock({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline_rounded,
              size: 40,
              color: theme.colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(
              l10n.unexpectedError,
              style: SoleilTextStyles.body
                  .copyWith(color: theme.colorScheme.onSurfaceVariant),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: onRetry,
              style: FilledButton.styleFrom(
                shape: const StadiumBorder(),
                padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
              ),
              child: Text(l10n.retry),
            ),
          ],
        ),
      ),
    );
  }
}

class _ListSkeleton extends StatelessWidget {
  const _ListSkeleton();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = theme.colorScheme.outline.withValues(alpha: 0.2);

    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 24, 16, 16),
      children: [
        const _Header(),
        const SizedBox(height: 20),
        for (var i = 0; i < 4; i++) ...[
          Container(
            height: 96,
            decoration: BoxDecoration(
              color: color,
              borderRadius: BorderRadius.circular(AppTheme.radiusXl),
            ),
          ),
          const SizedBox(height: 12),
        ],
      ],
    );
  }
}
