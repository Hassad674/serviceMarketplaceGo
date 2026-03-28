import 'package:flutter/material.dart';

import '../../domain/entities/app_notification.dart';

/// A single notification row in the notification list.
///
/// Shows an icon based on notification type, title, body, time ago,
/// and an unread indicator dot.
class NotificationTile extends StatelessWidget {
  final AppNotification notification;
  final VoidCallback? onTap;

  const NotificationTile({
    super.key,
    required this.notification,
    this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isUnread = !notification.isRead;

    return InkWell(
      onTap: onTap,
      child: Container(
        color: isUnread
            ? theme.colorScheme.primary.withValues(alpha: 0.04)
            : null,
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildIcon(theme),
            const SizedBox(width: 12),
            Expanded(child: _buildContent(theme, isUnread)),
            if (isUnread) ...[
              const SizedBox(width: 8),
              _buildUnreadDot(theme),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildIcon(ThemeData theme) {
    final (icon, bgColor, iconColor) = _iconForType(notification.type);
    return Container(
      width: 36,
      height: 36,
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: BorderRadius.circular(10),
      ),
      child: Icon(icon, size: 18, color: iconColor),
    );
  }

  Widget _buildContent(ThemeData theme, bool isUnread) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          notification.title,
          style: TextStyle(
            fontSize: 14,
            fontWeight: isUnread ? FontWeight.w600 : FontWeight.w400,
            color: theme.colorScheme.onSurface,
          ),
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        if (notification.body.isNotEmpty) ...[
          const SizedBox(height: 2),
          Text(
            notification.body,
            style: TextStyle(fontSize: 12, color: Colors.grey[500]),
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
          ),
        ],
        const SizedBox(height: 4),
        Text(
          _timeAgo(notification.createdAt),
          style: TextStyle(fontSize: 10, color: Colors.grey[400]),
        ),
      ],
    );
  }

  Widget _buildUnreadDot(ThemeData theme) {
    return Container(
      width: 8,
      height: 8,
      margin: const EdgeInsets.only(top: 4),
      decoration: BoxDecoration(
        color: theme.colorScheme.primary,
        shape: BoxShape.circle,
      ),
    );
  }

  (IconData, Color, Color) _iconForType(String type) {
    switch (type) {
      case 'proposal_received':
        return (
          Icons.description_outlined,
          Colors.blue[50]!,
          Colors.blue[600]!,
        );
      case 'proposal_accepted':
      case 'proposal_completed':
        return (
          Icons.check_circle_outline,
          Colors.green[50]!,
          Colors.green[600]!,
        );
      case 'proposal_declined':
        return (
          Icons.cancel_outlined,
          Colors.red[50]!,
          Colors.red[600]!,
        );
      case 'proposal_modified':
        return (
          Icons.refresh_outlined,
          Colors.amber[50]!,
          Colors.amber[700]!,
        );
      case 'proposal_paid':
        return (
          Icons.credit_card_outlined,
          Colors.teal[50]!,
          Colors.teal[600]!,
        );
      case 'completion_requested':
        return (
          Icons.check_circle_outline,
          Colors.purple[50]!,
          Colors.purple[600]!,
        );
      case 'review_received':
        return (
          Icons.star_outline,
          Colors.amber[50]!,
          Colors.amber[600]!,
        );
      case 'new_message':
        return (
          Icons.chat_outlined,
          Colors.lightBlue[50]!,
          Colors.lightBlue[600]!,
        );
      default:
        return (
          Icons.notifications_outlined,
          Colors.grey[100]!,
          Colors.grey[600]!,
        );
    }
  }

  String _timeAgo(DateTime date) {
    final diff = DateTime.now().difference(date);
    if (diff.inSeconds < 60) return 'just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m';
    if (diff.inHours < 24) return '${diff.inHours}h';
    return '${diff.inDays}d';
  }
}
