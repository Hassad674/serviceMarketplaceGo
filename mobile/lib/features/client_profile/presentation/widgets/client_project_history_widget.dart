import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/project_history_card.dart';
import '../../domain/entities/client_profile.dart';
import '../../../../core/theme/app_palette.dart';

/// Unified "Project history" section for the client profile.
///
/// Mirrors the provider-side pattern — one card per completed mission
/// with the matching provider→client review embedded inline, falling
/// back to an "Awaiting review" placeholder when the review has not
/// been submitted yet. The provider chip is rendered as the shared
/// card's footer so users can jump to the provider's public profile.
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

    if (projects.isEmpty) {
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
            _Header(
              title: l10n.clientProfileProjectHistoryTitle,
              subtitle: null,
            ),
            const SizedBox(height: 12),
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Text(
                l10n.clientProfileProjectHistoryEmpty,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: appColors?.mutedForeground,
                  fontStyle: FontStyle.italic,
                ),
              ),
            ),
          ],
        ),
      );
    }

    final subtitle = '${projects.length} '
        '${projects.length > 1 ? 'projects' : 'project'}';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
          child: _Header(
            title: l10n.clientProfileProjectHistoryTitle,
            subtitle: subtitle,
          ),
        ),
        ...projects.map(
          (entry) => ProjectHistoryCard(
            title: entry.title,
            amountCents: entry.amount,
            completedAt: entry.completedAt,
            review: entry.review,
            footer: _ProviderChip(provider: entry.provider),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Header — gradient history icon + title + optional count subtitle
// ---------------------------------------------------------------------------

class _Header extends StatelessWidget {
  const _Header({required this.title, required this.subtitle});

  final String title;
  final String? subtitle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Row(
      children: [
        Container(
          width: 36,
          height: 36,
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(10),
            gradient: const LinearGradient(
              colors: [AppPalette.rose100, AppPalette.red50],
            ),
          ),
          child: const Icon(
            Icons.history,
            size: 18,
            color: AppPalette.rose600,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                title,
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
              ),
              if (subtitle != null)
                Text(
                  subtitle!,
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
            ],
          ),
        ),
      ],
    );
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
            // Tiny avatar: 10 lp = 60 px @ 3x DPR. 64 is enough.
            backgroundImage: provider.avatarUrl != null &&
                    provider.avatarUrl!.isNotEmpty
                ? CachedNetworkImageProvider(
                    provider.avatarUrl!,
                    maxWidth: 64,
                    maxHeight: 64,
                  )
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
