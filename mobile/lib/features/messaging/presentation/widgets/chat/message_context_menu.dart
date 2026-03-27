import 'package:flutter/material.dart';

import '../../../../../l10n/app_localizations.dart';

/// Shows a bottom sheet with reply/edit/delete actions for a message.
void showMessageContextMenu({
  required BuildContext context,
  required AppLocalizations l10n,
  VoidCallback? onReply,
  VoidCallback? onEdit,
  VoidCallback? onDelete,
}) {
  showModalBottomSheet(
    context: context,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
    ),
    builder: (ctx) => SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (onReply != null)
            ListTile(
              leading: const Icon(Icons.reply_outlined),
              title: Text(l10n.messagingReply),
              onTap: () {
                Navigator.pop(ctx);
                onReply();
              },
            ),
          if (onEdit != null)
            ListTile(
              leading: const Icon(Icons.edit_outlined),
              title: Text(l10n.messagingEditMessage),
              onTap: () {
                Navigator.pop(ctx);
                onEdit();
              },
            ),
          if (onDelete != null)
            ListTile(
              leading: Icon(
                Icons.delete_outline,
                color: Theme.of(context).colorScheme.error,
              ),
              title: Text(
                l10n.messagingDeleteMessage,
                style: TextStyle(
                  color: Theme.of(context).colorScheme.error,
                ),
              ),
              onTap: () {
                Navigator.pop(ctx);
                onDelete();
              },
            ),
        ],
      ),
    ),
  );
}
