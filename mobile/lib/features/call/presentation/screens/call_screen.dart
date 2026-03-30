import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:livekit_client/livekit_client.dart';

import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/call_entity.dart';
import '../providers/call_provider.dart';
import '../widgets/call_event_listener.dart';
import '../widgets/video_renderer_widget.dart';

/// Full-screen view shown during an active call (audio or video).
///
/// Auto-pops when the call ends (remote hangup, room disconnect, etc.)
/// by listening to [callProvider] state changes.
class CallScreen extends ConsumerStatefulWidget {
  const CallScreen({
    super.key,
    this.recipientName = '',
    this.callType = CallType.audio,
  });

  final String recipientName;
  final CallType callType;

  @override
  ConsumerState<CallScreen> createState() => _CallScreenState();
}

class _CallScreenState extends ConsumerState<CallScreen> {
  bool _controlsVisible = true;
  Timer? _hideControlsTimer;
  double _localVideoX = 16;
  double _localVideoY = 16;
  EventsListener<RoomEvent>? _roomListener;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(callScreenVisibleProvider.notifier).state = true;
    });
    _listenForCallEnd();
    _listenForRoomChanges();
    _resetControlsTimer();
  }

  @override
  void dispose() {
    _roomListener?.dispose();
    _hideControlsTimer?.cancel();
    // Capture the notifier before super.dispose() invalidates ref.
    final notifier = ref.read(callScreenVisibleProvider.notifier);
    notifier.state = false;
    super.dispose();
  }

  void _listenForCallEnd() {
    ref.listenManual(callProvider, (previous, next) {
      if (next.status == CallStatus.idle && mounted) {
        Navigator.of(context).pop();
      }
    });
  }

  /// Watch for room availability and set up track event listeners.
  ///
  /// Handles the case where the room connects AFTER the screen mounts
  /// (both for the initiator and the recipient).
  void _listenForRoomChanges() {
    // Try immediately (room might already be available).
    _setupRoomListener();

    // Also listen for state changes to catch when room becomes available.
    ref.listenManual(callProvider, (previous, next) {
      final room = ref.read(callProvider.notifier).room;
      if (room != null && _roomListener == null) {
        _setupRoomListener();
      }
    });
  }

  /// Attach LiveKit track-level event listeners to the room.
  ///
  /// Safe to call multiple times -- only sets up once (guarded by
  /// [_roomListener] being non-null).
  void _setupRoomListener() {
    final room = ref.read(callProvider.notifier).room;
    if (room == null) {
      debugPrint('[Call] _setupRoomListener: room is null, waiting...');
      return;
    }
    if (_roomListener != null) return;

    debugPrint('[Call] _setupRoomListener: room available, setting up listeners');
    _roomListener = room.createListener();
    _roomListener!
      ..on<TrackSubscribedEvent>((_) {
        debugPrint('[Call] TrackSubscribed event, triggering rebuild');
        _triggerRebuild();
      })
      ..on<TrackUnsubscribedEvent>((_) => _triggerRebuild())
      ..on<TrackMutedEvent>((_) => _triggerRebuild())
      ..on<TrackUnmutedEvent>((_) => _triggerRebuild())
      ..on<ParticipantConnectedEvent>((_) => _triggerRebuild());

    // Trigger an initial rebuild to pick up any tracks already on the room.
    _triggerRebuild();
  }

  void _triggerRebuild() {
    if (mounted) setState(() {});
  }

  void _resetControlsTimer() {
    // Controls always visible — no auto-hide.
    _hideControlsTimer?.cancel();
    if (!_controlsVisible) {
      setState(() => _controlsVisible = true);
    }
  }

  String _formatDuration(int seconds) {
    final m = (seconds ~/ 60).toString().padLeft(2, '0');
    final s = (seconds % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }

  String get _initials {
    final name = widget.recipientName;
    if (name.isEmpty) return '?';
    final parts = name.split(' ');
    return parts
        .map((w) => w.isNotEmpty ? w[0] : '')
        .take(2)
        .join()
        .toUpperCase();
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(callProvider);
    final notifier = ref.read(callProvider.notifier);
    final l10n = AppLocalizations.of(context)!;
    final isVideo = widget.callType == CallType.video;

    return Scaffold(
      backgroundColor: const Color(0xFF0F172A),
      body: GestureDetector(
        onTap: _resetControlsTimer,
        child: Stack(
          fit: StackFit.expand,
          children: [
            _buildBackground(notifier, isVideo, state, l10n),
            if (isVideo) _buildLocalVideoThumbnail(notifier, state),
            _buildTopBar(state, l10n),
            _buildBottomBar(state, notifier, l10n, isVideo),
          ],
        ),
      ),
    );
  }

  /// The main background: remote video for video calls, avatar for audio.
  Widget _buildBackground(
    CallNotifier notifier,
    bool isVideo,
    CallState state,
    AppLocalizations l10n,
  ) {
    final remoteTrack = _getRemoteVideoTrack(notifier);
    debugPrint(
      '[Call] _buildBackground: isVideo=$isVideo, '
      'remoteTrack=${remoteTrack != null}',
    );

    if (isVideo && remoteTrack != null) {
      return VideoRendererWidget(track: remoteTrack);
    }

    return _buildAvatarFallback(state, l10n);
  }

  Widget _buildAvatarFallback(CallState state, AppLocalizations l10n) {
    final isRinging = state.status == CallStatus.ringingOutgoing;

    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          _AvatarCircle(initials: _initials),
          const SizedBox(height: 24),
          Text(
            widget.recipientName.isNotEmpty
                ? widget.recipientName
                : l10n.callAudioCall,
            style: const TextStyle(
              color: Colors.white,
              fontSize: 24,
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            isRinging
                ? l10n.callCalling
                : _formatDuration(state.duration),
            style: TextStyle(
              color: Colors.white.withValues(alpha: 0.7),
              fontSize: 16,
              fontFamily: isRinging ? null : 'monospace',
            ),
          ),
        ],
      ),
    );
  }

  /// Draggable local camera thumbnail shown during video calls.
  Widget _buildLocalVideoThumbnail(
    CallNotifier notifier,
    CallState state,
  ) {
    if (state.isCameraOff) return const SizedBox.shrink();

    final localTrack = _getLocalVideoTrack(notifier);
    if (localTrack == null) return const SizedBox.shrink();

    return Positioned(
      right: _localVideoX,
      top: _localVideoY + MediaQuery.of(context).padding.top,
      child: GestureDetector(
        onPanUpdate: (details) {
          setState(() {
            _localVideoX -= details.delta.dx;
            _localVideoY += details.delta.dy;
          });
        },
        child: ClipRRect(
          borderRadius: BorderRadius.circular(12),
          child: SizedBox(
            width: 120,
            height: 90,
            child: VideoRendererWidget(
              track: localTrack,
              mirror: true,
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildTopBar(CallState state, AppLocalizations l10n) {
    final isRinging = state.status == CallStatus.ringingOutgoing;

    return Positioned(
      top: 0,
      left: 0,
      right: 0,
      child: AnimatedOpacity(
        opacity: _controlsVisible ? 1.0 : 0.0,
        duration: const Duration(milliseconds: 200),
        child: IgnorePointer(
          ignoring: !_controlsVisible,
          child: Container(
            padding: EdgeInsets.only(
              top: MediaQuery.of(context).padding.top + 8,
              left: 16,
              right: 16,
              bottom: 12,
            ),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topCenter,
                end: Alignment.bottomCenter,
                colors: [
                  Colors.black.withValues(alpha: 0.6),
                  Colors.transparent,
                ],
              ),
            ),
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    widget.recipientName,
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 16,
                      fontWeight: FontWeight.w600,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                Text(
                  isRinging
                      ? l10n.callCalling
                      : _formatDuration(state.duration),
                  style: TextStyle(
                    color: Colors.white.withValues(alpha: 0.8),
                    fontSize: 14,
                    fontFamily: isRinging ? null : 'monospace',
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildBottomBar(
    CallState state,
    CallNotifier notifier,
    AppLocalizations l10n,
    bool isVideo,
  ) {
    return Positioned(
      bottom: 0,
      left: 0,
      right: 0,
      child: AnimatedOpacity(
        opacity: _controlsVisible ? 1.0 : 0.0,
        duration: const Duration(milliseconds: 200),
        child: IgnorePointer(
          ignoring: !_controlsVisible,
          child: Container(
            padding: EdgeInsets.only(
              bottom: MediaQuery.of(context).padding.bottom + 32,
              top: 24,
            ),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.bottomCenter,
                end: Alignment.topCenter,
                colors: [
                  Colors.black.withValues(alpha: 0.6),
                  Colors.transparent,
                ],
              ),
            ),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                _CallControlButton(
                  icon: state.isMuted ? Icons.mic_off : Icons.mic,
                  label: state.isMuted ? l10n.callUnmute : l10n.callMute,
                  isActive: state.isMuted,
                  onPressed: notifier.toggleMute,
                ),
                if (isVideo) ...[
                  const SizedBox(width: 32),
                  _CallControlButton(
                    icon: state.isCameraOff
                        ? Icons.videocam_off
                        : Icons.videocam,
                    label: state.isCameraOff
                        ? l10n.callCameraOn
                        : l10n.callCameraOff,
                    isActive: state.isCameraOff,
                    onPressed: notifier.toggleCamera,
                  ),
                ],
                const SizedBox(width: 32),
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
          ),
        ),
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // Track helpers
  // ---------------------------------------------------------------------------

  VideoTrack? _getRemoteVideoTrack(CallNotifier notifier) {
    final room = notifier.room;
    if (room == null) return null;
    final participants = room.remoteParticipants.values;
    if (participants.isEmpty) return null;

    final pubs = participants.first.videoTrackPublications;
    debugPrint(
      '[Call] _getRemoteVideoTrack: '
      'participants=${participants.length}, pubs=${pubs.length}',
    );
    if (pubs.isEmpty) return null;
    return pubs.first.track as VideoTrack?;
  }

  VideoTrack? _getLocalVideoTrack(CallNotifier notifier) {
    final room = notifier.room;
    if (room == null) return null;
    final pubs = room.localParticipant?.videoTrackPublications;
    if (pubs == null || pubs.isEmpty) return null;
    return pubs.first.track as VideoTrack?;
  }
}

// ---------------------------------------------------------------------------
// Sub-widgets
// ---------------------------------------------------------------------------

class _AvatarCircle extends StatelessWidget {
  const _AvatarCircle({required this.initials});

  final String initials;

  @override
  Widget build(BuildContext context) {
    return Container(
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
