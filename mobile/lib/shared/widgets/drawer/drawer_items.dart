import 'package:flutter/material.dart';

import '../../../core/router/app_router.dart';
import '../../../core/theme/app_palette.dart';

/// Single drawer navigation item descriptor.
///
/// Encodes the label key, icon, route, and access gates (role + optional
/// org type). Pure data — rendered by `DrawerNavTile`.
class DrawerItem {
  const DrawerItem({
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

/// Role badge background+foreground colors — matches web sidebar
/// `ROLE_COLORS`.
const drawerRoleBadgeColors = {
  'agency': (AppPalette.blue100, AppPalette.blue700), // blue-100, blue-700
  'enterprise': (AppPalette.purple100, AppPalette.purple700), // purple-100, purple-700
  'provider': (AppPalette.rose100, AppPalette.rose700), // rose-100, rose-700
};

/// Primary navigation entries (top section of the drawer).
const drawerPrimaryItems = [
  DrawerItem(
    labelKey: 'drawerDashboard',
    icon: Icons.dashboard_outlined,
    route: RoutePaths.dashboard,
  ),
  DrawerItem(
    labelKey: 'drawerMessages',
    icon: Icons.chat_outlined,
    route: RoutePaths.messaging,
  ),
  DrawerItem(
    labelKey: 'drawerNotifications',
    icon: Icons.notifications_outlined,
    route: RoutePaths.notifications,
  ),
  DrawerItem(
    labelKey: 'drawerProjects',
    icon: Icons.folder_open_outlined,
    route: RoutePaths.missions,
  ),
  DrawerItem(
    labelKey: 'drawerJobs',
    icon: Icons.work_outline,
    route: RoutePaths.jobs,
    roles: ['enterprise', 'agency'],
  ),
  DrawerItem(
    labelKey: 'drawerOpportunities',
    icon: Icons.work_outline,
    route: RoutePaths.opportunities,
    roles: ['provider', 'agency'],
  ),
  DrawerItem(
    labelKey: 'drawerMyApplications',
    icon: Icons.description_outlined,
    route: RoutePaths.myApplications,
    roles: ['provider', 'agency'],
  ),
  DrawerItem(
    labelKey: 'drawerTeam',
    icon: Icons.group_outlined,
    route: RoutePaths.team,
    roles: ['agency', 'enterprise'],
  ),
  DrawerItem(
    labelKey: 'drawerProfile',
    icon: Icons.person_outline,
    route: RoutePaths.profile,
  ),
  DrawerItem(
    labelKey: 'navClientProfile',
    icon: Icons.business_center_outlined,
    route: RoutePaths.clientProfile,
    roles: ['agency', 'enterprise'],
    orgTypes: ['agency', 'enterprise'],
  ),
  DrawerItem(
    labelKey: 'drawerPaymentInfo',
    icon: Icons.credit_card_outlined,
    route: RoutePaths.paymentInfo,
    roles: ['agency', 'provider'],
  ),
  DrawerItem(
    labelKey: 'drawerWallet',
    icon: Icons.account_balance_wallet_outlined,
    route: RoutePaths.wallet,
    roles: ['agency', 'provider'],
  ),
  // Invoicing entry — only provider + agency can subscribe / receive
  // commission invoices. Enterprise users are buyers and don't see
  // invoices addressed to them at this stage.
  DrawerItem(
    labelKey: 'drawerInvoices',
    icon: Icons.description_outlined,
    route: RoutePaths.invoices,
    roles: ['agency', 'provider'],
  ),
  // Account preferences — surfaces notifications, email, password and
  // GDPR data + deletion. Available to every role since every role
  // can manage their personal account.
  DrawerItem(
    labelKey: 'drawerMyAccount',
    icon: Icons.manage_accounts_outlined,
    route: RoutePaths.account,
  ),
];

/// Search / discovery entries (bottom section of the drawer).
const drawerSearchItems = [
  DrawerItem(
    labelKey: 'drawerFindFreelancers',
    icon: Icons.person_search,
    route: '/search/freelancer',
    roles: ['agency', 'enterprise'],
  ),
  DrawerItem(
    labelKey: 'drawerFindAgencies',
    icon: Icons.business_outlined,
    route: '/search/agency',
    roles: ['enterprise'],
  ),
  DrawerItem(
    labelKey: 'drawerFindReferrers',
    icon: Icons.handshake_outlined,
    route: '/search/referrer',
    roles: ['agency', 'enterprise'],
  ),
];
