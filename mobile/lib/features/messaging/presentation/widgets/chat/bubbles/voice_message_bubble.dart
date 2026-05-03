import 'package:flutter/material.dart';

import '../../../../../../core/theme/app_theme.dart';
import '../../../../domain/entities/message_entity.dart';
import '../voice_message.dart';
import '../../../../../../core/theme/app_palette.dart';

/// Bubble wrapping the inline voice player. Reads the URL and duration
/// out of the message metadata.
class VoiceMessageBubble extends StatelessWidget {
  const VoiceMessageBubble({
    super.key,
    required this.message,
    required this.isOwn,
  });

  final MessageEntity message;
  final bool isOwn;

  @override
  Widget build(BuildContext context) {
    final appColors = Theme.of(context).extension<AppColors>();
    final url = message.metadata!['url'] as String? ?? '';
    final duration =
        (message.metadata!['duration'] as num?)?.toDouble() ?? 0;

    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Align(
        alignment: isOwn ? Alignment.centerRight : Alignment.centerLeft,
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxWidth: MediaQuery.sizeOf(context).width * 0.65,
            minWidth: 180,
          ),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
            decoration: BoxDecoration(
              color: isOwn
                  ? AppPalette.rose500
                  : (appColors?.muted ?? AppPalette.slate100),
              borderRadius: BorderRadius.only(
                topLeft: const Radius.circular(16),
                topRight: const Radius.circular(16),
                bottomLeft: Radius.circular(isOwn ? 16 : 4),
                bottomRight: Radius.circular(isOwn ? 4 : 16),
              ),
            ),
            child: VoiceMessageWidget(
              url: url,
              duration: duration,
              isOwn: isOwn,
            ),
          ),
        ),
      ),
    );
  }
}
