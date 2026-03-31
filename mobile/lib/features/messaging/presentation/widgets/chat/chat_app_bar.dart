import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../core/utils/extensions.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../domain/entities/conversation_entity.dart';

/// AppBar for the chat screen showing avatar, name, and online/typing status.
class ChatAppBar extends StatelessWidget implements PreferredSizeWidget {
  const ChatAppBar({
    super.key,
    required this.conversation,
    this.typingUserName,
    this.onStartCall,
    this.onStartVideoCall,
    this.onReportUser,
  });

  final ConversationEntity? conversation;
  final String? typingUserName;
  final VoidCallback? onStartCall;
  final VoidCallback? onStartVideoCall;
  final VoidCallback? onReportUser;

  String get _initials =>
      conversation?.otherUserName.initials ?? '?';

  @override
  Size get preferredSize => const Size.fromHeight(kToolbarHeight);

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final online = conversation?.online ?? false;

    String subtitle;
    if (typingUserName != null) {
      subtitle = l10n.messagingTyping(typingUserName!);
    } else if (online) {
      subtitle = l10n.messagingOnline;
    } else {
      subtitle = l10n.messagingOffline;
    }

    return AppBar(
      titleSpacing: 0,
      title: GestureDetector(
        onTap: () {
          final id = conversation?.otherUserId;
          final role = conversation?.otherUserRole;
          if (id == null) return;
          final path = role == 'agency' ? '/profiles/$id' : '/profiles/$id';
          context.push(path);
        },
        child: Row(
        children: [
          // Avatar
          Stack(
            clipBehavior: Clip.none,
            children: [
              Container(
                width: 36,
                height: 36,
                decoration: const BoxDecoration(
                  shape: BoxShape.circle,
                  gradient: LinearGradient(
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                    colors: [
                      Color(0xFFF43F5E),
                      Color(0xFF8B5CF6),
                    ],
                  ),
                ),
                child: Center(
                  child: Text(
                    _initials,
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 13,
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
                    width: 10,
                    height: 10,
                    decoration: BoxDecoration(
                      color: const Color(0xFF22C55E),
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: theme.colorScheme.surface,
                        width: 2,
                      ),
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(width: 12),

          // Name + status
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  conversation?.otherUserName ?? '',
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontSize: 15),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                Text(
                  subtitle,
                  style: TextStyle(
                    fontSize: 12,
                    color: typingUserName != null
                        ? theme.colorScheme.primary
                        : online
                            ? const Color(0xFF22C55E)
                            : theme
                                .extension<AppColors>()
                                ?.mutedForeground,
                  ),
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
                ? const Color(0xFFF43F5E)
                : theme.extension<AppColors>()?.mutedForeground,
          ),
          onPressed: online ? onStartVideoCall : null,
          tooltip: online
              ? l10n.callStartVideoCall
              : l10n.callRecipientOffline,
        ),
        IconButton(
          icon: Icon(
            Icons.phone_outlined,
            size: 20,
            color: online
                ? const Color(0xFF22C55E)
                : theme.extension<AppColors>()?.mutedForeground,
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
                    color: Theme.of(context).colorScheme.error,
                  ),
                  const SizedBox(width: 8),
                  Text(
                    AppLocalizations.of(context)!.reportUser,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
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
