import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import 'profile_section_card.dart';

/// "About" card on the legacy profile screen — long-form bio with a
/// localised placeholder when empty. Tapping the card invokes [onTap]
/// (used to open the edit bottom sheet).
class ProfileAboutSection extends StatelessWidget {
  const ProfileAboutSection({
    super.key,
    this.about,
    this.onTap,
  });

  final String? about;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final hasAbout = about != null && about!.isNotEmpty;

    return GestureDetector(
      onTap: onTap,
      child: ProfileSectionCard(
        title: l10n.about,
        icon: Icons.info_outline,
        child: SizedBox(
          width: double.infinity,
          child: hasAbout
              ? Text(
                  about!,
                  softWrap: true,
                  style: theme.textTheme.bodyMedium?.copyWith(height: 1.5),
                )
              : Text(
                  l10n.aboutPlaceholder,
                  softWrap: true,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: appColors?.mutedForeground,
                    fontStyle: FontStyle.italic,
                  ),
                ),
        ),
      ),
    );
  }
}

/// "Professional title" card — short-form headline with localised
/// placeholder when empty. Read-only on the legacy screen.
class ProfileTitleSection extends StatelessWidget {
  const ProfileTitleSection({super.key, required this.title});

  final String? title;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final hasTitle = title != null && title!.isNotEmpty;

    return ProfileSectionCard(
      title: l10n.professionalTitle,
      icon: Icons.badge_outlined,
      child: hasTitle
          ? Text(title!, style: theme.textTheme.bodyMedium)
          : Text(
              l10n.titlePlaceholder,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: appColors?.mutedForeground,
                fontStyle: FontStyle.italic,
              ),
            ),
    );
  }
}
