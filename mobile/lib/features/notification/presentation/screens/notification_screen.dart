import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/app_notification.dart';
import '../providers/notification_provider.dart';
import '../widgets/notification_tile.dart';

/// M-19 — Notifications screen, Soleil v2 visual port.
///
/// Adapts `AppNotifications` from
/// `design/assets/sources/phase1/soleil-app-lot4.jsx` (lines 72-121):
///
///   - Editorial Fraunces title with italic corail accent ("Notifications
///     récentes") + Fraunces italic subtitle ("5 non lues · tout marquer
///     lu") that doubles as the mark-all-read affordance when there are
///     unread items, and is a calm informational sentence otherwise.
///   - Notifications grouped chronologically (Aujourd'hui / Hier / Cette
///     semaine / Plus ancien) under mono uppercase eyebrows.
///   - Each group rendered in a rounded ivoire card (radius 14, sable
///     border) with 1px sable dividers between rows.
///   - Pull-to-refresh + swipe-to-delete preserved (the underlying
///     repository API is untouched).
///   - Empty state: corail-soft circular icon chip + Fraunces title + a
///     calm Fraunces italic subtitle in tabac.
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
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;
    final state = ref.watch(notificationListProvider);
    final unreadCount = state.notifications.where((n) => !n.isRead).length;

    return Scaffold(
      backgroundColor: colorScheme.surface,
      appBar: AppBar(
        backgroundColor: colorScheme.surface,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
        leading: const _BackButton(),
        title: const SizedBox.shrink(),
        toolbarHeight: 56,
      ),
      body: SafeArea(
        top: false,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _Header(
              unreadCount: unreadCount,
              hasNotifications: state.notifications.isNotEmpty,
              isLoading: state.isLoading,
              onMarkAllRead: () =>
                  ref.read(notificationListProvider.notifier).markAllAsRead(),
            ),
            const SizedBox(height: 14),
            Expanded(child: _Body(state: state, l10n: l10n)),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Top-bar back button — calm 36px ivoire round button (Soleil v2 signature).
// ---------------------------------------------------------------------------

class _BackButton extends StatelessWidget {
  const _BackButton();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;

    return Padding(
      padding: const EdgeInsets.only(left: 12, top: 8, bottom: 8),
      child: Material(
        color: colorScheme.surfaceContainerLowest,
        shape: CircleBorder(
          side: BorderSide(color: colors.border),
        ),
        child: InkWell(
          customBorder: const CircleBorder(),
          onTap: () => Navigator.of(context).maybePop(),
          child: SizedBox(
            width: 36,
            height: 36,
            child: Icon(
              Icons.arrow_back_rounded,
              size: 18,
              color: colorScheme.onSurface,
            ),
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Header — Fraunces title + italic subtitle (mark-all-read affordance).
// ---------------------------------------------------------------------------

class _Header extends StatelessWidget {
  final int unreadCount;
  final bool hasNotifications;
  final bool isLoading;
  final VoidCallback onMarkAllRead;

  const _Header({
    required this.unreadCount,
    required this.hasNotifications,
    required this.isLoading,
    required this.onMarkAllRead,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;

    final subtitleText = !hasNotifications
        ? l10n.noNotificationsDesc
        : (unreadCount == 0
            ? l10n.notificationsSubtitleAllRead
            : l10n.notificationsSubtitleUnread(unreadCount));

    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 6, 20, 0),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Editorial Fraunces title with italic corail accent.
          Text.rich(
            TextSpan(
              children: [
                TextSpan(
                  text: '${l10n.notifications} ',
                  style: SoleilTextStyles.headlineLarge.copyWith(
                    fontSize: 26,
                    fontWeight: FontWeight.w600,
                    letterSpacing: -0.5,
                    color: colorScheme.onSurface,
                  ),
                ),
                TextSpan(
                  text: l10n.notificationsTitleAccent,
                  style: SoleilTextStyles.headlineLarge.copyWith(
                    fontSize: 26,
                    fontWeight: FontWeight.w500,
                    letterSpacing: -0.5,
                    fontStyle: FontStyle.italic,
                    color: colorScheme.primary,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 2),
          if (hasNotifications && unreadCount > 0 && !isLoading)
            InkWell(
              onTap: onMarkAllRead,
              borderRadius: BorderRadius.circular(6),
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 4),
                child: Text(
                  subtitleText,
                  style: SoleilTextStyles.body.copyWith(
                    fontSize: 12.5,
                    fontStyle: FontStyle.italic,
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
              ),
            )
          else
            Text(
              subtitleText,
              style: SoleilTextStyles.body.copyWith(
                fontSize: 12.5,
                fontStyle: FontStyle.italic,
                color: colorScheme.onSurfaceVariant,
              ),
            ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Body — loading / empty / grouped list.
// ---------------------------------------------------------------------------

class _Body extends ConsumerWidget {
  final NotificationListState state;
  final AppLocalizations l10n;

  const _Body({required this.state, required this.l10n});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (state.isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (state.notifications.isEmpty) {
      return _EmptyState(l10n: l10n);
    }

    final groups = _groupByRelativeDay(state.notifications, l10n);

    return RefreshIndicator(
      onRefresh: () =>
          ref.read(notificationListProvider.notifier).load(),
      child: ListView.builder(
        padding: const EdgeInsets.fromLTRB(20, 0, 20, 24),
        physics: const AlwaysScrollableScrollPhysics(),
        itemCount: groups.length,
        itemBuilder: (context, index) {
          return _NotificationGroup(
            group: groups[index],
            isLast: index == groups.length - 1,
            onTap: (notification) => ref
                .read(notificationListProvider.notifier)
                .markAsRead(notification.id),
            onDelete: (notification) => ref
                .read(notificationListProvider.notifier)
                .deleteNotification(notification.id),
          );
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Group eyebrow + rounded ivoire card hosting a list of notification rows.
// ---------------------------------------------------------------------------

class _NotificationGroup extends StatelessWidget {
  final _Group group;
  final bool isLast;
  final void Function(AppNotification) onTap;
  final void Function(AppNotification) onDelete;

  const _NotificationGroup({
    required this.group,
    required this.isLast,
    required this.onTap,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;

    return Padding(
      padding: EdgeInsets.only(bottom: isLast ? 0 : 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(4, 6, 4, 10),
            child: Text(
              group.label,
              style: SoleilTextStyles.mono.copyWith(
                fontSize: 11,
                fontWeight: FontWeight.w700,
                letterSpacing: 0.7,
                color: colors.subtleForeground,
              ),
            ),
          ),
          ClipRRect(
            borderRadius: BorderRadius.circular(14),
            child: Container(
              decoration: BoxDecoration(
                color: colorScheme.surfaceContainerLowest,
                border: Border.all(color: colors.border),
                borderRadius: BorderRadius.circular(14),
              ),
              child: Column(
                children: [
                  for (var i = 0; i < group.items.length; i++) ...[
                    Dismissible(
                      key: Key(group.items[i].id),
                      direction: DismissDirection.endToStart,
                      background: Container(
                        color: colorScheme.error,
                        alignment: Alignment.centerRight,
                        padding: const EdgeInsets.only(right: 20),
                        child: Icon(
                          Icons.delete_outline_rounded,
                          color: colorScheme.onError,
                          size: 20,
                        ),
                      ),
                      onDismissed: (_) => onDelete(group.items[i]),
                      child: NotificationTile(
                        notification: group.items[i],
                        onTap: () => onTap(group.items[i]),
                      ),
                    ),
                    if (i < group.items.length - 1)
                      Divider(
                        height: 1,
                        thickness: 1,
                        color: colors.border,
                      ),
                  ],
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Empty state — calm corail-soft icon chip + Fraunces copy.
// ---------------------------------------------------------------------------

class _EmptyState extends StatelessWidget {
  final AppLocalizations l10n;

  const _EmptyState({required this.l10n});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;

    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32),
        child: Container(
          padding: const EdgeInsets.fromLTRB(24, 36, 24, 36),
          decoration: BoxDecoration(
            color: colorScheme.surfaceContainerLowest,
            border: Border.all(color: colors.border),
            borderRadius: BorderRadius.circular(20),
            boxShadow: AppTheme.cardShadow,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 56,
                height: 56,
                decoration: BoxDecoration(
                  color: colors.accentSoft,
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.notifications_none_rounded,
                  size: 26,
                  color: colorScheme.primary,
                ),
              ),
              const SizedBox(height: 14),
              Text(
                l10n.noNotifications,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.titleLarge.copyWith(
                  fontSize: 20,
                  letterSpacing: -0.2,
                  color: colorScheme.onSurface,
                ),
              ),
              const SizedBox(height: 6),
              Text(
                l10n.noNotificationsDesc,
                textAlign: TextAlign.center,
                style: SoleilTextStyles.body.copyWith(
                  fontSize: 13.5,
                  fontStyle: FontStyle.italic,
                  height: 1.5,
                  color: colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Grouping — pure data transformation, no Flutter dependency.
// ---------------------------------------------------------------------------

class _Group {
  final String label;
  final List<AppNotification> items;

  _Group({required this.label, required this.items});
}

List<_Group> _groupByRelativeDay(
  List<AppNotification> items,
  AppLocalizations l10n,
) {
  final now = DateTime.now();
  final startOfToday = DateTime(now.year, now.month, now.day);
  final startOfYesterday =
      startOfToday.subtract(const Duration(days: 1));
  final startOfThisWeek = startOfToday.subtract(const Duration(days: 6));

  final today = <AppNotification>[];
  final yesterday = <AppNotification>[];
  final thisWeek = <AppNotification>[];
  final earlier = <AppNotification>[];

  for (final n in items) {
    if (!n.createdAt.isBefore(startOfToday)) {
      today.add(n);
    } else if (!n.createdAt.isBefore(startOfYesterday)) {
      yesterday.add(n);
    } else if (!n.createdAt.isBefore(startOfThisWeek)) {
      thisWeek.add(n);
    } else {
      earlier.add(n);
    }
  }

  final groups = <_Group>[];
  if (today.isNotEmpty) {
    groups.add(_Group(label: l10n.notificationsGroupToday, items: today));
  }
  if (yesterday.isNotEmpty) {
    groups.add(
      _Group(label: l10n.notificationsGroupYesterday, items: yesterday),
    );
  }
  if (thisWeek.isNotEmpty) {
    groups.add(
      _Group(label: l10n.notificationsGroupThisWeek, items: thisWeek),
    );
  }
  if (earlier.isNotEmpty) {
    groups.add(
      _Group(label: l10n.notificationsGroupEarlier, items: earlier),
    );
  }
  return groups;
}
