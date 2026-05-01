import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/app_theme.dart';
import '../../../l10n/app_localizations.dart';
import 'drawer_items.dart';
import 'drawer_label_resolver.dart';

/// Individual drawer link entry — icon + label + active indicator.
///
/// Active state is detected via the matched route from `GoRouter`. Tapping
/// closes the drawer and pushes (search routes) or replaces (primary
/// navigation) the underlying route.
class DrawerNavTile extends StatelessWidget {
  const DrawerNavTile({
    super.key,
    required this.item,
    required this.isActive,
    required this.l10n,
    this.labelOverride,
  });

  final DrawerItem item;
  final bool isActive;
  final AppLocalizations l10n;
  final String? labelOverride;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 1),
      child: Material(
        color: isActive ? primary.withValues(alpha: 0.08) : Colors.transparent,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        child: InkWell(
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          onTap: () {
            Navigator.of(context).pop(); // Close drawer first
            if (item.route.startsWith('/search/')) {
              GoRouter.of(context).push(item.route);
            } else {
              GoRouter.of(context).go(item.route);
            }
          },
          child: Padding(
            padding: const EdgeInsets.symmetric(
              horizontal: 12,
              vertical: 10,
            ),
            child: Row(
              children: [
                Icon(
                  item.icon,
                  size: 20,
                  color: isActive
                      ? primary
                      : theme.colorScheme.onSurface.withValues(alpha: 0.6),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    labelOverride ?? resolveDrawerLabel(l10n, item.labelKey),
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight:
                          isActive ? FontWeight.w600 : FontWeight.w500,
                      color: isActive
                          ? primary
                          : theme.colorScheme.onSurface,
                    ),
                  ),
                ),
                if (isActive)
                  Container(
                    width: 4,
                    height: 20,
                    decoration: BoxDecoration(
                      color: primary,
                      borderRadius: BorderRadius.circular(2),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
