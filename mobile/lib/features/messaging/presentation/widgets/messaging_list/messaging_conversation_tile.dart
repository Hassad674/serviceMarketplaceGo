import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../core/utils/extensions.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../domain/entities/conversation_entity.dart';

// M-18 Liste conversations — Soleil v2.
//
// Compact row: Portrait-style avatar + name + last message excerpt +
// relative timestamp + corail unread badge. Tap pushes M-17.

const _portraitPalettes = <List<Color>>[
  [Color(0xFFFDE9E3), Color(0xFFE8A890), Color(0xFFC43A26)],
  [Color(0xFFE8F2EB), Color(0xFFD4A584), Color(0xFF5A9670)],
  [Color(0xFFFDE6ED), Color(0xFFD49A82), Color(0xFFC84D72)],
  [Color(0xFFFBF0DC), Color(0xFFC4926E), Color(0xFFD4924A)],
  [Color(0xFFE8E4F4), Color(0xFFD8A890), Color(0xFF6B5B9A)],
  [Color(0xFFDFECEF), Color(0xFFC89478), Color(0xFF3A6B7A)],
];

int _portraitId(String seed) {
  var h = 0;
  for (final c in seed.codeUnits) {
    h = (h * 31 + c) & 0x7fffffff;
  }
  return h % _portraitPalettes.length;
}

/// Single conversation row — Soleil avatar + name + last message
/// preview + relative timestamp + unread badge.
class MessagingConversationTile extends StatelessWidget {
  const MessagingConversationTile({
    super.key,
    required this.conversation,
    required this.onTap,
    this.isTyping = false,
  });

  final ConversationEntity conversation;
  final VoidCallback onTap;
  final bool isTyping;

  String _formatTime() {
    final raw = conversation.lastMessageAt;
    if (raw == null || raw.isEmpty) return '';
    try {
      final dt = DateTime.parse(raw);
      return dt.toRelative();
    } catch (_) {
      return raw;
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final hasUnread = conversation.unreadCount > 0;

    return InkWell(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          border: Border(
            bottom: BorderSide(
              color: appColors?.border ?? theme.dividerColor,
              width: 0.5,
            ),
          ),
        ),
        child: Row(
          children: [
            _SoleilAvatar(
              seed: conversation.id.isNotEmpty
                  ? conversation.id
                  : conversation.otherOrgName,
              online: conversation.online,
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          conversation.otherOrgName,
                          style: theme.textTheme.titleMedium?.copyWith(
                            fontSize: 14,
                            fontWeight: hasUnread
                                ? FontWeight.w700
                                : FontWeight.w600,
                            color: theme.colorScheme.onSurface,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      Text(
                        _formatTime(),
                        style: theme.textTheme.bodySmall?.copyWith(
                          fontSize: 11,
                          fontFamily: 'monospace',
                          color: appColors?.mutedForeground,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 2),
                  _SubtitleRow(
                    conversation: conversation,
                    isTyping: isTyping,
                    appColors: appColors,
                    primary: theme.colorScheme.primary,
                    onPrimary: theme.colorScheme.onPrimary,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _SubtitleRow extends StatelessWidget {
  const _SubtitleRow({
    required this.conversation,
    required this.isTyping,
    required this.primary,
    required this.onPrimary,
    this.appColors,
  });

  final ConversationEntity conversation;
  final bool isTyping;
  final AppColors? appColors;
  final Color primary;
  final Color onPrimary;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final hasUnread = conversation.unreadCount > 0;

    return Row(
      children: [
        Expanded(
          child: isTyping
              ? Text(
                  l10n.messagingTypingShort,
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontStyle: FontStyle.italic,
                    color: primary,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                )
              : Text(
                  conversation.lastMessage ?? l10n.messagingNoMessages,
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight:
                        hasUnread ? FontWeight.w500 : FontWeight.w400,
                    color: hasUnread
                        ? theme.colorScheme.onSurface
                        : appColors?.mutedForeground,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
        ),
        if (hasUnread)
          Container(
            margin: const EdgeInsets.only(left: 8),
            padding:
                const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
            decoration: BoxDecoration(
              color: primary,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Text(
              '${conversation.unreadCount}',
              style: TextStyle(
                color: onPrimary,
                fontSize: 10,
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
      ],
    );
  }
}

class _SoleilAvatar extends StatelessWidget {
  const _SoleilAvatar({required this.seed, required this.online});

  final String seed;
  final bool online;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final palette = _portraitPalettes[_portraitId(seed)];
    final bg = palette[0];
    final skin = palette[1];
    final shirt = palette[2];

    return Stack(
      clipBehavior: Clip.none,
      children: [
        Container(
          width: 44,
          height: 44,
          decoration: BoxDecoration(
            color: bg,
            shape: BoxShape.circle,
          ),
          clipBehavior: Clip.antiAlias,
          child: CustomPaint(
            painter: _PortraitPainter(skin: skin, shirt: shirt),
          ),
        ),
        if (online)
          Positioned(
            bottom: 0,
            right: 0,
            child: Container(
              width: 12,
              height: 12,
              decoration: BoxDecoration(
                color: theme.extension<AppColors>()?.success ??
                    const Color(0xFF5A9670),
                shape: BoxShape.circle,
                border: Border.all(
                  color: theme.colorScheme.surface,
                  width: 2,
                ),
              ),
            ),
          ),
      ],
    );
  }
}

class _PortraitPainter extends CustomPainter {
  _PortraitPainter({required this.skin, required this.shirt});

  final Color skin;
  final Color shirt;

  @override
  void paint(Canvas canvas, Size size) {
    final scale = size.width / 60;
    final shirtPaint = Paint()..color = shirt;
    final skinPaint = Paint()..color = skin;
    final hairPaint = Paint()..color = const Color(0xFF3D2618);

    // Shoulders
    final shirtPath = Path()
      ..moveTo(8 * scale, 60 * scale)
      ..quadraticBezierTo(8 * scale, 46 * scale, 30 * scale, 44 * scale)
      ..quadraticBezierTo(52 * scale, 46 * scale, 52 * scale, 60 * scale)
      ..close();
    canvas.drawPath(shirtPath, shirtPaint);

    // Neck
    canvas.drawRect(
      Rect.fromLTWH(24 * scale, 38 * scale, 12 * scale, 10 * scale),
      skinPaint,
    );

    // Head
    canvas.drawOval(
      Rect.fromCenter(
        center: Offset(30 * scale, 28 * scale),
        width: 22 * scale,
        height: 26 * scale,
      ),
      skinPaint,
    );

    // Hair
    final hairPath = Path()
      ..moveTo(19 * scale, 24 * scale)
      ..quadraticBezierTo(19 * scale, 13 * scale, 30 * scale, 13 * scale)
      ..quadraticBezierTo(41 * scale, 13 * scale, 41 * scale, 24 * scale)
      ..quadraticBezierTo(41 * scale, 21 * scale, 36 * scale, 19 * scale)
      ..quadraticBezierTo(30 * scale, 17 * scale, 24 * scale, 19 * scale)
      ..quadraticBezierTo(19 * scale, 21 * scale, 19 * scale, 28 * scale)
      ..close();
    canvas.drawPath(hairPath, hairPaint);
  }

  @override
  bool shouldRepaint(covariant _PortraitPainter old) =>
      old.skin != skin || old.shirt != shirt;
}
