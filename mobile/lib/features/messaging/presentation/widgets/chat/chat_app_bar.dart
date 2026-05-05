import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../domain/entities/conversation_entity.dart';

// M-17 chat AppBar — Soleil v2.
//
// Layout: back arrow + Soleil avatar (with online dot) + name +
// online/typing status + phone / video / more buttons.

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

/// AppBar for the chat screen showing avatar, name, and online/typing status.
class ChatAppBar extends StatelessWidget implements PreferredSizeWidget {
  const ChatAppBar({
    super.key,
    required this.conversation,
    this.currentOrgType,
    this.typingUserName,
    this.onStartCall,
    this.onStartVideoCall,
    this.onReportUser,
  });

  final ConversationEntity? conversation;
  final String? currentOrgType;
  final String? typingUserName;
  final VoidCallback? onStartCall;
  final VoidCallback? onStartVideoCall;
  final VoidCallback? onReportUser;

  @override
  Size get preferredSize => const Size.fromHeight(kToolbarHeight);

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final online = conversation?.online ?? false;

    String subtitle;
    Color subtitleColor;
    if (typingUserName != null) {
      subtitle = l10n.messagingTyping(typingUserName!);
      subtitleColor = theme.colorScheme.primary;
    } else if (online) {
      subtitle = l10n.messagingOnline;
      subtitleColor = appColors?.success ?? theme.colorScheme.primary;
    } else {
      subtitle = l10n.messagingOffline;
      subtitleColor =
          appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;
    }

    final otherOrgType = conversation?.otherOrgType;
    final isClientCounterparty = otherOrgType == 'enterprise';
    final canViewProfile = conversation?.otherOrgId != null &&
        conversation!.otherOrgId.isNotEmpty;

    return AppBar(
      backgroundColor: theme.colorScheme.surface,
      elevation: 0,
      scrolledUnderElevation: 0,
      titleSpacing: 0,
      title: GestureDetector(
        onTap: canViewProfile
            ? () {
                final id = conversation?.otherOrgId;
                if (id == null || id.isEmpty) return;
                if (isClientCounterparty) {
                  context.push('/clients/$id');
                } else {
                  context.push('/profiles/$id');
                }
              }
            : null,
        child: Row(
          children: [
            _SoleilHeaderAvatar(
              seed: conversation?.id.isNotEmpty == true
                  ? conversation!.id
                  : (conversation?.otherOrgName ?? '?'),
              online: online,
              successColor:
                  appColors?.success ?? const Color(0xFF5A9670),
              borderColor: theme.colorScheme.surface,
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    conversation?.otherOrgName ?? '',
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontSize: 15,
                      fontWeight: FontWeight.w600,
                      color: theme.colorScheme.onSurface,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  Text(
                    subtitle,
                    style: theme.textTheme.bodySmall?.copyWith(
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                      color: subtitleColor,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
      actions: [
        IconButton(
          icon: Icon(
            Icons.videocam_outlined,
            size: 20,
            color: online
                ? theme.colorScheme.primary
                : appColors?.mutedForeground,
          ),
          onPressed: online ? onStartVideoCall : null,
          tooltip:
              online ? l10n.callStartVideoCall : l10n.callRecipientOffline,
        ),
        IconButton(
          icon: Icon(
            Icons.phone_outlined,
            size: 20,
            color: online
                ? (appColors?.success ?? theme.colorScheme.primary)
                : appColors?.mutedForeground,
          ),
          onPressed: online ? onStartCall : null,
          tooltip: online ? l10n.callStartCall : l10n.callRecipientOffline,
        ),
        PopupMenuButton<String>(
          icon: const Icon(Icons.more_vert, size: 20),
          onSelected: (value) {
            if (value == 'report_user') {
              onReportUser?.call();
            }
          },
          itemBuilder: (context) => [
            PopupMenuItem<String>(
              value: 'report_user',
              child: Row(
                children: [
                  Icon(
                    Icons.flag_outlined,
                    size: 18,
                    color: theme.colorScheme.error,
                  ),
                  const SizedBox(width: 8),
                  Text(
                    l10n.reportUser,
                    style: TextStyle(
                      color: theme.colorScheme.error,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ],
    );
  }
}

class _SoleilHeaderAvatar extends StatelessWidget {
  const _SoleilHeaderAvatar({
    required this.seed,
    required this.online,
    required this.successColor,
    required this.borderColor,
  });

  final String seed;
  final bool online;
  final Color successColor;
  final Color borderColor;

  @override
  Widget build(BuildContext context) {
    final palette = _portraitPalettes[_portraitId(seed)];
    final bg = palette[0];
    final skin = palette[1];
    final shirt = palette[2];

    return Stack(
      clipBehavior: Clip.none,
      children: [
        Container(
          width: 36,
          height: 36,
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
              width: 10,
              height: 10,
              decoration: BoxDecoration(
                color: successColor,
                shape: BoxShape.circle,
                border: Border.all(color: borderColor, width: 2),
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

    final shirtPath = Path()
      ..moveTo(8 * scale, 60 * scale)
      ..quadraticBezierTo(8 * scale, 46 * scale, 30 * scale, 44 * scale)
      ..quadraticBezierTo(52 * scale, 46 * scale, 52 * scale, 60 * scale)
      ..close();
    canvas.drawPath(shirtPath, shirtPaint);

    canvas.drawRect(
      Rect.fromLTWH(24 * scale, 38 * scale, 12 * scale, 10 * scale),
      skinPaint,
    );

    canvas.drawOval(
      Rect.fromCenter(
        center: Offset(30 * scale, 28 * scale),
        width: 22 * scale,
        height: 26 * scale,
      ),
      skinPaint,
    );

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
