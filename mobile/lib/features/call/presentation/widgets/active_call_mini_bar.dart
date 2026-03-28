import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';

/// A small green bar displayed at the top of the screen when the user
/// navigates away from the [CallScreen] while a call is still active.
///
/// Tapping it navigates back to the call screen.
class ActiveCallMiniBar extends StatelessWidget {
  const ActiveCallMiniBar({
    super.key,
    required this.participantName,
    required this.durationSeconds,
    required this.onTap,
  });

  final String participantName;
  final int durationSeconds;
  final VoidCallback onTap;

  String get _formattedDuration {
    final m = (durationSeconds ~/ 60).toString().padLeft(2, '0');
    final s = (durationSeconds % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return GestureDetector(
      onTap: onTap,
      child: SafeArea(
        bottom: false,
        child: Container(
          height: 44,
          decoration: const BoxDecoration(
            color: Color(0xFF22C55E),
            borderRadius: BorderRadius.only(
              bottomLeft: Radius.circular(12),
              bottomRight: Radius.circular(12),
            ),
          ),
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Row(
            children: [
              // Pulsing green dot
              Container(
                width: 8,
                height: 8,
                decoration: const BoxDecoration(
                  color: Colors.white,
                  shape: BoxShape.circle,
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  '$participantName  $_formattedDuration',
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              Text(
                l10n.callTapToReturn,
                style: TextStyle(
                  color: Colors.white.withValues(alpha: 0.85),
                  fontSize: 12,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
