import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/router/app_router.dart';
import '../../core/theme/app_theme.dart';
import '../../features/auth/presentation/providers/auth_provider.dart';
import '../../l10n/app_localizations.dart';
import 'drawer/drawer_header.dart';
import 'drawer/drawer_items.dart';
import 'drawer/drawer_logout_tile.dart';
import 'drawer/drawer_nav_tile.dart';
import 'drawer/drawer_premium_row.dart';
import 'drawer/drawer_workspace_switch.dart';

/// Application drawer with user header, role-based navigation, and logout.
///
/// Mirrors the web sidebar navigation structure. Supports dark mode and
/// adapts visible links based on the authenticated user's role.
class AppDrawer extends ConsumerWidget {
  const AppDrawer({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final role = authState.user?['role'] as String? ?? 'provider';
    final location = GoRouterState.of(context).matchedLocation;

    final isProvider = role == 'provider';
    final referrerEnabled =
        authState.user?['referrer_enabled'] as bool? ?? false;
    final showWorkspaceSwitch = isProvider && referrerEnabled;
    final orgType = authState.organization?['type'] as String?;

    return Drawer(
      backgroundColor: theme.colorScheme.surface,
      child: SafeArea(
        child: Column(
          children: [
            DrawerHeaderTile(user: authState.user, role: role),
            // Premium entry mirrors the web sidebar: visible only for
            // roles that can subscribe (provider + agency — enterprise
            // is a buyer, not a seller). The badge covers all four
            // states (loading, free, past_due, active) — tapping opens
            // the manage bottom-sheet if subscribed, the pricing screen
            // if free.
            if (role == 'provider' || role == 'agency')
              DrawerPremiumRow(role: role),
            Divider(height: 1, color: appColors?.border ?? theme.dividerColor),
            if (showWorkspaceSwitch) DrawerWorkspaceSwitch(l10n: l10n),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.symmetric(vertical: 8),
                children: [
                  ..._buildSection(
                    context,
                    items: drawerPrimaryItems,
                    role: role,
                    orgType: orgType,
                    location: location,
                    l10n: l10n,
                  ),
                  Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 4,
                    ),
                    child: Divider(
                      height: 1,
                      color: appColors?.border ?? theme.dividerColor,
                    ),
                  ),
                  ..._buildSection(
                    context,
                    items: drawerSearchItems,
                    role: role,
                    orgType: orgType,
                    location: location,
                    l10n: l10n,
                  ),
                ],
              ),
            ),
            Divider(height: 1, color: appColors?.border ?? theme.dividerColor),
            DrawerLogoutTile(l10n: l10n),
          ],
        ),
      ),
    );
  }

  List<Widget> _buildSection(
    BuildContext context, {
    required List<DrawerItem> items,
    required String role,
    required String? orgType,
    required String location,
    required AppLocalizations l10n,
  }) {
    return items
        .where((item) => item.roles.contains(role))
        .where(
          (item) =>
              item.orgTypes == null || item.orgTypes!.contains(orgType),
        )
        .map(
          (item) => DrawerNavTile(
            item: item,
            isActive: _isActive(location, item.route),
            l10n: l10n,
            // For agency operators we rename "My profile" to the
            // more specific "Provider profile" so the new
            // "Client profile" entry is not ambiguous.
            labelOverride: _overrideLabelFor(item, orgType, l10n),
          ),
        )
        .toList();
  }

  String? _overrideLabelFor(
    DrawerItem item,
    String? orgType,
    AppLocalizations l10n,
  ) {
    if (item.labelKey == 'drawerProfile' && orgType == 'agency') {
      return l10n.navProviderProfile;
    }
    return null;
  }

  bool _isActive(String location, String route) {
    if (route == RoutePaths.dashboard) return location == route;
    return location.startsWith(route);
  }
}
