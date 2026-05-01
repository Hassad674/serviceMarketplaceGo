import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';

/// Outlined logout button rendered at the bottom of the freelance and
/// referrer profile screens.
class FreelanceLogoutButton extends StatelessWidget {
  const FreelanceLogoutButton({super.key, required this.onPressed});

  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return SizedBox(
      width: double.infinity,
      child: OutlinedButton.icon(
        onPressed: onPressed,
        icon: Icon(Icons.logout, color: theme.colorScheme.error),
        label: Text(
          l10n.signOut,
          style: TextStyle(color: theme.colorScheme.error),
        ),
        style: OutlinedButton.styleFrom(
          side: BorderSide(
            color: theme.colorScheme.error.withValues(alpha: 0.3),
          ),
          minimumSize: const Size(double.infinity, 48),
        ),
      ),
    );
  }
}
