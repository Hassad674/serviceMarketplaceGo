import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../providers/notification_provider.dart';

/// Bell icon button with an unread notification count badge.
///
/// Watches [unreadNotificationCountProvider] and displays a badge
/// when the count is greater than zero.
class NotificationBadge extends ConsumerWidget {
  final VoidCallback onTap;

  const NotificationBadge({super.key, required this.onTap});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final countAsync = ref.watch(unreadNotificationCountProvider);
    final count = countAsync.valueOrNull ?? 0;

    return IconButton(
      icon: Badge(
        isLabelVisible: count > 0,
        label: Text(
          count > 99 ? '99+' : '$count',
          style: const TextStyle(fontSize: 9, fontWeight: FontWeight.bold),
        ),
        backgroundColor: Theme.of(context).colorScheme.primary,
        child: const Icon(Icons.notifications_outlined, size: 22),
      ),
      onPressed: onTap,
    );
  }
}
