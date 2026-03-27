import 'package:audioplayers/audioplayers.dart';
import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';

/// Renders a voice message player inside a chat bubble.
///
/// Displays play/pause button, progress slider, and formatted duration.
/// Uses [AudioPlayer] from the `audioplayers` package for playback.
class VoiceMessageWidget extends StatefulWidget {
  const VoiceMessageWidget({
    super.key,
    required this.url,
    required this.duration,
    required this.isOwn,
  });

  final String url;
  final double duration;
  final bool isOwn;

  @override
  State<VoiceMessageWidget> createState() => _VoiceMessageWidgetState();
}

class _VoiceMessageWidgetState extends State<VoiceMessageWidget> {
  final AudioPlayer _player = AudioPlayer();
  bool _isPlaying = false;
  Duration _position = Duration.zero;
  Duration _totalDuration = Duration.zero;

  @override
  void initState() {
    super.initState();
    _totalDuration = Duration(
      milliseconds: (widget.duration * 1000).round(),
    );
    _player.onPositionChanged.listen((pos) {
      if (mounted) setState(() => _position = pos);
    });
    _player.onDurationChanged.listen((dur) {
      if (mounted && dur.inMilliseconds > 0) {
        setState(() => _totalDuration = dur);
      }
    });
    _player.onPlayerComplete.listen((_) {
      if (mounted) {
        setState(() {
          _isPlaying = false;
          _position = Duration.zero;
        });
      }
    });
  }

  @override
  void dispose() {
    _player.dispose();
    super.dispose();
  }

  Future<void> _togglePlay() async {
    if (_isPlaying) {
      await _player.pause();
      setState(() => _isPlaying = false);
    } else {
      await _player.play(UrlSource(widget.url));
      setState(() => _isPlaying = true);
    }
  }

  String _formatDuration(Duration d) {
    final m = d.inMinutes;
    final s = (d.inSeconds % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }

  @override
  Widget build(BuildContext context) {
    final appColors = Theme.of(context).extension<AppColors>();
    final progress = _totalDuration.inMilliseconds > 0
        ? _position.inMilliseconds / _totalDuration.inMilliseconds
        : 0.0;

    final displayDuration = _isPlaying || _position.inMilliseconds > 0
        ? _formatDuration(_position)
        : _formatDuration(_totalDuration);

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        // Play / Pause
        GestureDetector(
          onTap: _togglePlay,
          child: Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: widget.isOwn
                  ? Colors.white.withValues(alpha: 0.2)
                  : const Color(0xFFFCE4EC),
              shape: BoxShape.circle,
            ),
            child: Icon(
              _isPlaying ? Icons.pause : Icons.play_arrow,
              size: 20,
              color: widget.isOwn
                  ? Colors.white
                  : const Color(0xFFF43F5E),
            ),
          ),
        ),

        const SizedBox(width: 10),

        // Progress + duration
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              // Progress bar
              ClipRRect(
                borderRadius: BorderRadius.circular(2),
                child: LinearProgressIndicator(
                  value: progress.clamp(0.0, 1.0),
                  minHeight: 4,
                  backgroundColor: widget.isOwn
                      ? Colors.white.withValues(alpha: 0.2)
                      : (appColors?.border ?? const Color(0xFFE2E8F0)),
                  valueColor: AlwaysStoppedAnimation<Color>(
                    widget.isOwn
                        ? Colors.white
                        : const Color(0xFFF43F5E),
                  ),
                ),
              ),
              const SizedBox(height: 4),
              Row(
                children: [
                  Icon(
                    Icons.mic,
                    size: 12,
                    color: widget.isOwn
                        ? Colors.white.withValues(alpha: 0.6)
                        : (appColors?.mutedForeground ??
                            const Color(0xFF94A3B8)),
                  ),
                  const SizedBox(width: 4),
                  Text(
                    displayDuration,
                    style: TextStyle(
                      fontSize: 10,
                      fontFamily: 'monospace',
                      color: widget.isOwn
                          ? Colors.white.withValues(alpha: 0.7)
                          : (appColors?.mutedForeground ??
                              const Color(0xFF94A3B8)),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ],
    );
  }
}
