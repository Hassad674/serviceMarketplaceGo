import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../core/router/app_router.dart';
import '../../core/theme/app_theme.dart';
import '../../features/auth/presentation/providers/auth_provider.dart';
import '../../l10n/app_localizations.dart';

// Role badge colors — matches web sidebar ROLE_COLORS
const _roleBadgeColors = {
  'agency': (Color(0xFFDBEAFE), Color(0xFF1D4ED8)), // blue-100, blue-700
  'enterprise': (Color(0xFFF3E8FF), Color(0xFF7E22CE)), // purple-100, purple-700
  'provider': (Color(0xFFFFE4E6), Color(0xFFBE123C)), // rose-100, rose-700
};

// Drawer navigation item data
class _DrawerItem {
  const _DrawerItem({
    required this.labelKey,
    required this.icon,
    required this.route,
    this.roles = const ['agency', 'enterprise', 'provider'],
    this.orgTypes,
  });

  final String labelKey;
  final IconData icon;
  final String route;
  final List<String> roles;

  /// Optional additional gate based on `organization.type`. When set,
  /// the item is hidden unless the authenticated user's org type is
  /// in this list. Used to keep the Client-profile entry away from
  /// `provider_personal` operators even though their role is
  /// `provider` (which satisfies the role gate).
  final List<String>? orgTypes;
}

// Primary navigation items
const _primaryItems = [
  _DrawerItem(
    labelKey: 'drawerDashboard',
    icon: Icons.dashboard_outlined,
    route: RoutePaths.dashboard,
  ),
  _DrawerItem(
    labelKey: 'drawerMessages',
    icon: Icons.chat_outlined,
    route: RoutePaths.messaging,
  ),
  _DrawerItem(
    labelKey: 'drawerNotifications',
    icon: Icons.notifications_outlined,
    route: RoutePaths.notifications,
  ),
  _DrawerItem(
    labelKey: 'drawerProjects',
    icon: Icons.folder_open_outlined,
    route: RoutePaths.missions,
  ),
  _DrawerItem(
    labelKey: 'drawerJobs',
    icon: Icons.work_outline,
    route: RoutePaths.jobs,
    roles: ['enterprise', 'agency'],
  ),
  _DrawerItem(
    labelKey: 'drawerOpportunities',
    icon: Icons.work_outline,
    route: RoutePaths.opportunities,
    roles: ['provider', 'agency'],
  ),
  _DrawerItem(
    labelKey: 'drawerMyApplications',
    icon: Icons.description_outlined,
    route: RoutePaths.myApplications,
    roles: ['provider', 'agency'],
  ),
  _DrawerItem(
    labelKey: 'drawerTeam',
    icon: Icons.group_outlined,
    route: RoutePaths.team,
    roles: ['agency', 'enterprise'],
  ),
  _DrawerItem(
    labelKey: 'drawerProfile',
    icon: Icons.person_outline,
    route: RoutePaths.profile,
  ),
  _DrawerItem(
    labelKey: 'navClientProfile',
    icon: Icons.business_center_outlined,
    route: RoutePaths.clientProfile,
    roles: ['agency', 'enterprise'],
    orgTypes: ['agency', 'enterprise'],
  ),
  _DrawerItem(
    labelKey: 'drawerPaymentInfo',
    icon: Icons.credit_card_outlined,
    route: RoutePaths.paymentInfo,
    roles: ['agency', 'provider'],
  ),
  _DrawerItem(
    labelKey: 'drawerWallet',
    icon: Icons.account_balance_wallet_outlined,
    route: RoutePaths.wallet,
    roles: ['agency', 'provider'],
  ),
];

// Search / discovery items
const _searchItems = [
  _DrawerItem(
    labelKey: 'drawerFindFreelancers',
    icon: Icons.person_search,
    route: '/search/freelancer',
    roles: ['agency', 'enterprise'],
  ),
  _DrawerItem(
    labelKey: 'drawerFindAgencies',
    icon: Icons.business_outlined,
    route: '/search/agency',
    roles: ['enterprise'],
  ),
  _DrawerItem(
    labelKey: 'drawerFindReferrers',
    icon: Icons.handshake_outlined,
    route: '/search/referrer',
    roles: ['agency', 'enterprise'],
  ),
];

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
            _DrawerHeader(user: authState.user, role: role),
            Divider(height: 1, color: appColors?.border ?? theme.dividerColor),
            if (showWorkspaceSwitch)
              _WorkspaceSwitch(l10n: l10n),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.symmetric(vertical: 8),
                children: [
                  ..._buildSection(
                    context,
                    items: _primaryItems,
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
                    items: _searchItems,
                    role: role,
                    orgType: orgType,
                    location: location,
                    l10n: l10n,
                  ),
                ],
              ),
            ),
            Divider(height: 1, color: appColors?.border ?? theme.dividerColor),
            _LogoutTile(l10n: l10n),
          ],
        ),
      ),
    );
  }

  List<Widget> _buildSection(
    BuildContext context, {
    required List<_DrawerItem> items,
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
          (item) => _DrawerNavTile(
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
    _DrawerItem item,
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

// Drawer header — avatar + name + role badge

class _DrawerHeader extends StatelessWidget {
  const _DrawerHeader({required this.user, required this.role});

  final Map<String, dynamic>? user;
  final String role;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final displayName =
        user?['display_name'] as String? ??
        user?['first_name'] as String? ??
        'User';
    final initials = _computeInitials(user);
    final badgeColors =
        _roleBadgeColors[role] ??
        (const Color(0xFFF1F5F9), const Color(0xFF64748B));

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
      child: Row(
        children: [
          // Avatar circle with initials or gradient
          Container(
            width: 48,
            height: 48,
            decoration: const BoxDecoration(
              shape: BoxShape.circle,
              gradient: LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [Color(0xFFF43F5E), Color(0xFF8B5CF6)],
              ),
            ),
            alignment: Alignment.center,
            child: Text(
              initials,
              style: const TextStyle(
                color: Colors.white,
                fontSize: 16,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  displayName,
                  style: theme.textTheme.titleMedium,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 4),
                _RoleBadge(
                  role: role,
                  backgroundColor: badgeColors.$1,
                  foregroundColor: badgeColors.$2,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  String _computeInitials(Map<String, dynamic>? user) {
    if (user == null) return '?';
    final firstName = user['first_name'] as String? ?? '';
    final lastName = user['last_name'] as String? ?? '';
    if (firstName.isNotEmpty && lastName.isNotEmpty) {
      return '${firstName[0]}${lastName[0]}'.toUpperCase();
    }
    final displayName = user['display_name'] as String? ?? '';
    if (displayName.length >= 2) {
      return displayName.substring(0, 2).toUpperCase();
    }
    return displayName.isNotEmpty ? displayName[0].toUpperCase() : '?';
  }
}

// Role badge pill

class _RoleBadge extends StatelessWidget {
  const _RoleBadge({
    required this.role,
    required this.backgroundColor,
    required this.foregroundColor,
  });

  final String role;
  final Color backgroundColor;
  final Color foregroundColor;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final label = _roleLabel(l10n, role);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: backgroundColor,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
      ),
      child: Text(
        label,
        style: TextStyle(
          color: foregroundColor,
          fontSize: 11,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.3,
        ),
      ),
    );
  }

  String _roleLabel(AppLocalizations l10n, String role) {
    switch (role) {
      case 'agency':
        return l10n.roleAgency;
      case 'enterprise':
        return l10n.roleEnterprise;
      case 'provider':
        return l10n.roleFreelance;
      default:
        return role;
    }
  }
}

// Navigation tile — single drawer link

class _DrawerNavTile extends StatelessWidget {
  const _DrawerNavTile({
    required this.item,
    required this.isActive,
    required this.l10n,
    this.labelOverride,
  });

  final _DrawerItem item;
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
                    labelOverride ?? _resolveLabel(l10n, item.labelKey),
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

  String _resolveLabel(AppLocalizations l10n, String key) {
    switch (key) {
      case 'drawerDashboard':
        return l10n.drawerDashboard;
      case 'drawerMessages':
        return l10n.drawerMessages;
      case 'drawerProjects':
        return l10n.drawerProjects;
      case 'drawerJobs':
        return l10n.drawerJobs;
      case 'drawerOpportunities':
        return 'Opportunit\u00e9s';
      case 'drawerMyApplications':
        return 'Mes candidatures';
      case 'drawerTeam':
        return l10n.drawerTeam;
      case 'drawerProfile':
        return l10n.drawerProfile;
      case 'navClientProfile':
        return l10n.navClientProfile;
      case 'navProviderProfile':
        return l10n.navProviderProfile;
      case 'drawerFindFreelancers':
        return l10n.drawerFindFreelancers;
      case 'drawerFindAgencies':
        return l10n.drawerFindAgencies;
      case 'drawerFindReferrers':
        return l10n.drawerFindReferrers;
      case 'drawerPaymentInfo':
        return l10n.drawerPaymentInfo;
      case 'drawerWallet':
        return l10n.drawerWallet;
      case 'drawerNotifications':
        return l10n.drawerNotifications;
      default:
        return key;
    }
  }
}

// Workspace switch — Freelance ↔ Referrer (provider only)
const _kWorkspacePref = 'workspace_mode';

class _WorkspaceSwitch extends StatefulWidget {
  const _WorkspaceSwitch({required this.l10n});
  final AppLocalizations l10n;

  @override
  State<_WorkspaceSwitch> createState() => _WorkspaceSwitchState();
}

class _WorkspaceSwitchState extends State<_WorkspaceSwitch> {
  bool _isReferrerMode = false;

  @override
  void initState() {
    super.initState();
    SharedPreferences.getInstance().then((prefs) {
      if (mounted) {
        setState(() {
          _isReferrerMode = prefs.getString(_kWorkspacePref) == 'referrer';
        });
      }
    });
  }

  Future<void> _toggleWorkspace() async {
    final newMode = !_isReferrerMode;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_kWorkspacePref, newMode ? 'referrer' : 'freelance');
    if (!mounted) return;
    setState(() => _isReferrerMode = newMode);
    Navigator.of(context).pop();
    GoRouter.of(context).go(
      newMode ? RoutePaths.dashboardReferrer : RoutePaths.dashboard,
    );
  }

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final isRef = _isReferrerMode;
    final label = isRef
        ? widget.l10n.drawerSwitchToFreelance
        : widget.l10n.drawerSwitchToReferrer;
    final icon = isRef ? Icons.swap_horiz : Icons.auto_awesome;
    final fgColor = isRef
        ? (isDark ? const Color(0xFF6EE7B7) : const Color(0xFF059669))
        : Colors.white;
    final bgDecor = isRef
        ? BoxDecoration(
            color: isDark
                ? const Color(0xFF065F46).withValues(alpha: 0.25)
                : const Color(0xFFECFDF5),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          )
        : BoxDecoration(
            gradient: const LinearGradient(
              colors: [Color(0xFFF43F5E), Color(0xFF8B5CF6)],
            ),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          );

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: GestureDetector(
        onTap: _toggleWorkspace,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: bgDecor,
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(icon, size: 18, color: fgColor),
              const SizedBox(width: 8),
              Text(
                label,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                  color: fgColor,
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
// Logout tile with confirmation dialog
// ---------------------------------------------------------------------------

class _LogoutTile extends ConsumerWidget {
  const _LogoutTile({required this.l10n});

  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);

    return Padding(
      padding: const EdgeInsets.all(8),
      child: Material(
        color: Colors.transparent,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        child: InkWell(
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          onTap: () => _confirmLogout(context, ref),
          child: Padding(
            padding: const EdgeInsets.symmetric(
              horizontal: 12,
              vertical: 10,
            ),
            child: Row(
              children: [
                Icon(
                  Icons.logout_outlined,
                  size: 20,
                  color: theme.colorScheme.error,
                ),
                const SizedBox(width: 12),
                Text(
                  l10n.drawerLogout,
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w500,
                    color: theme.colorScheme.error,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _confirmLogout(BuildContext context, WidgetRef ref) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        ),
        title: Text(l10n.drawerLogout),
        content: Text(l10n.drawerLogoutConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(l10n.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            style: TextButton.styleFrom(
              foregroundColor: Theme.of(ctx).colorScheme.error,
            ),
            child: Text(l10n.drawerLogout),
          ),
        ],
      ),
    );

    if (confirmed == true && context.mounted) {
      Navigator.of(context).pop(); // Close drawer
      await ref.read(authProvider.notifier).logout();
    }
  }
}
