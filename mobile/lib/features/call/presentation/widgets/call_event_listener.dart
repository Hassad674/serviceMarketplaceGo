import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/router/app_router.dart' show rootNavigatorKey;
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../messaging/data/messaging_ws_service.dart';
import '../../domain/entities/call_entity.dart';
import '../providers/call_provider.dart';
import '../screens/call_screen.dart';
import 'active_call_mini_bar.dart';
import 'incoming_call_overlay.dart';

/// Tracks whether the [CallScreen] is currently visible. Set to true when
/// CallScreen is pushed and false when it is popped. Used to decide whether
/// to show the [ActiveCallMiniBar].
final callScreenVisibleProvider = StateProvider<bool>((_) => false);

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

  Future<void> _handleAcceptCall() async {
    final callState = ref.read(callProvider);
    final callerName = callState.incomingCallerName;
    final callType = callState.call?.callType ?? CallType.audio;

    // Accept the call first (API + LiveKit room connection) before navigating.
    await ref.read(callProvider.notifier).acceptCall();
    if (!mounted) return;

    // Use the global navigator key because CallEventListener sits ABOVE the
    // GoRouter navigator in the widget tree (mounted via MaterialApp.builder).
    rootNavigatorKey.currentState?.push(
      MaterialPageRoute(
        builder: (_) => CallScreen(
          recipientName: callerName,
          callType: callType,
        ),
      ),
    );
  }

  void _handleDeclineCall() {
    ref.read(callProvider.notifier).declineCall();
  }

  void _navigateToCallScreen() {
    final callState = ref.read(callProvider);
    final callerName = callState.incomingCallerName;
    final callType = callState.call?.callType ?? CallType.audio;

    rootNavigatorKey.currentState?.push(
      MaterialPageRoute(
        builder: (_) => CallScreen(
          recipientName: callerName,
          callType: callType,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authProvider);
    final callState = ref.watch(callProvider);
    final callScreenVisible = ref.watch(callScreenVisibleProvider);
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

    final showMiniBar = _shouldShowMiniBar(callState, callScreenVisible);

    return Stack(
      children: [
        widget.child,
        if (callState.status == CallStatus.ringingIncoming)
          IncomingCallOverlay(
            callerName: callState.incomingCallerName.isNotEmpty
                ? callState.incomingCallerName
                : l10n?.callUnknownCaller ?? 'Unknown',
            callType: callState.call?.callType ?? CallType.audio,
            onAccept: _handleAcceptCall,
            onDecline: _handleDeclineCall,
          ),
        if (showMiniBar)
          Positioned(
            top: 0,
            left: 0,
            right: 0,
            child: ActiveCallMiniBar(
              participantName: callState.incomingCallerName,
              durationSeconds: callState.duration,
              onTap: _navigateToCallScreen,
            ),
          ),
      ],
    );
  }

  bool _shouldShowMiniBar(CallState callState, bool callScreenVisible) {
    if (callScreenVisible) return false;
    final isActive = callState.status == CallStatus.active ||
        callState.status == CallStatus.ringingOutgoing;
    return isActive;
  }
}
