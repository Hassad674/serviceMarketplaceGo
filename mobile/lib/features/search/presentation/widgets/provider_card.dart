import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/theme/app_theme.dart';

/// A card displaying an organization's public marketplace summary.
///
/// Shows avatar (photo or initials), org name, title, and a colored
/// org-type badge. Tapping navigates to the public profile screen.
///
/// Since phase R2 this row describes the organization behind the
/// offering, not an individual user — the payload follows the web
/// PublicProfileSummary shape (organization_id / name / org_type / …).
class ProviderCard extends StatelessWidget {
  const ProviderCard({super.key, required this.profile});

  final Map<String, dynamic> profile;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    final orgId = profile['organization_id'] as String? ?? '';
    final displayName = _resolveDisplayName();
    final title = profile['title'] as String?;
    final photoUrl = profile['photo_url'] as String?;
    final orgType = profile['org_type'] as String?;
    final initials = _buildInitials(displayName);

    return GestureDetector(
      onTap: () => context.push(
        '/profiles/$orgId',
        extra: <String, dynamic>{
          'display_name': displayName,
          'org_type': orgType,
        },
      ),
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(AppTheme.radiusLg),
          border: Border.all(
            color: appColors?.border ?? theme.dividerColor,
          ),
          boxShadow: AppTheme.cardShadow,
        ),
        child: Row(
          children: [
            _Avatar(
              photoUrl: photoUrl,
              initials: initials,
              roleColor: _roleColor(orgType),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    displayName,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                      fontSize: 15,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 2),
                  Text(
                    title ?? 'No title',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: appColors?.mutedForeground,
                      fontSize: 13,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  if ((profile['review_count'] as int? ?? 0) > 0) ...[
                    const SizedBox(height: 4),
                    Row(
                      children: [
                        const Icon(
                          Icons.star,
                          size: 12,
                          color: Color(0xFFFBBF24),
                        ),
                        const SizedBox(width: 3),
                        Text(
                          (profile['average_rating'] as num? ?? 0)
                              .toDouble()
                              .toStringAsFixed(1),
                          style: theme.textTheme.labelSmall?.copyWith(
                            fontWeight: FontWeight.w600,
                            fontSize: 11,
                          ),
                        ),
                        const SizedBox(width: 3),
                        Text(
                          '(${profile['review_count']})',
                          style: theme.textTheme.labelSmall?.copyWith(
                            color: appColors?.mutedForeground,
                            fontSize: 11,
                          ),
                        ),
                      ],
                    ),
                  ],
                ],
              ),
            ),
            const SizedBox(width: 8),
            _OrgTypeBadge(orgType: orgType),
          ],
        ),
      ),
    );
  }

  String _resolveDisplayName() {
    final name = profile['name'] as String?;
    return (name != null && name.isNotEmpty) ? name : 'Unknown';
  }

  String _buildInitials(String name) {
    if (name.isEmpty || name == 'Unknown') return '?';
    final parts = name.trim().split(RegExp(r'\s+'));
    if (parts.length == 1) return parts[0][0].toUpperCase();
    return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
  }

  Color _roleColor(String? orgType) {
    switch (orgType) {
      case 'agency':
        return const Color(0xFF2563EB); // blue-600
      case 'enterprise':
        return const Color(0xFF8B5CF6); // violet-500
      case 'provider_personal':
        return const Color(0xFFF43F5E); // rose-500
      default:
        return const Color(0xFF64748B); // slate-500
    }
  }
}

// ---------------------------------------------------------------------------
// Avatar — circular with CachedNetworkImage or initials fallback
// ---------------------------------------------------------------------------

class _Avatar extends StatelessWidget {
  const _Avatar({
    required this.initials,
    required this.roleColor,
    this.photoUrl,
  });

  final String? photoUrl;
  final String initials;
  final Color roleColor;

  @override
  Widget build(BuildContext context) {
    if (photoUrl != null && photoUrl!.isNotEmpty) {
      return CachedNetworkImage(
        imageUrl: photoUrl!,
        imageBuilder: (context, imageProvider) => CircleAvatar(
          radius: 24,
          backgroundImage: imageProvider,
        ),
        placeholder: (context, url) => CircleAvatar(
          radius: 24,
          backgroundColor: roleColor.withValues(alpha: 0.1),
          child: Text(
            initials,
            style: TextStyle(
              color: roleColor,
              fontWeight: FontWeight.w600,
              fontSize: 14,
            ),
          ),
        ),
        errorWidget: (context, url, error) => _InitialsAvatar(
          initials: initials,
          color: roleColor,
        ),
      );
    }

    return _InitialsAvatar(initials: initials, color: roleColor);
  }
}

class _InitialsAvatar extends StatelessWidget {
  const _InitialsAvatar({
    required this.initials,
    required this.color,
  });

  final String initials;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return CircleAvatar(
      radius: 24,
      backgroundColor: color.withValues(alpha: 0.1),
      child: Text(
        initials,
        style: TextStyle(
          color: color,
          fontWeight: FontWeight.w600,
          fontSize: 14,
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Org-type badge — colored pill
// ---------------------------------------------------------------------------

class _OrgTypeBadge extends StatelessWidget {
  const _OrgTypeBadge({required this.orgType});

  final String? orgType;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: _color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        _label,
        style: TextStyle(
          color: _color,
          fontWeight: FontWeight.w600,
          fontSize: 11,
        ),
      ),
    );
  }

  String get _label {
    switch (orgType) {
      case 'agency':
        return 'Agency';
      case 'enterprise':
        return 'Enterprise';
      case 'provider_personal':
        return 'Freelance';
      default:
        return orgType ?? 'Unknown';
    }
  }

  Color get _color {
    switch (orgType) {
      case 'agency':
        return const Color(0xFF2563EB);
      case 'enterprise':
        return const Color(0xFF8B5CF6);
      case 'provider_personal':
        return const Color(0xFFF43F5E);
      default:
        return const Color(0xFF64748B);
    }
  }
}
