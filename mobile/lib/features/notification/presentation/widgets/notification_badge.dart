import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../providers/notification_provider.dart';

/// Bell icon button with an unread notification count badge — Soleil v2 trim.
///
/// Watches [unreadNotificationCountProvider] and renders a corail badge
/// when the count is greater than zero. Uses Geist Mono digits via
/// `SoleilTextStyles.mono` so the count typography matches the rest of
/// the Soleil numeric style (transactions, time pills, IDs).
class NotificationBadge extends ConsumerWidget {
  final VoidCallback onTap;

  const NotificationBadge({super.key, required this.onTap});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final countAsync = ref.watch(unreadNotificationCountProvider);
    final count = countAsync.valueOrNull ?? 0;

    return IconButton(
      icon: Badge(
        isLabelVisible: count > 0,
        label: Text(
          count > 99 ? '99+' : '$count',
          style: SoleilTextStyles.mono.copyWith(
            fontSize: 9,
            fontWeight: FontWeight.w700,
            color: colorScheme.onPrimary,
          ),
        ),
        backgroundColor: colorScheme.primary,
        child: Icon(
          Icons.notifications_none_rounded,
          size: 22,
          color: colorScheme.onSurfaceVariant,
        ),
      ),
      onPressed: onTap,
    );
  }
}
