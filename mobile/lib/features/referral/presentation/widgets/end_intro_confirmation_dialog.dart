import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';

/// EndIntroConfirmationDialog — Run D parity with web Run C
/// `EndIntroConfirmationModal`. AlertDialog with a destructive
/// "Terminer définitivement" confirm and a "Annuler" cancel button.
/// The body interpolates the provider + client names with fallbacks
/// when one is missing.
///
/// Returns `true` when the user confirms, `false` (or null) when
/// they cancel.
Future<bool?> showEndIntroConfirmationDialog({
  required BuildContext context,
  String? providerName,
  String? clientName,
  bool pending = false,
}) {
  return showDialog<bool>(
    context: context,
    barrierDismissible: !pending,
    builder: (dialogContext) => EndIntroConfirmationDialog(
      providerName: providerName,
      clientName: clientName,
      pending: pending,
    ),
  );
}

class EndIntroConfirmationDialog extends StatelessWidget {
  const EndIntroConfirmationDialog({
    super.key,
    this.providerName,
    this.clientName,
    this.pending = false,
  });

  final String? providerName;
  final String? clientName;
  final bool pending;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final providerLabel = (providerName != null && providerName!.isNotEmpty)
        ? providerName!
        : l10n.referralEndIntroModalFallbackProvider;
    final clientLabel = (clientName != null && clientName!.isNotEmpty)
        ? clientName!
        : l10n.referralEndIntroModalFallbackClient;
    return AlertDialog(
      key: const ValueKey('end-intro-confirmation-dialog'),
      title: Text(l10n.referralEndIntroModalTitle),
      content: Text(
        l10n.referralEndIntroModalBody(providerLabel, clientLabel),
        style: theme.textTheme.bodyMedium,
      ),
      actions: [
        TextButton(
          key: const ValueKey('end-intro-cancel'),
          onPressed:
              pending ? null : () => Navigator.of(context).pop(false),
          child: Text(l10n.referralEndIntroModalCancel),
        ),
        FilledButton(
          key: const ValueKey('end-intro-confirm'),
          style: FilledButton.styleFrom(
            backgroundColor: theme.colorScheme.error,
            foregroundColor: theme.colorScheme.onError,
          ),
          onPressed:
              pending ? null : () => Navigator.of(context).pop(true),
          child: pending
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    valueColor:
                        AlwaysStoppedAnimation<Color>(Colors.white),
                  ),
                )
              : Text(l10n.referralEndIntroModalConfirm),
        ),
      ],
    );
  }
}
