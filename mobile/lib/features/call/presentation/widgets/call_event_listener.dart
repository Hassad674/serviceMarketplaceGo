import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../messaging/data/messaging_ws_service.dart';
import '../../domain/entities/call_entity.dart';
import '../providers/call_provider.dart';
import '../screens/call_screen.dart';
import 'incoming_call_overlay.dart';

/// Global listener for WebSocket call events.
///
/// Wraps the entire app (via MaterialApp.builder) so incoming call events
/// are handled regardless of which screen the user is on.
///
/// Ensures the WebSocket connection is established as soon as the user
/// is authenticated, so call events are received even before the user
/// visits the messaging screen.
class CallEventListener extends ConsumerStatefulWidget {
  const CallEventListener({super.key, required this.child});

  final Widget child;

  @override
  ConsumerState<CallEventListener> createState() =>
      _CallEventListenerState();
}

class _CallEventListenerState extends ConsumerState<CallEventListener> {
  StreamSubscription<Map<String, dynamic>>? _wsSubscription;
  bool _wsConnected = false;

  @override
  void dispose() {
    _wsSubscription?.cancel();
    super.dispose();
  }

  /// Ensures the WebSocket is connected and the event stream is subscribed.
  ///
  /// Called reactively from [build] whenever the auth state is authenticated.
  /// Safe to call multiple times -- guards against duplicate subscriptions.
  void _ensureWsConnected() {
    if (_wsConnected) return;
    _wsConnected = true;

    final wsService = ref.read(messagingWsServiceProvider);

    // Subscribe to the broadcast stream (listens even before connect).
    _wsSubscription?.cancel();
    _wsSubscription = wsService.events.listen(_onWsEvent);

    // Kick off the WS connection if not already connected.
    if (!wsService.isConnected) {
      wsService.connect();
    }
  }

  /// Tears down the subscription when the user logs out.
  void _disconnectWs() {
    _wsSubscription?.cancel();
    _wsSubscription = null;
    _wsConnected = false;
  }

  void _onWsEvent(Map<String, dynamic> event) {
    final type = event['type'] as String? ?? '';
    if (type != 'call_event') return;

    final payload =
        event['payload'] as Map<String, dynamic>? ?? event;
    debugPrint('[Call] WS call_event received: ${payload['event']}');
    ref.read(callProvider.notifier).handleCallEvent(payload);
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
    final authState = ref.watch(authProvider);
    final callState = ref.watch(callProvider);
    final l10n = AppLocalizations.of(context);

    // Ensure WS is connected whenever the user is authenticated.
    if (authState.status == AuthStatus.authenticated) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _ensureWsConnected();
      });
    } else if (authState.status == AuthStatus.unauthenticated) {
      // Clean up on logout.
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _disconnectWs();
      });
    }

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
