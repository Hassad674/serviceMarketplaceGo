import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../notification/presentation/widgets/notification_badge.dart';
import '../widgets/dashboard_atoms.dart';

/// Main dashboard / home screen — Soleil v2.
///
/// Role-aware dispatch (provider / agency / enterprise) over a single
/// editorial layout: corail mono eyebrow, Fraunces italic-corail welcome
/// title, role-specific subtitle, then a Soleil stat grid (3 cards) and
/// (provider only) a referrer-mode switch pill.
///
/// Out-of-scope flagged (NOT implemented this batch — features absent from
/// repo, mirroring the W-11 web flag set):
///   - "Cette semaine chez Atelier" editorial card (no blog/content feature)
///   - "Atelier Premium" sidebar bottom CTA (subscription tier UX shipped
///     elsewhere)
///   - "Tes missions du moment" 3-mission progress list (dashboard data
///     layer currently exposes only stat counts, no mission feed)
///   - "Conversations en cours" 3-thread list (would require a new hook)
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

// ---------------------------------------------------------------------------
// Shared layout (Soleil)
// ---------------------------------------------------------------------------

class _DashboardScaffold extends StatelessWidget {
  const _DashboardScaffold({
    required this.greeting,
    this.searchActions,
    this.referrerSwitch,
    required this.statCards,
  });

  final Widget greeting;
  final Widget? searchActions;
  final Widget? referrerSwitch;
  final List<DashboardStatCard> statCards;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      backgroundColor: theme.colorScheme.surface,
      appBar: AppBar(
        backgroundColor: theme.colorScheme.surface,
        elevation: 0,
        scrolledUnderElevation: 0,
        leading: IconButton(
          icon: const Icon(Icons.menu_rounded),
          color: theme.colorScheme.onSurface,
          onPressed: openShellDrawer,
        ),
        title: Text(
          AppLocalizations.of(context)!.dashboard,
          style: SoleilTextStyles.titleMedium.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
        actions: [
          NotificationBadge(
            onTap: () =>
                GoRouter.of(context).go(RoutePaths.notifications),
          ),
          const SizedBox(width: 4),
        ],
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(20, 20, 20, 32),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              greeting,
              if (referrerSwitch != null) ...[
                const SizedBox(height: 20),
                referrerSwitch!,
              ],
              if (searchActions != null) ...[
                const SizedBox(height: 24),
                searchActions!,
              ],
              const SizedBox(height: 28),
              DashboardStatGrid(cards: statCards),
            ],
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Agency
// ---------------------------------------------------------------------------

class _AgencyDashboard extends ConsumerWidget {
  const _AgencyDashboard();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final l10n = AppLocalizations.of(context)!;
    final displayName =
        authState.user?['display_name'] as String? ?? 'Agency';

    return _DashboardScaffold(
      greeting: DashboardWelcomeBanner(
        displayName: displayName,
        subtitle: l10n.mobileDashboard_agencySubtitle,
      ),
      searchActions: DashboardSearchActions(
        actions: [
          DashboardSearchAction(
            label: l10n.findFreelancers,
            icon: Icons.person_search_rounded,
            type: 'freelancer',
            tone: DashboardTone.corail(context),
          ),
          DashboardSearchAction(
            label: l10n.findReferrers,
            icon: Icons.handshake_rounded,
            type: 'referrer',
            tone: DashboardTone.amber(context),
          ),
        ],
      ),
      statCards: [
        DashboardStatCard(
          icon: Icons.work_outline_rounded,
          title: l10n.activeMissions,
          value: '0',
          tone: DashboardTone.corail(context),
        ),
        DashboardStatCard(
          icon: Icons.chat_bubble_outline_rounded,
          title: l10n.unreadMessages,
          value: '0',
          tone: DashboardTone.pink(context),
        ),
        DashboardStatCard(
          icon: Icons.trending_up_rounded,
          title: l10n.monthlyRevenue,
          value: '0 EUR',
          tone: DashboardTone.sapin(context),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Enterprise
// ---------------------------------------------------------------------------

class _EnterpriseDashboard extends ConsumerWidget {
  const _EnterpriseDashboard();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final l10n = AppLocalizations.of(context)!;
    final displayName =
        authState.user?['display_name'] as String? ?? 'Enterprise';

    return _DashboardScaffold(
      greeting: DashboardWelcomeBanner(
        displayName: displayName,
        subtitle: l10n.mobileDashboard_enterpriseSubtitle,
      ),
      searchActions: DashboardSearchActions(
        actions: [
          DashboardSearchAction(
            label: l10n.findFreelancers,
            icon: Icons.person_search_rounded,
            type: 'freelancer',
            tone: DashboardTone.corail(context),
          ),
          DashboardSearchAction(
            label: l10n.findAgencies,
            icon: Icons.business_rounded,
            type: 'agency',
            tone: DashboardTone.pink(context),
          ),
          DashboardSearchAction(
            label: l10n.findReferrers,
            icon: Icons.handshake_rounded,
            type: 'referrer',
            tone: DashboardTone.amber(context),
          ),
        ],
      ),
      statCards: [
        DashboardStatCard(
          icon: Icons.folder_open_rounded,
          title: l10n.activeProjects,
          value: '0',
          tone: DashboardTone.corail(context),
        ),
        DashboardStatCard(
          icon: Icons.chat_bubble_outline_rounded,
          title: l10n.unreadMessages,
          value: '0',
          tone: DashboardTone.pink(context),
        ),
        DashboardStatCard(
          icon: Icons.account_balance_wallet_outlined,
          title: l10n.totalBudget,
          value: '0 EUR',
          tone: DashboardTone.sapin(context),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Provider (freelance)
// ---------------------------------------------------------------------------

class _ProviderDashboard extends ConsumerWidget {
  const _ProviderDashboard();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final l10n = AppLocalizations.of(context)!;
    final displayName = authState.user?['first_name'] as String? ??
        authState.user?['display_name'] as String? ??
        'Provider';

    return _DashboardScaffold(
      greeting: DashboardWelcomeBanner(
        displayName: displayName,
        subtitle: l10n.mobileDashboard_providerSubtitle,
      ),
      referrerSwitch: Align(
        alignment: Alignment.centerLeft,
        child: DashboardSwitchPill(
          label: l10n.mobileDashboard_switchToReferrer,
          icon: Icons.swap_horiz_rounded,
          onPressed: () => context.go(RoutePaths.dashboardReferrer),
          tone: DashboardTone.corail(context),
        ),
      ),
      statCards: [
        DashboardStatCard(
          icon: Icons.work_outline_rounded,
          title: l10n.activeMissions,
          value: '0',
          tone: DashboardTone.corail(context),
        ),
        DashboardStatCard(
          icon: Icons.chat_bubble_outline_rounded,
          title: l10n.unreadMessages,
          value: '0',
          tone: DashboardTone.pink(context),
        ),
        DashboardStatCard(
          icon: Icons.trending_up_rounded,
          title: l10n.monthlyRevenue,
          value: '0 EUR',
          tone: DashboardTone.sapin(context),
        ),
      ],
    );
  }
}
