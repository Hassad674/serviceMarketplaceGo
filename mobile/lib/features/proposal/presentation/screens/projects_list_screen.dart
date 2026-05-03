import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/proposal_entity.dart';
import '../providers/proposal_provider.dart';
import '../../../../core/theme/app_palette.dart';

/// Displays the list of projects (proposals that are paid/active/completed).
///
/// Pulls data from `GET /api/v1/projects` via [projectsProvider].
/// Supports pull-to-refresh and shows empty state guidance.
class ProjectsListScreen extends ConsumerWidget {
  const ProjectsListScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final projectsAsync = ref.watch(projectsProvider);

    return Scaffold(
      appBar: AppBar(
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.activeProjects),
      ),
      body: SafeArea(
        child: projectsAsync.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (error, _) => _buildErrorState(context, ref, l10n),
          data: (projects) => projects.isEmpty
              ? _buildEmptyState(context, theme, l10n)
              : _buildProjectsList(context, ref, theme, l10n, projects),
        ),
      ),
    );
  }

  Widget _buildProjectsList(
    BuildContext context,
    WidgetRef ref,
    ThemeData theme,
    AppLocalizations l10n,
    List<ProposalEntity> projects,
  ) {
    return RefreshIndicator(
      onRefresh: () async => ref.invalidate(projectsProvider),
      child: ListView.separated(
        padding: const EdgeInsets.all(16),
        itemCount: projects.length,
        separatorBuilder: (_, __) => const SizedBox(height: 12),
        itemBuilder: (context, index) {
          return _ProjectCard(proposal: projects[index]);
        },
      ),
    );
  }

  Widget _buildEmptyState(
    BuildContext context,
    ThemeData theme,
    AppLocalizations l10n,
  ) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 80,
              height: 80,
              decoration: BoxDecoration(
                color: theme.colorScheme.primary.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(AppTheme.radiusXl),
              ),
              child: Icon(
                Icons.folder_open_outlined,
                size: 40,
                color: theme.colorScheme.primary,
              ),
            ),
            const SizedBox(height: 24),
            Text(
              l10n.noActiveProjects,
              style: theme.textTheme.titleLarge,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              l10n.noActiveProjectsDesc,
              style: theme.textTheme.bodyMedium?.copyWith(
                color:
                    theme.colorScheme.onSurface.withValues(alpha: 0.6),
              ),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildErrorState(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
  ) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(l10n.unexpectedError),
          const SizedBox(height: 12),
          ElevatedButton(
            onPressed: () => ref.invalidate(projectsProvider),
            child: Text(l10n.retry),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Project card widget
// ---------------------------------------------------------------------------

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
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Title + status badge
          Row(
            children: [
              Expanded(
                child: Text(
                  proposal.title,
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              const SizedBox(width: 8),
              _StatusBadge(status: proposal.status),
            ],
          ),
          const SizedBox(height: 12),

          // Amount
          Row(
            children: [
              Icon(
                Icons.euro_outlined,
                size: 16,
                color: appColors?.mutedForeground,
              ),
              const SizedBox(width: 6),
              Text(
                '\u20AC ${proposal.amountInEuros.toStringAsFixed(2)}',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.primary,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ),

          // Deadline
          if (proposal.deadline != null) ...[
            const SizedBox(height: 8),
            Row(
              children: [
                Icon(
                  Icons.calendar_today_outlined,
                  size: 16,
                  color: appColors?.mutedForeground,
                ),
                const SizedBox(width: 6),
                Text(
                  _formatDeadline(proposal.deadline!),
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight: FontWeight.w500,
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

// ---------------------------------------------------------------------------
// Status badge
// ---------------------------------------------------------------------------

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final (label, bgColor, fgColor) = _statusStyle(context);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          color: fgColor,
        ),
      ),
    );
  }

  (String, Color, Color) _statusStyle(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    switch (status) {
      case 'paid':
      case 'active':
        return (
          l10n.projectStatusActive,
          AppPalette.green100,
          AppPalette.green800,
        );
      case 'disputed':
        return (
          l10n.projectStatusDisputed,
          AppPalette.orange100, // orange-100
          AppPalette.orange700, // orange-700
        );
      case 'completed':
        return (
          l10n.projectStatusCompleted,
          AppPalette.sky100,
          AppPalette.sky800,
        );
      case 'accepted':
        return (
          l10n.proposalAccepted,
          AppPalette.amber100,
          AppPalette.amber800,
        );
      default:
        return (
          status,
          AppPalette.slate100,
          AppPalette.slate600,
        );
    }
  }
}
