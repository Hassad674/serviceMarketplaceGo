import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../providers/notification_provider.dart';
import '../widgets/notification_tile.dart';

/// Full-screen notification list with pull-to-refresh and swipe-to-delete.
class NotificationScreen extends ConsumerStatefulWidget {
  const NotificationScreen({super.key});

  @override
  ConsumerState<NotificationScreen> createState() =>
      _NotificationScreenState();
}

class _NotificationScreenState extends ConsumerState<NotificationScreen> {
  @override
  void initState() {
    super.initState();
    Future.microtask(
      () => ref.read(notificationListProvider.notifier).load(),
    );
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(notificationListProvider);
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.notifications),
        actions: [
          if (state.notifications.isNotEmpty)
            TextButton(
              onPressed: () {
                ref.read(notificationListProvider.notifier).markAllAsRead();
              },
              child: Text(
                l10n.markAllRead,
                style: TextStyle(
                  color: theme.colorScheme.primary,
                  fontSize: 13,
                ),
              ),
            ),
        ],
      ),
      body: _buildBody(state, theme, l10n),
    );
  }

  Widget _buildBody(
    NotificationListState state,
    ThemeData theme,
    AppLocalizations l10n,
  ) {
    if (state.isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (state.notifications.isEmpty) {
      return _buildEmptyState(theme, l10n);
    }

    return RefreshIndicator(
      onRefresh: () {
        return ref.read(notificationListProvider.notifier).load();
      },
      child: ListView.separated(
        itemCount: state.notifications.length,
        separatorBuilder: (_, __) => Divider(
          height: 1,
          color: theme.dividerColor.withValues(alpha: 0.3),
        ),
        itemBuilder: (context, index) {
          final notification = state.notifications[index];
          return Dismissible(
            key: Key(notification.id),
            direction: DismissDirection.endToStart,
            background: Container(
              alignment: Alignment.centerRight,
              padding: const EdgeInsets.only(right: 20),
              color: Colors.red[400],
              child: const Icon(
                Icons.delete_outline,
                color: Colors.white,
                size: 20,
              ),
            ),
            onDismissed: (_) {
              ref
                  .read(notificationListProvider.notifier)
                  .deleteNotification(notification.id);
            },
            child: NotificationTile(
              notification: notification,
              onTap: () {
                ref
                    .read(notificationListProvider.notifier)
                    .markAsRead(notification.id);
              },
            ),
          );
        },
      ),
    );
  }

  Widget _buildEmptyState(ThemeData theme, AppLocalizations l10n) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.notifications_off_outlined,
            size: 48,
            color: Colors.grey[300],
          ),
          const SizedBox(height: 12),
          Text(
            l10n.noNotifications,
            style: TextStyle(
              color: Colors.grey[500],
              fontSize: 16,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            l10n.noNotificationsDesc,
            style: TextStyle(
              color: Colors.grey[400],
              fontSize: 13,
            ),
          ),
        ],
      ),
    );
  }
}
