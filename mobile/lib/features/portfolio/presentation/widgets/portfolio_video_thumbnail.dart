import 'package:flutter/material.dart';
import 'package:video_player/video_player.dart';
import '../../../../core/theme/app_palette.dart';

/// Renders the first frame of a video as a thumbnail.
///
/// Uses video_player to load the video metadata + first frame, then displays
/// it as a static image. Shows a dark placeholder while loading and on error.
class PortfolioVideoThumbnail extends StatefulWidget {
  final String videoUrl;
  final BoxFit fit;

  const PortfolioVideoThumbnail({
    super.key,
    required this.videoUrl,
    this.fit = BoxFit.cover,
  });

  @override
  State<PortfolioVideoThumbnail> createState() =>
      _PortfolioVideoThumbnailState();
}

class _PortfolioVideoThumbnailState extends State<PortfolioVideoThumbnail> {
  VideoPlayerController? _controller;
  bool _ready = false;
  bool _failed = false;

  @override
  void initState() {
    super.initState();
    _init();
  }

  Future<void> _init() async {
    try {
      _controller = VideoPlayerController.networkUrl(
        Uri.parse(widget.videoUrl),
      );
      await _controller!.initialize();
      // Seek to a tiny bit in to ensure the first frame is rendered.
      await _controller!.seekTo(const Duration(milliseconds: 100));
      if (mounted) setState(() => _ready = true);
    } catch (_) {
      if (mounted) setState(() => _failed = true);
    }
  }

  @override
  void dispose() {
    _controller?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_failed || _controller == null) {
      return _placeholder();
    }
    if (!_ready) {
      return _placeholder();
    }
    return FittedBox(
      fit: widget.fit,
      clipBehavior: Clip.hardEdge,
      child: SizedBox(
        width: _controller!.value.size.width,
        height: _controller!.value.size.height,
        child: VideoPlayer(_controller!),
      ),
    );
  }

  Widget _placeholder() {
    return Container(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [AppPalette.slate700, AppPalette.slate900],
        ),
      ),
    );
  }
}
