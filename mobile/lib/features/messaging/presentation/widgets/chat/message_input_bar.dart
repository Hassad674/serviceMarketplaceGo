import 'dart:async';

import 'package:flutter/material.dart';
import 'package:path_provider/path_provider.dart';
import 'package:record/record.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';

/// Bottom input bar for composing and sending messages.
///
/// WhatsApp-style UX: the right-hand button switches between mic (input empty)
/// and send (input non-empty). During recording the entire bar transforms into
/// a recording strip with cancel / stop controls.
class MessageInputBar extends StatefulWidget {
  const MessageInputBar({
    super.key,
    required this.controller,
    required this.onSend,
    required this.onAttach,
    this.onProposal,
    this.onVoiceRecorded,
    this.replyToName,
    this.replyToContent,
    this.onCancelReply,
  });

  final TextEditingController controller;
  final VoidCallback onSend;
  final VoidCallback onAttach;
  final VoidCallback? onProposal;

  /// Called with the recorded audio file path when voice recording completes.
  final void Function(String path, int durationSeconds)? onVoiceRecorded;

  /// Reply preview data. When non-null, a preview bar is shown above the input.
  final String? replyToName;
  final String? replyToContent;
  final VoidCallback? onCancelReply;

  @override
  State<MessageInputBar> createState() => _MessageInputBarState();
}

class _MessageInputBarState extends State<MessageInputBar>
    with SingleTickerProviderStateMixin {
  static const _primaryColor = Color(0xFFF43F5E);

  final AudioRecorder _recorder = AudioRecorder();
  bool _isRecording = false;
  int _recordingDuration = 0;
  Timer? _timer;

  late final AnimationController _pulseController;
  late final Animation<double> _pulseAnimation;

  @override
  void initState() {
    super.initState();
    _pulseController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1000),
    );
    _pulseAnimation = Tween<double>(begin: 0.6, end: 1.0).animate(
      CurvedAnimation(parent: _pulseController, curve: Curves.easeInOut),
    );
  }

  @override
  void dispose() {
    _timer?.cancel();
    _pulseController.dispose();
    _recorder.dispose();
    super.dispose();
  }

  Future<void> _startRecording() async {
    if (!await _recorder.hasPermission()) return;

    final dir = await getTemporaryDirectory();
    final ts = DateTime.now().millisecondsSinceEpoch;
    final path = '${dir.path}/voice_$ts.m4a';

    await _recorder.start(
      const RecordConfig(encoder: AudioEncoder.aacLc),
      path: path,
    );
    setState(() {
      _isRecording = true;
      _recordingDuration = 0;
    });
    _pulseController.repeat(reverse: true);
    _timer = Timer.periodic(const Duration(seconds: 1), (_) {
      setState(() => _recordingDuration++);
    });
  }

  Future<void> _stopRecording() async {
    _timer?.cancel();
    _timer = null;
    _pulseController.stop();
    final path = await _recorder.stop();
    final duration = _recordingDuration;
    setState(() {
      _isRecording = false;
      _recordingDuration = 0;
    });
    if (path != null && path.isNotEmpty) {
      widget.onVoiceRecorded?.call(path, duration);
    }
  }

  void _cancelRecording() {
    _timer?.cancel();
    _timer = null;
    _pulseController.stop();
    _recorder.stop();
    setState(() {
      _isRecording = false;
      _recordingDuration = 0;
    });
  }

  String _formatDuration(int seconds) {
    final m = seconds ~/ 60;
    final s = (seconds % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        // Reply preview bar
        if (widget.replyToName != null)
          _buildReplyPreview(theme, appColors, l10n),
        Container(
          padding: EdgeInsets.only(
            left: 12,
            right: 12,
            top: 8,
            bottom: MediaQuery.paddingOf(context).bottom + 8,
          ),
          decoration: BoxDecoration(
            color: theme.colorScheme.surface,
            border: Border(
              top: BorderSide(
                color: appColors?.border ?? theme.dividerColor,
                width: 1,
              ),
            ),
          ),
          child: _isRecording
              ? _buildRecordingBar(appColors, l10n)
              : _buildInputBar(theme, appColors, l10n),
        ),
      ],
    );
  }

  Widget _buildReplyPreview(
    ThemeData theme,
    AppColors? appColors,
    AppLocalizations l10n,
  ) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      decoration: BoxDecoration(
        color: appColors?.muted ?? const Color(0xFFF1F5F9),
        border: Border(
          top: BorderSide(
            color: appColors?.border ?? theme.dividerColor,
          ),
        ),
      ),
      child: Row(
        children: [
          Container(width: 2, height: 32, color: _primaryColor),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  l10n.messagingReplyingTo(widget.replyToName!),
                  style: const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: _primaryColor,
                  ),
                ),
                if (widget.replyToContent != null)
                  Text(
                    widget.replyToContent!.length > 50
                        ? '${widget.replyToContent!.substring(0, 50)}...'
                        : widget.replyToContent!,
                    style: TextStyle(
                      fontSize: 12,
                      color: appColors?.mutedForeground,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
              ],
            ),
          ),
          IconButton(
            icon: Icon(
              Icons.close,
              size: 18,
              color: appColors?.mutedForeground,
            ),
            onPressed: widget.onCancelReply,
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(),
          ),
        ],
      ),
    );
  }

  Widget _buildRecordingBar(AppColors? appColors, AppLocalizations l10n) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
      decoration: BoxDecoration(
        color: const Color(0xFFFEE2E2),
        borderRadius: BorderRadius.circular(24),
      ),
      child: Row(
        children: [
          // Cancel button
          GestureDetector(
            onTap: _cancelRecording,
            child: Container(
              width: 40,
              height: 40,
              decoration: BoxDecoration(
                color: Colors.white.withValues(alpha: 0.8),
                shape: BoxShape.circle,
              ),
              child: Icon(
                Icons.delete_outline,
                size: 20,
                color: appColors?.mutedForeground ?? const Color(0xFF64748B),
              ),
            ),
          ),
          const SizedBox(width: 12),
          // Pulsing red dot
          AnimatedBuilder(
            animation: _pulseAnimation,
            builder: (context, child) {
              return Opacity(
                opacity: _pulseAnimation.value,
                child: Container(
                  width: 10,
                  height: 10,
                  decoration: const BoxDecoration(
                    color: Color(0xFFEF4444),
                    shape: BoxShape.circle,
                  ),
                ),
              );
            },
          ),
          const SizedBox(width: 8),
          // Timer
          Text(
            _formatDuration(_recordingDuration),
            style: const TextStyle(
              fontSize: 15,
              fontFamily: 'monospace',
              fontWeight: FontWeight.w600,
              color: Color(0xFFEF4444),
            ),
          ),
          const Spacer(),
          // Stop and send button
          GestureDetector(
            onTap: _stopRecording,
            child: Container(
              width: 40,
              height: 40,
              decoration: const BoxDecoration(
                color: _primaryColor,
                shape: BoxShape.circle,
              ),
              child: const Icon(
                Icons.stop,
                size: 20,
                color: Colors.white,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildInputBar(
    ThemeData theme,
    AppColors? appColors,
    AppLocalizations l10n,
  ) {
    return Row(
      children: [
        // Attachment
        IconButton(
          icon: Icon(
            Icons.attach_file,
            size: 20,
            color: appColors?.mutedForeground,
          ),
          onPressed: widget.onAttach,
        ),

        // Proposal
        if (widget.onProposal != null)
          IconButton(
            icon: Icon(
              Icons.description_outlined,
              size: 20,
              color: appColors?.mutedForeground,
            ),
            tooltip: l10n.proposalPropose,
            onPressed: widget.onProposal,
          ),

        // Text field
        Expanded(
          child: TextField(
            controller: widget.controller,
            textInputAction: TextInputAction.send,
            onSubmitted: (_) => widget.onSend(),
            decoration: InputDecoration(
              hintText: l10n.messagingWriteMessage,
              filled: true,
              fillColor: appColors?.muted ?? const Color(0xFFF1F5F9),
              contentPadding: const EdgeInsets.symmetric(
                horizontal: 16,
                vertical: 10,
              ),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(24),
                borderSide: BorderSide.none,
              ),
              enabledBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(24),
                borderSide: BorderSide.none,
              ),
              focusedBorder: OutlineInputBorder(
                borderRadius: BorderRadius.circular(24),
                borderSide: BorderSide(
                  color: theme.colorScheme.primary
                      .withValues(alpha: 0.3),
                ),
              ),
            ),
          ),
        ),

        const SizedBox(width: 8),

        // Primary action button: mic when empty, send when has text
        _buildPrimaryButton(appColors, l10n),
      ],
    );
  }

  /// The single right-hand button that switches between mic and send.
  Widget _buildPrimaryButton(AppColors? appColors, AppLocalizations l10n) {
    return ListenableBuilder(
      listenable: widget.controller,
      builder: (context, _) {
        final hasText = widget.controller.text.trim().isNotEmpty;
        final bool canVoice = widget.onVoiceRecorded != null;

        // When there is text, show send button
        if (hasText) {
          return GestureDetector(
            onTap: widget.onSend,
            child: AnimatedContainer(
              duration: const Duration(milliseconds: 200),
              width: 40,
              height: 40,
              decoration: const BoxDecoration(
                color: _primaryColor,
                shape: BoxShape.circle,
              ),
              child: const Icon(
                Icons.send,
                size: 18,
                color: Colors.white,
              ),
            ),
          );
        }

        // When input is empty and voice is available, show mic
        if (canVoice) {
          return GestureDetector(
            onTap: _startRecording,
            child: AnimatedContainer(
              duration: const Duration(milliseconds: 200),
              width: 40,
              height: 40,
              decoration: const BoxDecoration(
                color: _primaryColor,
                shape: BoxShape.circle,
              ),
              child: const Icon(
                Icons.mic,
                size: 20,
                color: Colors.white,
              ),
            ),
          );
        }

        // No voice capability and no text: disabled send
        return GestureDetector(
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 200),
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: appColors?.muted ?? const Color(0xFFF1F5F9),
              shape: BoxShape.circle,
            ),
            child: Icon(
              Icons.send,
              size: 18,
              color: appColors?.mutedForeground ?? const Color(0xFF94A3B8),
            ),
          ),
        );
      },
    );
  }
}
