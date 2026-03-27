import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:livekit_client/livekit_client.dart';
import 'package:permission_handler/permission_handler.dart';

import '../../../../core/network/api_client.dart';
import '../../data/call_repository_impl.dart';
import '../../domain/entities/call_entity.dart';

/// Provides the [CallRepository] singleton.
final callRepositoryProvider = Provider<CallRepository>((ref) {
  return CallRepository(ref.watch(apiClientProvider));
});

/// State for the global call feature.
class CallState {
  const CallState({
    this.status = CallStatus.idle,
    this.call,
    this.isMuted = false,
    this.duration = 0,
    this.incomingCallerName = '',
  });

  final CallStatus status;
  final CallEntity? call;
  final bool isMuted;
  final int duration;
  final String incomingCallerName;

  CallState copyWith({
    CallStatus? status,
    CallEntity? call,
    bool? isMuted,
    int? duration,
    String? incomingCallerName,
  }) {
    return CallState(
      status: status ?? this.status,
      call: call ?? this.call,
      isMuted: isMuted ?? this.isMuted,
      duration: duration ?? this.duration,
      incomingCallerName: incomingCallerName ?? this.incomingCallerName,
    );
  }
}

/// Manages call lifecycle: initiate, accept, decline, end, mute.
class CallNotifier extends StateNotifier<CallState> {
  CallNotifier(this._repo) : super(const CallState());

  final CallRepository _repo;
  Room? _room;
  Timer? _ringTimer;
  Timer? _durationTimer;

  /// Start an outgoing call.
  Future<void> initiateCall({
    required String conversationId,
    required String recipientId,
  }) async {
    if (state.status != CallStatus.idle) return;

    final micPermission = await Permission.microphone.request();
    if (!micPermission.isGranted) return;

    state = state.copyWith(status: CallStatus.ringingOutgoing);

    try {
      final result = await _repo.initiateCall(
        conversationId: conversationId,
        recipientId: recipientId,
      );

      final call = CallEntity(
        callId: result.callId,
        conversationId: conversationId,
        initiatorId: '',
        recipientId: recipientId,
        callType: CallType.audio,
        roomName: result.roomName,
        token: result.token,
      );
      state = state.copyWith(call: call);

      await _connectToRoom(result.token);
      _startRingTimeout();
    } catch (_) {
      _cleanup();
    }
  }

  /// Accept an incoming call.
  Future<void> acceptCall() async {
    final call = state.call;
    if (call == null || state.status != CallStatus.ringingIncoming) return;

    final micPermission = await Permission.microphone.request();
    if (!micPermission.isGranted) return;

    try {
      final result = await _repo.acceptCall(call.callId);
      state = state.copyWith(
        status: CallStatus.active,
        call: call.copyWith(
          token: result.token,
          roomName: result.roomName,
          startedAt: DateTime.now(),
        ),
      );
      _cancelRingTimer();
      _startDurationTimer();
      await _connectToRoom(result.token);
    } catch (_) {
      _cleanup();
    }
  }

  /// Decline an incoming call.
  Future<void> declineCall() async {
    final call = state.call;
    if (call == null) {
      _cleanup();
      return;
    }
    try {
      await _repo.declineCall(call.callId);
    } catch (_) {
      // Ignore
    }
    _cleanup();
  }

  /// End an active call.
  Future<void> endCall() async {
    final call = state.call;
    if (call == null) {
      _cleanup();
      return;
    }
    try {
      await _repo.endCall(call.callId, state.duration);
    } catch (_) {
      // Ignore
    }
    _cleanup();
  }

  /// Toggle microphone mute.
  void toggleMute() {
    if (_room == null) return;
    final newMuted = !state.isMuted;
    _room!.localParticipant?.setMicrophoneEnabled(!newMuted);
    state = state.copyWith(isMuted: newMuted);
  }

  /// Handle an incoming call event from WebSocket.
  void handleCallEvent(Map<String, dynamic> payload) {
    final event = payload['event'] as String? ?? '';

    switch (event) {
      case 'call_incoming':
        if (state.status != CallStatus.idle) return;
        state = state.copyWith(
          status: CallStatus.ringingIncoming,
          call: CallEntity(
            callId: payload['call_id'] as String? ?? '',
            conversationId: payload['conversation_id'] as String? ?? '',
            initiatorId: payload['initiator_id'] as String? ?? '',
            recipientId: payload['recipient_id'] as String? ?? '',
            callType: CallType.audio,
          ),
        );
        _startRingTimeout();

      case 'call_accepted':
        _cancelRingTimer();
        state = state.copyWith(
          status: CallStatus.active,
          call: state.call?.copyWith(startedAt: DateTime.now()),
        );
        _startDurationTimer();

      case 'call_declined':
      case 'call_ended':
        _cleanup();
    }
  }

  Future<void> _connectToRoom(String token) async {
    _room = Room();

    _room!.addListener(_onRoomDisconnected);

    const lkUrl = String.fromEnvironment(
      'LIVEKIT_URL',
      defaultValue: '',
    );
    if (lkUrl.isEmpty) return;

    await _room!.connect(lkUrl, token);
    await _room!.localParticipant?.setMicrophoneEnabled(true);
  }

  void _onRoomDisconnected() {
    _cleanup();
  }

  void _startRingTimeout() {
    _ringTimer?.cancel();
    _ringTimer = Timer(const Duration(seconds: 30), () {
      if (state.status == CallStatus.ringingIncoming) {
        declineCall();
      } else if (state.status == CallStatus.ringingOutgoing) {
        endCall();
      }
    });
  }

  void _cancelRingTimer() {
    _ringTimer?.cancel();
    _ringTimer = null;
  }

  void _startDurationTimer() {
    _durationTimer?.cancel();
    final start = DateTime.now();
    _durationTimer = Timer.periodic(const Duration(seconds: 1), (_) {
      final secs = DateTime.now().difference(start).inSeconds;
      state = state.copyWith(duration: secs);
    });
  }

  void _cleanup() {
    _ringTimer?.cancel();
    _ringTimer = null;
    _durationTimer?.cancel();
    _durationTimer = null;
    _room?.removeListener(_onRoomDisconnected);
    _room?.disconnect();
    _room = null;
    state = const CallState();
  }

  @override
  void dispose() {
    _cleanup();
    super.dispose();
  }
}

/// Global call state provider.
final callProvider = StateNotifierProvider<CallNotifier, CallState>((ref) {
  return CallNotifier(ref.watch(callRepositoryProvider));
});
