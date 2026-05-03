import 'package:flutter/material.dart';

import '../../../../../../l10n/app_localizations.dart';
import '../../../../domain/entities/message_entity.dart';
import '../../../../../../core/theme/app_palette.dart';

/// Compact preview of the replied-to message rendered above the new
/// bubble's content.
class ReplyPreviewWidget extends StatelessWidget {
  const ReplyPreviewWidget({
    super.key,
    required this.replyTo,
    required this.isOwn,
  });

  final ReplyToInfo replyTo;
  final bool isOwn;

  @override
  Widget build(BuildContext context) {
    final truncated = replyTo.content.length > 50
        ? '${replyTo.content.substring(0, 50)}...'
        : replyTo.content;

    return Container(
      margin: const EdgeInsets.only(bottom: 6),
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        border: const Border(
          left: BorderSide(color: AppPalette.rose500, width: 2),
        ),
        color: isOwn
            ? Colors.white.withValues(alpha: 0.15)
            : AppPalette.rose500.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        truncated.isEmpty
            ? AppLocalizations.of(context)!.messagingDeleted
            : truncated,
        style: TextStyle(
          fontSize: 12,
          color: isOwn
              ? Colors.white.withValues(alpha: 0.8)
              : AppPalette.slate500,
        ),
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
      ),
    );
  }
}
