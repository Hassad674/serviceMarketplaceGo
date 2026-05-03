import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../notification/presentation/widgets/notification_badge.dart';
import '../widgets/dashboard_atoms.dart';
import '../../../../core/theme/app_palette.dart';

/// Main dashboard / home screen with role-based stats cards.
///
/// Dispatches to the agency / enterprise / provider variants based on
/// the signed-in user's role. Default falls back to the provider
/// dashboard so unknown roles still render something.
class DashboardScreen extends ConsumerWidget {
  const DashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final role = authState.user?['role'] as String?;

    switch (role) {
      case 'agency':
        return const _AgencyDashboard();
      case 'enterprise':
        return const _EnterpriseDashboard();
      case 'provider':
        return const _ProviderDashboard();
      default:
        return const _ProviderDashboard();
    }
  }
}

class _AgencyDashboard extends ConsumerWidget {
  const _AgencyDashboard();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final l10n = AppLocalizations.of(context)!;
    final displayName =
        authState.user?['display_name'] as String? ?? 'Agency';

    return Scaffold(
      appBar: AppBar(
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: const Text('Marketplace'),
        actions: [
          NotificationBadge(
            onTap: () =>
                GoRouter.of(context).go(RoutePaths.notifications),
          ),
        ],
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              DashboardWelcomeBanner(
                displayName: displayName,
                subtitle: l10n.roleAgencyDesc,
              ),
              const SizedBox(height: 24),
              DashboardSearchActions(
                actions: [
                  DashboardSearchAction(
                    label: l10n.findFreelancers,
                    icon: Icons.person_search,
                    type: 'freelancer',
                    color: AppPalette.rose500,
                  ),
                  DashboardSearchAction(
                    label: l10n.findReferrers,
                    icon: Icons.handshake_outlined,
                    type: 'referrer',
                    color: AppPalette.amber500,
                  ),
                ],
              ),
              const SizedBox(height: 24),
              _agencyStats(l10n),
            ],
          ),
        ),
      ),
    );
  }

  Widget _agencyStats(AppLocalizations l10n) {
    return Column(
      children: [
        DashboardStatCard(
          icon: Icons.work_outline,
          title: l10n.activeMissions,
          value: '0',
          subtitle: l10n.activeContracts,
          color: AppPalette.blue600,
        ),
        const SizedBox(height: 12),
        DashboardStatCard(
          icon: Icons.chat_outlined,
          title: l10n.unreadMessages,
          value: '0',
          subtitle: l10n.conversations,
          color: AppPalette.violet500,
        ),
        const SizedBox(height: 12),
        DashboardStatCard(
          icon: Icons.trending_up,
          title: l10n.monthlyRevenue,
          value: '0 EUR',
          subtitle: l10n.thisMonth,
          color: AppPalette.green500,
        ),
      ],
    );
  }
}

class _EnterpriseDashboard extends ConsumerWidget {
  const _EnterpriseDashboard();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final l10n = AppLocalizations.of(context)!;
    final displayName =
        authState.user?['display_name'] as String? ?? 'Enterprise';

    return Scaffold(
      appBar: AppBar(
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: const Text('Marketplace'),
        actions: [
          NotificationBadge(
            onTap: () =>
                GoRouter.of(context).go(RoutePaths.notifications),
          ),
        ],
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              DashboardWelcomeBanner(
                displayName: displayName,
                subtitle: l10n.roleEnterpriseDesc,
              ),
              const SizedBox(height: 24),
              DashboardSearchActions(
                actions: [
                  DashboardSearchAction(
                    label: l10n.findFreelancers,
                    icon: Icons.person_search,
                    type: 'freelancer',
                    color: AppPalette.rose500,
                  ),
                  DashboardSearchAction(
                    label: l10n.findAgencies,
                    icon: Icons.business,
                    type: 'agency',
                    color: AppPalette.blue600,
                  ),
                  DashboardSearchAction(
                    label: l10n.findReferrers,
                    icon: Icons.handshake_outlined,
                    type: 'referrer',
                    color: AppPalette.amber500,
                  ),
                ],
              ),
              const SizedBox(height: 24),
              _enterpriseStats(l10n),
            ],
          ),
        ),
      ),
    );
  }

  Widget _enterpriseStats(AppLocalizations l10n) {
    return Column(
      children: [
        DashboardStatCard(
          icon: Icons.folder_open_outlined,
          title: l10n.activeProjects,
          value: '0',
          subtitle: l10n.activeProjects,
          color: AppPalette.blue600,
        ),
        const SizedBox(height: 12),
        DashboardStatCard(
          icon: Icons.chat_outlined,
          title: l10n.unreadMessages,
          value: '0',
          subtitle: l10n.conversations,
          color: AppPalette.violet500,
        ),
        const SizedBox(height: 12),
        DashboardStatCard(
          icon: Icons.account_balance_wallet_outlined,
          title: l10n.totalBudget,
          value: '0 EUR',
          subtitle: l10n.spentThisMonth,
          color: AppPalette.green500,
        ),
      ],
    );
  }
}

class _ProviderDashboard extends ConsumerWidget {
  const _ProviderDashboard();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final l10n = AppLocalizations.of(context)!;
    final displayName = authState.user?['first_name'] as String? ??
        authState.user?['display_name'] as String? ??
        'Provider';

    return Scaffold(
      appBar: AppBar(
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: const Text('Marketplace'),
        actions: [
          NotificationBadge(
            onTap: () =>
                GoRouter.of(context).go(RoutePaths.notifications),
          ),
        ],
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              DashboardWelcomeBanner(
                displayName: displayName,
                subtitle: l10n.roleFreelanceDesc,
              ),
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  onPressed: () => context.go(RoutePaths.dashboardReferrer),
                  icon: const Icon(Icons.swap_horiz),
                  label: Text(l10n.businessReferrerMode),
                ),
              ),
              const SizedBox(height: 24),
              _providerStats(l10n),
            ],
          ),
        ),
      ),
    );
  }

  Widget _providerStats(AppLocalizations l10n) {
    return Column(
      children: [
        DashboardStatCard(
          icon: Icons.work_outline,
          title: l10n.activeMissions,
          value: '0',
          subtitle: l10n.activeContracts,
          color: AppPalette.blue600,
        ),
        const SizedBox(height: 12),
        DashboardStatCard(
          icon: Icons.chat_outlined,
          title: l10n.unreadMessages,
          value: '0',
          subtitle: l10n.conversations,
          color: AppPalette.violet500,
        ),
        const SizedBox(height: 12),
        DashboardStatCard(
          icon: Icons.trending_up,
          title: l10n.monthlyRevenue,
          value: '0 EUR',
          subtitle: l10n.thisMonth,
          color: AppPalette.green500,
        ),
      ],
    );
  }
}
