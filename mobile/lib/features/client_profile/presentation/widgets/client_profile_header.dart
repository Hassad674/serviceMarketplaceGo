import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';

/// Compact header for the client profile — avatar + company name +
/// 4 key stats (total spent, review count, average rating, projects
/// completed). Safe to use on both the private editable screen and
/// the public read-only screen.
///
/// The avatar is NOT tappable in this widget — callers wrap it with
/// a `GestureDetector` when they want to enable editing (private
/// screen only). This keeps the widget pure and trivially testable.
class ClientProfileHeader extends StatelessWidget {
  const ClientProfileHeader({
    super.key,
    required this.companyName,
    required this.totalSpentCents,
    required this.reviewCount,
    required this.averageRating,
    required this.projectsCompleted,
    this.avatarUrl,
    this.orgType,
    this.onAvatarTap,
  });

  final String companyName;
  final String? avatarUrl;

  /// Organization type — used to pick the badge color. Expected values:
  /// `agency`, `enterprise`, `provider_personal`. Null hides the badge.
  final String? orgType;

  final int totalSpentCents;
  final int reviewCount;
  final double averageRating;
  final int projectsCompleted;

  /// Optional tap handler used on the private screen to trigger the
  /// avatar upload bottom sheet. When null the avatar renders as a
  /// plain [CircleAvatar].
  final VoidCallback? onAvatarTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();
    final initials = _computeInitials(companyName);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        children: [
          _Avatar(
            initials: initials,
            avatarUrl: avatarUrl,
            onTap: onAvatarTap,
          ),
          const SizedBox(height: 12),
          Text(
            companyName.isNotEmpty ? companyName : '—',
            style: theme.textTheme.titleLarge,
            textAlign: TextAlign.center,
          ),
          if (orgType != null && orgType!.isNotEmpty) ...[
            const SizedBox(height: 8),
            _OrgTypeBadge(orgType: orgType!),
          ],
          const SizedBox(height: 16),
          _StatsGrid(
            totalSpentLabel: l10n.clientProfileTotalSpent,
            reviewsLabel: l10n.clientProfileReviewsReceived,
            ratingLabel: l10n.clientProfileAverageRating,
            projectsLabel: l10n.clientProfileProjectsCompleted,
            totalSpentCents: totalSpentCents,
            reviewCount: reviewCount,
            averageRating: averageRating,
            projectsCompleted: projectsCompleted,
            labelColor: appColors?.mutedForeground,
          ),
        ],
      ),
    );
  }

  String _computeInitials(String value) {
    final trimmed = value.trim();
    if (trimmed.isEmpty) return '?';
    final parts = trimmed.split(RegExp(r'\s+'));
    if (parts.length == 1) return parts[0][0].toUpperCase();
    return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
  }
}

// ---------------------------------------------------------------------------
// Avatar — circle with photo or initials, optional tap
// ---------------------------------------------------------------------------

class _Avatar extends StatelessWidget {
  const _Avatar({
    required this.initials,
    this.avatarUrl,
    this.onTap,
  });

  final String initials;
  final String? avatarUrl;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    final child = Stack(
      children: [
        CircleAvatar(
          radius: 40,
          backgroundColor: primary.withValues(alpha: 0.1),
          backgroundImage: _image(),
          child: _image() == null
              ? Text(
                  initials,
                  style: TextStyle(
                    fontSize: 22,
                    fontWeight: FontWeight.bold,
                    color: primary,
                  ),
                )
              : null,
        ),
        if (onTap != null)
          Positioned(
            bottom: 0,
            right: 0,
            child: Container(
              width: 26,
              height: 26,
              decoration: BoxDecoration(
                color: primary,
                shape: BoxShape.circle,
                border: Border.all(
                  color: theme.colorScheme.surface,
                  width: 2,
                ),
              ),
              child: const Icon(
                Icons.camera_alt,
                size: 14,
                color: Colors.white,
              ),
            ),
          ),
      ],
    );

    if (onTap == null) return child;
    return GestureDetector(onTap: onTap, child: child);
  }

  ImageProvider? _image() {
    if (avatarUrl == null || avatarUrl!.isEmpty) return null;
    // 40 lp radius × 3x DPR = ~240 px. Cap at 256 (PERF-M-05).
    return CachedNetworkImageProvider(
      avatarUrl!,
      maxWidth: 256,
      maxHeight: 256,
    );
  }
}

// ---------------------------------------------------------------------------
// Org-type badge
// ---------------------------------------------------------------------------

class _OrgTypeBadge extends StatelessWidget {
  const _OrgTypeBadge({required this.orgType});

  final String orgType;

  @override
  Widget build(BuildContext context) {
    final (label, color) = _style(orgType);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: color,
          fontSize: 12,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }

  (String, Color) _style(String type) {
    switch (type) {
      case 'agency':
        return ('Agency', const Color(0xFF2563EB));
      case 'enterprise':
        return ('Enterprise', const Color(0xFF8B5CF6));
      case 'provider_personal':
        return ('Freelance', const Color(0xFFF43F5E));
      default:
        return (type, const Color(0xFF64748B));
    }
  }
}

// ---------------------------------------------------------------------------
// Stats grid — 2x2 layout of key client metrics
// ---------------------------------------------------------------------------

class _StatsGrid extends StatelessWidget {
  const _StatsGrid({
    required this.totalSpentLabel,
    required this.reviewsLabel,
    required this.ratingLabel,
    required this.projectsLabel,
    required this.totalSpentCents,
    required this.reviewCount,
    required this.averageRating,
    required this.projectsCompleted,
    this.labelColor,
  });

  final String totalSpentLabel;
  final String reviewsLabel;
  final String ratingLabel;
  final String projectsLabel;

  final int totalSpentCents;
  final int reviewCount;
  final double averageRating;
  final int projectsCompleted;

  final Color? labelColor;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: _StatTile(
            label: totalSpentLabel,
            value: _formatEuros(totalSpentCents),
            labelColor: labelColor,
          ),
        ),
        Expanded(
          child: _StatTile(
            label: projectsLabel,
            value: projectsCompleted.toString(),
            labelColor: labelColor,
          ),
        ),
        Expanded(
          child: _StatTile(
            label: ratingLabel,
            value: averageRating > 0
                ? averageRating.toStringAsFixed(1)
                : '—',
            labelColor: labelColor,
            icon: averageRating > 0 ? Icons.star : null,
            iconColor: const Color(0xFFFBBF24),
          ),
        ),
        Expanded(
          child: _StatTile(
            label: reviewsLabel,
            value: reviewCount.toString(),
            labelColor: labelColor,
          ),
        ),
      ],
    );
  }

  String _formatEuros(int cents) {
    if (cents <= 0) return '€0';
    final euros = cents / 100;
    if (euros >= 1000) {
      // Group thousands with a thin space to stay readable on
      // narrow phone widths without depending on `intl`.
      final whole = euros.toStringAsFixed(0);
      final buffer = StringBuffer();
      final reversed = whole.split('').reversed.toList();
      for (var i = 0; i < reversed.length; i++) {
        if (i > 0 && i % 3 == 0) buffer.write(' ');
        buffer.write(reversed[i]);
      }
      return '€${buffer.toString().split('').reversed.join()}';
    }
    return '€${euros.toStringAsFixed(0)}';
  }
}

class _StatTile extends StatelessWidget {
  const _StatTile({
    required this.label,
    required this.value,
    this.labelColor,
    this.icon,
    this.iconColor,
  });

  final String label;
  final String value;
  final Color? labelColor;
  final IconData? icon;
  final Color? iconColor;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.center,
          mainAxisSize: MainAxisSize.min,
          children: [
            if (icon != null) ...[
              Icon(icon, size: 14, color: iconColor),
              const SizedBox(width: 3),
            ],
            Text(
              value,
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w700,
              ),
            ),
          ],
        ),
        const SizedBox(height: 2),
        Text(
          label,
          textAlign: TextAlign.center,
          style: theme.textTheme.bodySmall?.copyWith(
            color: labelColor,
          ),
        ),
      ],
    );
  }
}
