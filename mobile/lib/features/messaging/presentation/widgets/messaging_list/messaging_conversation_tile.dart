import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../core/utils/extensions.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../domain/entities/conversation_entity.dart';
import '../../../../../core/theme/app_palette.dart';

const _roleColors = {
  'agency': AppPalette.blue600, // blue-600
  'provider_personal': AppPalette.rose500, // rose-500
  'enterprise': AppPalette.violet500, // purple-500
};

/// Single conversation row — avatar + name + last message + unread
/// count + relative timestamp.
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

  String get _initials => conversation.otherOrgName.initials;

  Color get _roleColor =>
      _roleColors[conversation.otherOrgType] ?? Colors.grey;

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

    return InkWell(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          border: Border(
            left: BorderSide(
              color: conversation.unreadCount > 0
                  ? _roleColor
                  : Colors.transparent,
              width: 3,
            ),
            bottom: BorderSide(
              color: appColors?.border ?? theme.dividerColor,
              width: 0.5,
            ),
          ),
        ),
        child: Row(
          children: [
            _Avatar(initials: _initials, online: conversation.online),
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
                            fontWeight: conversation.unreadCount > 0
                                ? FontWeight.w700
                                : FontWeight.w600,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      Text(
                        _formatTime(),
                        style: theme.textTheme.bodySmall?.copyWith(
                          fontSize: 11,
                          color: appColors?.mutedForeground,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 2),
                  _SubtitleRow(
                    conversation: conversation,
                    isTyping: isTyping,
                    mutedFg: appColors?.mutedForeground,
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
    this.mutedFg,
  });

  final ConversationEntity conversation;
  final bool isTyping;
  final Color? mutedFg;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Row(
      children: [
        Expanded(
          child: isTyping
              ? Text(
                  l10n.messagingTypingShort,
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontStyle: FontStyle.italic,
                    color: AppPalette.rose500,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                )
              : Text(
                  conversation.lastMessage ?? l10n.messagingNoMessages,
                  style: theme.textTheme.bodySmall?.copyWith(color: mutedFg),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
        ),
        if (conversation.unreadCount > 0)
          Container(
            margin: const EdgeInsets.only(left: 8),
            padding:
                const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
            decoration: BoxDecoration(
              color: AppPalette.rose500,
              borderRadius: BorderRadius.circular(10),
            ),
            child: Text(
              '${conversation.unreadCount}',
              style: const TextStyle(
                color: Colors.white,
                fontSize: 10,
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
      ],
    );
  }
}

class _Avatar extends StatelessWidget {
  const _Avatar({required this.initials, required this.online});

  final String initials;
  final bool online;

  @override
  Widget build(BuildContext context) {
    return Stack(
      clipBehavior: Clip.none,
      children: [
        Container(
          width: 44,
          height: 44,
          decoration: const BoxDecoration(
            shape: BoxShape.circle,
            gradient: LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: [
                AppPalette.rose500, // rose-500
                AppPalette.violet500, // purple-600
              ],
            ),
          ),
          child: Center(
            child: Text(
              initials,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 14,
                fontWeight: FontWeight.w600,
              ),
            ),
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
                color: AppPalette.green500, // emerald-500
                shape: BoxShape.circle,
                border: Border.all(
                  color: Theme.of(context).colorScheme.surface,
                  width: 2,
                ),
              ),
            ),
          ),
      ],
    );
  }
}
