import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';

import '../../core/widgets/portrait.dart';

/// Generic identity header used by both freelance and referrer
/// profile screens. Renders an avatar, display name, title and a
/// trailing slot (typically an [AvailabilityPill]). Pure display
/// widget — no business logic, no repository access, no feature
/// imports.
class ProfileIdentityHeader extends StatelessWidget {
  const ProfileIdentityHeader({
    super.key,
    required this.displayName,
    required this.initials,
    required this.accentColor,
    this.title,
    this.photoUrl,
    this.trailing,
    this.onPhotoTap,
    this.showEditBadge = false,
    this.portraitSeed,
  });

  /// Full display name rendered under the avatar.
  final String displayName;

  /// Fallback initials when the photo is missing AND no [portraitSeed]
  /// is provided. Kept for backward compatibility with existing
  /// callers that opted out of the Soleil v2 [Portrait] illustration.
  final String initials;

  /// Persona accent color (rose for freelance, teal for referrer).
  final Color accentColor;

  /// Optional professional title (e.g. "Full-stack engineer").
  final String? title;

  /// Optional persistent photo URL. Nil renders the initials or the
  /// stylized [Portrait] (when [portraitSeed] is supplied).
  final String? photoUrl;

  /// Optional trailing widget rendered next to the name — typically
  /// an availability pill.
  final Widget? trailing;

  /// Optional tap handler. When non-null a small camera badge is
  /// rendered over the avatar.
  final VoidCallback? onPhotoTap;

  /// Forces the camera badge even when `onPhotoTap` is null. Useful
  /// when upstream code wants to visually hint that the header is
  /// editable from a different surface.
  final bool showEditBadge;

  /// When non-null and no [photoUrl] is set, renders a deterministic
  /// stylized [Portrait] illustration instead of the initials avatar.
  /// The seed selects one of 6 Soleil palettes via `seed % 6`.
  final int? portraitSeed;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      children: [
        _Avatar(
          initials: initials,
          accentColor: accentColor,
          photoUrl: photoUrl,
          onTap: onPhotoTap,
          showEditBadge: showEditBadge || onPhotoTap != null,
          portraitSeed: portraitSeed,
        ),
        const SizedBox(height: 12),
        Text(
          displayName.isEmpty ? '—' : displayName,
          style: theme.textTheme.titleLarge,
          textAlign: TextAlign.center,
        ),
        if (title != null && title!.isNotEmpty) ...[
          const SizedBox(height: 4),
          Text(
            title!,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
            textAlign: TextAlign.center,
          ),
        ],
        if (trailing != null) ...[
          const SizedBox(height: 12),
          trailing!,
        ],
      ],
    );
  }
}

class _Avatar extends StatelessWidget {
  const _Avatar({
    required this.initials,
    required this.accentColor,
    required this.showEditBadge,
    this.photoUrl,
    this.onTap,
    this.portraitSeed,
  });

  final String initials;
  final Color accentColor;
  final String? photoUrl;
  final VoidCallback? onTap;
  final bool showEditBadge;
  final int? portraitSeed;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasPhoto = photoUrl != null && photoUrl!.isNotEmpty;
    final hasPortrait = !hasPhoto && portraitSeed != null;

    final Widget avatar;
    if (hasPortrait) {
      // 48 lp radius == 96 lp diameter — match the previous CircleAvatar
      // footprint exactly so layouts that depend on the avatar's size
      // (column gaps, edit-badge offset) remain visually identical.
      avatar = Portrait(id: portraitSeed!, size: 96);
    } else {
      avatar = CircleAvatar(
        radius: 48,
        backgroundColor: accentColor.withValues(alpha: 0.1),
        // 48 lp radius = 96 lp diameter; 3x DPR = ~288 px. 256 is the
        // next 2-power that fits and gives crisp rendering on tablets.
        // Avoids decoding the original full-res JPEG to RAM (PERF-M-05).
        backgroundImage: hasPhoto
            ? CachedNetworkImageProvider(
                photoUrl!,
                maxWidth: 256,
                maxHeight: 256,
              )
            : null,
        child: hasPhoto
            ? null
            : Text(
                initials,
                style: TextStyle(
                  fontSize: 28,
                  fontWeight: FontWeight.bold,
                  color: accentColor,
                ),
              ),
      );
    }

    return GestureDetector(
      onTap: onTap,
      child: Stack(
        children: [
          avatar,
          if (showEditBadge)
            Positioned(
              bottom: 0,
              right: 0,
              child: Container(
                width: 32,
                height: 32,
                decoration: BoxDecoration(
                  color: accentColor,
                  shape: BoxShape.circle,
                  border: Border.all(
                    color: theme.colorScheme.surface,
                    width: 2,
                  ),
                ),
                child: const Icon(
                  Icons.camera_alt,
                  size: 16,
                  color: Colors.white,
                ),
              ),
            ),
        ],
      ),
    );
  }
}
