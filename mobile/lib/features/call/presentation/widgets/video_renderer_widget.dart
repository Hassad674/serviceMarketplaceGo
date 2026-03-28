import 'package:flutter/material.dart';
import 'package:livekit_client/livekit_client.dart';

/// Wraps LiveKit's [VideoTrackRenderer] with a fallback for null tracks.
///
/// When [track] is null, renders a dark placeholder container. This is
/// useful for showing a camera-off state or while waiting for a remote
/// participant to publish their video track.
class VideoRendererWidget extends StatelessWidget {
  const VideoRendererWidget({
    super.key,
    required this.track,
    this.mirror = false,
    this.fit = BoxFit.cover,
  });

  final VideoTrack? track;
  final bool mirror;
  final BoxFit fit;

  @override
  Widget build(BuildContext context) {
    if (track == null) {
      return Container(color: const Color(0xFF0F172A));
    }

    return VideoTrackRenderer(
      track!,
      fit: fit == BoxFit.cover
          ? VideoViewFit.cover
          : VideoViewFit.contain,
      mirrorMode: mirror
          ? VideoViewMirrorMode.mirror
          : VideoViewMirrorMode.off,
    );
  }
}
