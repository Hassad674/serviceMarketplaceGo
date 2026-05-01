import 'package:flutter/material.dart';

import '../../../../../../l10n/app_localizations.dart';
import '../../../../domain/entities/message_entity.dart';

/// Shows a dialog allowing the user to edit a message's content.
///
/// Calls [onConfirm] with the trimmed new content when the user taps
/// "Save". Dialog is dismissed first so the caller never has to manage
/// `Navigator.pop()`.
Future<void> showEditMessageDialog({
  required BuildContext context,
  required MessageEntity message,
  required Future<void> Function(String content) onConfirm,
}) async {
  final editController = TextEditingController(text: message.content);
  final l10n = AppLocalizations.of(context)!;

  await showDialog<void>(
    context: context,
    builder: (ctx) => AlertDialog(
      title: Text(l10n.messagingEditMessage),
      content: TextField(
        controller: editController,
        autofocus: true,
        maxLines: null,
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(ctx),
          child: Text(l10n.cancel),
        ),
        ElevatedButton(
          onPressed: () async {
            Navigator.pop(ctx);
            await onConfirm(editController.text.trim());
          },
          child: Text(l10n.save),
        ),
      ],
    ),
  );
}
