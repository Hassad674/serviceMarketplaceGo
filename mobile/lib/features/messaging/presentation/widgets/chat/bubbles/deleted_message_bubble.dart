import 'package:flutter/material.dart';

import '../../../../../../core/theme/app_theme.dart';
import '../../../../../../l10n/app_localizations.dart';
import '../../../../../../core/theme/app_palette.dart';

/// Bubble shown in place of a deleted message — italic placeholder
/// text behind a muted border.
class DeletedMessageBubble extends StatelessWidget {
  const DeletedMessageBubble({super.key, required this.isOwn});

  final bool isOwn;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Align(
        alignment: isOwn ? Alignment.centerRight : Alignment.centerLeft,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          decoration: BoxDecoration(
            color: appColors?.muted ?? AppPalette.slate100,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(
              color: appColors?.border ?? theme.dividerColor,
            ),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.block,
                size: 14,
                color: appColors?.mutedForeground,
              ),
              const SizedBox(width: 6),
              Text(
                l10n.messagingDeleted,
                style: TextStyle(
                  fontSize: 13,
                  fontStyle: FontStyle.italic,
                  color: appColors?.mutedForeground,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
