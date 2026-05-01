import 'package:flutter/material.dart';

import '../../../../../../l10n/app_localizations.dart';

/// Shows a confirmation dialog asking the user whether to delete a
/// message. Calls [onConfirm] when the destructive action is selected.
Future<void> showDeleteMessageDialog({
  required BuildContext context,
  required Future<void> Function() onConfirm,
}) async {
  final l10n = AppLocalizations.of(context)!;

  await showDialog<void>(
    context: context,
    builder: (ctx) => AlertDialog(
      title: Text(l10n.messagingDeleteMessage),
      content: Text(l10n.messagingDeleteConfirm),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(ctx),
          child: Text(l10n.cancel),
        ),
        ElevatedButton(
          style: ElevatedButton.styleFrom(
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
          onPressed: () async {
            Navigator.pop(ctx);
            await onConfirm();
          },
          child: Text(l10n.remove),
        ),
      ],
    ),
  );
}
