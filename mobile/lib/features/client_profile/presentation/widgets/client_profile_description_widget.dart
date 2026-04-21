import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';

/// "About the company" card. Read-only rendering of
/// `client_description` — editing is handled by the private screen's
/// edit sheet.
class ClientProfileDescriptionWidget extends StatelessWidget {
  const ClientProfileDescriptionWidget({
    super.key,
    required this.description,
    this.onTap,
  });

  final String description;

  /// When non-null the card shows an edit affordance (pencil icon) and
  /// opens the bottom sheet on tap. Used by the private screen.
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();
    final isEmpty = description.trim().isEmpty;

    final card = Container(
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
                Icons.info_outline,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  l10n.clientProfileDescription,
                  style: theme.textTheme.titleMedium,
                ),
              ),
              if (onTap != null)
                Icon(
                  Icons.edit_outlined,
                  size: 18,
                  color: appColors?.mutedForeground,
                ),
            ],
          ),
          const SizedBox(height: 12),
          SizedBox(
            width: double.infinity,
            child: Text(
              isEmpty ? l10n.clientProfileDescriptionHint : description,
              softWrap: true,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: isEmpty ? appColors?.mutedForeground : null,
                fontStyle: isEmpty ? FontStyle.italic : null,
                height: 1.5,
              ),
            ),
          ),
        ],
      ),
    );

    if (onTap == null) return card;
    return InkWell(
      borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      onTap: onTap,
      child: card,
    );
  }
}
