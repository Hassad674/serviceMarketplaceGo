import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';

/// Displays a typing indicator bar showing who is currently typing.
class TypingIndicatorWidget extends StatelessWidget {
  const TypingIndicatorWidget({super.key, required this.userName});

  final String userName;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      child: Text(
        AppLocalizations.of(context)!.messagingTyping(userName),
        style: TextStyle(
          fontSize: 12,
          fontStyle: FontStyle.italic,
          color: appColors?.mutedForeground,
        ),
      ),
    );
  }
}
