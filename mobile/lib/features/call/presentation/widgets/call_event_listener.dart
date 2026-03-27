import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../messaging/data/messaging_ws_service.dart';
import '../../domain/entities/call_entity.dart';
import '../providers/call_provider.dart';
import '../screens/call_screen.dart';
import '../../../../l10n/app_localizations.dart';
import 'incoming_call_overlay.dart';

/// Global listener for WebSocket call events.
///
/// Wraps the entire app so that incoming call events are handled
/// regardless of which screen the user is on. Shows an overlay
/// for incoming calls and navigates to the call screen on accept.
class CallEventListener extends ConsumerStatefulWidget {
  const CallEventListener({super.key, required this.child});

  final Widget child;

  @override
  ConsumerState<CallEventListener> createState() =>
      _CallEventListenerState();
}

class _CallEventListenerState extends ConsumerState<CallEventListener> {
  StreamSubscription<Map<String, dynamic>>? _wsSubscription;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _subscribeToWs();
    });
  }

  @override
  void dispose() {
    _wsSubscription?.cancel();
    super.dispose();
  }

  void _subscribeToWs() {
    final wsService = ref.read(messagingWsServiceProvider);
    _wsSubscription = wsService.events.listen(_onWsEvent);
  }

  void _onWsEvent(Map<String, dynamic> event) {
    final type = event['type'] as String? ?? '';
    if (type == 'call_event') {
      final payload =
          event['payload'] as Map<String, dynamic>? ?? event;
      ref.read(callProvider.notifier).handleCallEvent(payload);
    }
  }

  void _handleAcceptCall() {
    final callState = ref.read(callProvider);
    final callerName = callState.incomingCallerName;
    ref.read(callProvider.notifier).acceptCall();
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => CallScreen(recipientName: callerName),
      ),
    );
  }

  void _handleDeclineCall() {
    ref.read(callProvider.notifier).declineCall();
  }

  @override
  Widget build(BuildContext context) {
    final callState = ref.watch(callProvider);
    final l10n = AppLocalizations.of(context);

    return Stack(
      children: [
        widget.child,
        if (callState.status == CallStatus.ringingIncoming)
          IncomingCallOverlay(
            callerName: callState.incomingCallerName.isNotEmpty
                ? callState.incomingCallerName
                : l10n?.callUnknownCaller ?? 'Unknown',
            onAccept: _handleAcceptCall,
            onDecline: _handleDeclineCall,
          ),
      ],
    );
  }
}
