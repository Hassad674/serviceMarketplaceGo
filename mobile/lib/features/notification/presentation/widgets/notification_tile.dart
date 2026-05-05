import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/app_notification.dart';

/// M-19 — Soleil v2 notification row.
///
/// Mirrors the JSX `NotifRow` in
/// `design/assets/sources/phase1/soleil-app-lot4.jsx` (lines 123-153):
/// 36×36 rounded-square icon chip tinted with one of three accents
/// (corail / sapin / mute), Inter Tight title (700 if unread, 600 if
/// read), tabac body, mono pill on the right with the relative time, a
/// trailing 7px corail dot for unread items. Background is ivoire-soft
/// (`#fffaf3`) for unread to mirror the JSX's `n.unread ? '#fffaf3'`.
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
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    final isUnread = !notification.isRead;

    // Ivoire-soft background for unread (mirrors JSX `#fffaf3` — a 1-stop
    // warmer than `--background`). Resolved through the ColorScheme's
    // surface anchor so the dark theme keeps a calibrated warm tint.
    final unreadBg = Color.alphaBlend(
      colorScheme.primary.withValues(alpha: 0.04),
      colorScheme.surface,
    );

    return Material(
      color: isUnread ? unreadBg : colorScheme.surfaceContainerLowest,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _IconChip(type: notification.type),
              const SizedBox(width: 11),
              Expanded(
                child: _TileBody(
                  notification: notification,
                  isUnread: isUnread,
                  colorScheme: colorScheme,
                  colors: colors,
                  l10n: l10n,
                ),
              ),
              if (isUnread) ...[
                const SizedBox(width: 8),
                Container(
                  width: 7,
                  height: 7,
                  margin: const EdgeInsets.only(top: 6),
                  decoration: BoxDecoration(
                    color: colorScheme.primary,
                    shape: BoxShape.circle,
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

/// 36×36 rounded-square icon chip — corail / sapin / mute tint based on
/// the notification type.
class _IconChip extends StatelessWidget {
  final String type;

  const _IconChip({required this.type});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final spec = _resolveSpec(type, colorScheme, colors);

    return Container(
      width: 36,
      height: 36,
      decoration: BoxDecoration(
        color: spec.background,
        borderRadius: BorderRadius.circular(11),
      ),
      child: Icon(spec.icon, size: 15, color: spec.foreground),
    );
  }

  _ChipSpec _resolveSpec(
    String type,
    ColorScheme scheme,
    AppColors colors,
  ) {
    switch (type) {
      case 'proposal_received':
        return _ChipSpec(
          icon: Icons.work_outline_rounded,
          background: colors.successSoft,
          foreground: colors.success,
        );
      case 'proposal_accepted':
      case 'proposal_completed':
      case 'completion_requested':
        return _ChipSpec(
          icon: Icons.check_circle_outline_rounded,
          background: colors.successSoft,
          foreground: colors.success,
        );
      case 'proposal_paid':
        return _ChipSpec(
          icon: Icons.account_balance_wallet_outlined,
          background: colors.successSoft,
          foreground: colors.success,
        );
      case 'proposal_declined':
        return _ChipSpec(
          icon: Icons.cancel_outlined,
          background: colors.accentSoft,
          foreground: scheme.primary,
        );
      case 'proposal_modified':
        return _ChipSpec(
          icon: Icons.refresh_rounded,
          background: colors.accentSoft,
          foreground: scheme.primary,
        );
      case 'review_received':
        return _ChipSpec(
          icon: Icons.star_outline_rounded,
          background: colors.accentSoft,
          foreground: scheme.primary,
        );
      case 'new_message':
        return _ChipSpec(
          icon: Icons.chat_bubble_outline_rounded,
          background: colors.accentSoft,
          foreground: scheme.primary,
        );
      default:
        return _ChipSpec(
          icon: Icons.auto_awesome_outlined,
          background: scheme.surface,
          foreground: scheme.onSurfaceVariant,
        );
    }
  }
}

class _ChipSpec {
  final IconData icon;
  final Color background;
  final Color foreground;

  _ChipSpec({
    required this.icon,
    required this.background,
    required this.foreground,
  });
}

/// Title / body / mono time pill stack for a notification row.
class _TileBody extends StatelessWidget {
  final AppNotification notification;
  final bool isUnread;
  final ColorScheme colorScheme;
  final AppColors colors;
  final AppLocalizations l10n;

  const _TileBody({
    required this.notification,
    required this.isUnread,
    required this.colorScheme,
    required this.colors,
    required this.l10n,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(
              child: Text(
                notification.title,
                style: SoleilTextStyles.body.copyWith(
                  fontSize: 13,
                  height: 1.3,
                  color: colorScheme.onSurface,
                  fontWeight:
                      isUnread ? FontWeight.w700 : FontWeight.w600,
                ),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
            ),
            const SizedBox(width: 8),
            Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text(
                _formatTime(notification.createdAt, l10n),
                style: SoleilTextStyles.mono.copyWith(
                  fontSize: 10.5,
                  color: colorScheme.onSurfaceVariant,
                  letterSpacing: 0.4,
                ),
              ),
            ),
          ],
        ),
        if (notification.body.isNotEmpty) ...[
          const SizedBox(height: 2),
          Text(
            notification.body,
            style: SoleilTextStyles.body.copyWith(
              fontSize: 11.5,
              height: 1.4,
              color: colorScheme.onSurfaceVariant,
            ),
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
          ),
        ],
      ],
    );
  }

  String _formatTime(DateTime date, AppLocalizations l10n) {
    final diff = DateTime.now().difference(date);
    if (diff.inSeconds < 60) return l10n.notificationsTimeJustNow;
    if (diff.inMinutes < 60) return l10n.notificationsTimeMinutes(diff.inMinutes);
    if (diff.inHours < 24) return l10n.notificationsTimeHours(diff.inHours);
    return l10n.notificationsTimeDays(diff.inDays);
  }
}
