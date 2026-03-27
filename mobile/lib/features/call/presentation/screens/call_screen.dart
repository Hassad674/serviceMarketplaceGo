import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/call_entity.dart';
import '../providers/call_provider.dart';

/// Full-screen view shown during an active audio call.
class CallScreen extends ConsumerWidget {
  const CallScreen({super.key, this.recipientName = ''});

  final String recipientName;

  String _formatDuration(int seconds) {
    final m = (seconds ~/ 60).toString().padLeft(2, '0');
    final s = (seconds % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(callProvider);
    final notifier = ref.read(callProvider.notifier);
    final l10n = AppLocalizations.of(context)!;
    final isRinging = state.status == CallStatus.ringingOutgoing;

    final initials = recipientName.isNotEmpty
        ? recipientName
            .split(' ')
            .map((w) => w.isNotEmpty ? w[0] : '')
            .join()
            .substring(0, recipientName.split(' ').length.clamp(0, 2))
            .toUpperCase()
        : '?';

    return Scaffold(
      backgroundColor: const Color(0xFF0F172A),
      body: SafeArea(
        child: Column(
          children: [
            const Spacer(),

            // Avatar
            Container(
              width: 96,
              height: 96,
              decoration: const BoxDecoration(
                shape: BoxShape.circle,
                gradient: LinearGradient(
                  colors: [Color(0xFFF43F5E), Color(0xFF8B5CF6)],
                ),
              ),
              child: Center(
                child: Text(
                  initials,
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 32,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ),
            ),
            const SizedBox(height: 24),

            // Name
            Text(
              recipientName.isNotEmpty ? recipientName : l10n.callAudioCall,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 24,
                fontWeight: FontWeight.bold,
              ),
            ),
            const SizedBox(height: 8),

            // Status / timer
            Text(
              isRinging ? l10n.callCalling : _formatDuration(state.duration),
              style: TextStyle(
                color: Colors.white.withValues(alpha: 0.7),
                fontSize: 16,
                fontFamily: isRinging ? null : 'monospace',
              ),
            ),

            const Spacer(),

            // Controls
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                // Mute
                _CallControlButton(
                  icon: state.isMuted ? Icons.mic_off : Icons.mic,
                  label: state.isMuted ? l10n.callUnmute : l10n.callMute,
                  isActive: state.isMuted,
                  onPressed: notifier.toggleMute,
                ),
                const SizedBox(width: 48),
                // Hang up
                _CallControlButton(
                  icon: Icons.call_end,
                  label: l10n.callHangup,
                  isDestructive: true,
                  onPressed: () {
                    notifier.endCall();
                    Navigator.of(context).pop();
                  },
                ),
              ],
            ),
            const SizedBox(height: 48),
          ],
        ),
      ),
    );
  }
}

class _CallControlButton extends StatelessWidget {
  const _CallControlButton({
    required this.icon,
    required this.label,
    required this.onPressed,
    this.isActive = false,
    this.isDestructive = false,
  });

  final IconData icon;
  final String label;
  final VoidCallback onPressed;
  final bool isActive;
  final bool isDestructive;

  @override
  Widget build(BuildContext context) {
    final bgColor = isDestructive
        ? const Color(0xFFEF4444)
        : isActive
            ? const Color(0xFFEF4444).withValues(alpha: 0.2)
            : Colors.white.withValues(alpha: 0.1);

    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        GestureDetector(
          onTap: onPressed,
          child: Container(
            width: 64,
            height: 64,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: bgColor,
            ),
            child: Icon(icon, color: Colors.white, size: 28),
          ),
        ),
        const SizedBox(height: 8),
        Text(
          label,
          style: TextStyle(
            color: Colors.white.withValues(alpha: 0.8),
            fontSize: 12,
          ),
        ),
      ],
    );
  }
}
