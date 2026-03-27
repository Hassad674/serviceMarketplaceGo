import 'dart:async';

import 'package:flutter/material.dart';
import 'package:path_provider/path_provider.dart';
import 'package:record/record.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';

/// Bottom input bar for composing and sending messages.
///
/// Supports text, file attachments, proposals, and voice recording.
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

class _MessageInputBarState extends State<MessageInputBar> {
  final AudioRecorder _recorder = AudioRecorder();
  bool _isRecording = false;
  int _recordingDuration = 0;
  Timer? _timer;

  @override
  void dispose() {
    _timer?.cancel();
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
    _timer = Timer.periodic(const Duration(seconds: 1), (_) {
      setState(() => _recordingDuration++);
    });
  }

  Future<void> _stopRecording() async {
    _timer?.cancel();
    _timer = null;
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
          Container(
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
                Container(
                  width: 2,
                  height: 32,
                  color: const Color(0xFFF43F5E),
                ),
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
                          color: Color(0xFFF43F5E),
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
          ),
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

  Widget _buildRecordingBar(AppColors? appColors, AppLocalizations l10n) {
    return Row(
      children: [
        // Cancel
        IconButton(
          icon: const Icon(Icons.close, size: 20),
          color: appColors?.mutedForeground,
          tooltip: l10n.messagingCancelRecording,
          onPressed: _cancelRecording,
        ),
        // Red pulse + timer
        Container(
          width: 10,
          height: 10,
          decoration: const BoxDecoration(
            color: Color(0xFFEF4444),
            shape: BoxShape.circle,
          ),
        ),
        const SizedBox(width: 8),
        Text(
          _formatDuration(_recordingDuration),
          style: const TextStyle(
            fontSize: 14,
            fontFamily: 'monospace',
            fontWeight: FontWeight.w500,
            color: Color(0xFFEF4444),
          ),
        ),
        const SizedBox(width: 8),
        Text(
          l10n.messagingRecording,
          style: TextStyle(
            fontSize: 13,
            color: appColors?.mutedForeground,
          ),
        ),
        const Spacer(),
        // Stop and send
        GestureDetector(
          onTap: _stopRecording,
          child: Container(
            width: 40,
            height: 40,
            decoration: const BoxDecoration(
              color: Color(0xFFF43F5E),
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

        // Voice
        if (widget.onVoiceRecorded != null)
          IconButton(
            icon: Icon(
              Icons.mic_none,
              size: 20,
              color: appColors?.mutedForeground,
            ),
            tooltip: l10n.messagingVoiceMessage,
            onPressed: _startRecording,
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

        // Send button
        ListenableBuilder(
          listenable: widget.controller,
          builder: (context, _) {
            final hasText = widget.controller.text.trim().isNotEmpty;

            return GestureDetector(
              onTap: hasText ? widget.onSend : null,
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 200),
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: hasText
                      ? const Color(0xFFF43F5E)
                      : (appColors?.muted ?? const Color(0xFFF1F5F9)),
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.send,
                  size: 18,
                  color: hasText
                      ? Colors.white
                      : (appColors?.mutedForeground ??
                          const Color(0xFF94A3B8)),
                ),
              ),
            );
          },
        ),
      ],
    );
  }
}
