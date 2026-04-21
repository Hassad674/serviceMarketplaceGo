import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/client_profile.dart';

/// Renders the list of completed projects on the client profile.
///
/// Each row shows title + amount + completion date + a small chip for
/// the provider side of the engagement. The provider chip deep-links
/// to the provider's public profile via `/profiles/{orgId}` so users
/// can jump across the graph without a dedicated search step.
class ClientProjectHistoryWidget extends StatelessWidget {
  const ClientProjectHistoryWidget({
    super.key,
    required this.projects,
  });

  final List<ClientProjectEntry> projects;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                Icons.history,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Text(
                l10n.clientProfileProjectHistoryTitle,
                style: theme.textTheme.titleMedium,
              ),
            ],
          ),
          const SizedBox(height: 12),
          if (projects.isEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Text(
                l10n.clientProfileProjectHistoryEmpty,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: appColors?.mutedForeground,
                  fontStyle: FontStyle.italic,
                ),
              ),
            )
          else
            // List.generate keeps the code readable vs. interleaved
            // spread + Divider.
            for (var i = 0; i < projects.length; i++) ...[
              _ProjectRow(entry: projects[i]),
              if (i < projects.length - 1)
                Divider(
                  height: 20,
                  color: appColors?.border ?? theme.dividerColor,
                ),
            ],
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Single project row
// ---------------------------------------------------------------------------

class _ProjectRow extends StatelessWidget {
  const _ProjectRow({required this.entry});

  final ClientProjectEntry entry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(
              child: Text(
                entry.title,
                style: theme.textTheme.bodyLarge?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            const SizedBox(width: 12),
            Text(
              _formatEuros(entry.amount),
              style: theme.textTheme.bodyMedium?.copyWith(
                fontWeight: FontWeight.w700,
                color: theme.colorScheme.primary,
              ),
            ),
          ],
        ),
        const SizedBox(height: 4),
        Text(
          _formatDate(entry.completedAt),
          style: theme.textTheme.bodySmall?.copyWith(
            color: appColors?.mutedForeground,
          ),
        ),
        const SizedBox(height: 8),
        _ProviderChip(provider: entry.provider),
      ],
    );
  }

  String _formatEuros(int cents) {
    final euros = cents / 100;
    if (euros >= 1000) {
      final whole = euros.toStringAsFixed(0);
      final buffer = StringBuffer();
      final reversed = whole.split('').reversed.toList();
      for (var i = 0; i < reversed.length; i++) {
        if (i > 0 && i % 3 == 0) buffer.write(' ');
        buffer.write(reversed[i]);
      }
      return '€${buffer.toString().split('').reversed.join()}';
    }
    return '€${euros.toStringAsFixed(0)}';
  }

  String _formatDate(DateTime dt) {
    return '${dt.day.toString().padLeft(2, '0')}/'
        '${dt.month.toString().padLeft(2, '0')}/'
        '${dt.year}';
  }
}

// ---------------------------------------------------------------------------
// Provider chip — deep-links to `/profiles/{orgId}`
// ---------------------------------------------------------------------------

class _ProviderChip extends StatelessWidget {
  const _ProviderChip({required this.provider});

  final ClientProjectProvider provider;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final initials = _initials(provider.displayName);
    final hasTargetOrg = provider.organizationId.isNotEmpty;

    final content = Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
      decoration: BoxDecoration(
        color: appColors?.muted ?? theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          CircleAvatar(
            radius: 10,
            backgroundColor:
                theme.colorScheme.primary.withValues(alpha: 0.15),
            backgroundImage: provider.avatarUrl != null &&
                    provider.avatarUrl!.isNotEmpty
                ? CachedNetworkImageProvider(provider.avatarUrl!)
                : null,
            child: provider.avatarUrl == null || provider.avatarUrl!.isEmpty
                ? Text(
                    initials,
                    style: TextStyle(
                      fontSize: 10,
                      fontWeight: FontWeight.w700,
                      color: theme.colorScheme.primary,
                    ),
                  )
                : null,
          ),
          const SizedBox(width: 6),
          Text(
            provider.displayName.isEmpty ? '—' : provider.displayName,
            style: theme.textTheme.bodySmall?.copyWith(
              fontWeight: FontWeight.w500,
            ),
          ),
          if (hasTargetOrg) ...[
            const SizedBox(width: 4),
            Icon(
              Icons.chevron_right,
              size: 14,
              color: appColors?.mutedForeground,
            ),
          ],
        ],
      ),
    );

    if (!hasTargetOrg) return content;
    return InkWell(
      borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      onTap: () {
        context.push(
          '/profiles/${provider.organizationId}',
          extra: {'display_name': provider.displayName},
        );
      },
      child: content,
    );
  }

  String _initials(String name) {
    final trimmed = name.trim();
    if (trimmed.isEmpty) return '?';
    final parts = trimmed.split(RegExp(r'\s+'));
    if (parts.length == 1) return parts[0][0].toUpperCase();
    return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
  }
}
