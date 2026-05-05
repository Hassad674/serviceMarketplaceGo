import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../notification/presentation/widgets/notification_badge.dart';
import '../widgets/dashboard_atoms.dart';

/// Referrer (business referrer / "apporteur d'affaire") dashboard — Soleil v2.
///
/// Same editorial anatomy as [DashboardScreen] (corail mono eyebrow,
/// Fraunces italic title, tabac subtitle, Soleil stat grid) but with the
/// referrer-specific 4-stat layout: Filleuls / Missions actives /
/// Missions terminées / Commissions.
class ReferrerDashboardScreen extends ConsumerWidget {
  const ReferrerDashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final authState = ref.watch(authProvider);
    final l10n = AppLocalizations.of(context)!;

    final user = authState.user;
    final displayName =
        user?['first_name'] as String? ??
        user?['display_name'] as String? ??
        'Referrer';

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
          l10n.referrerMode,
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
              DashboardWelcomeBanner(
                displayName: displayName,
                subtitle: l10n.mobileDashboard_referrerSubtitle,
                eyebrow: l10n.mobileDashboard_referrerEyebrow,
              ),
              const SizedBox(height: 20),
              Align(
                alignment: Alignment.centerLeft,
                child: DashboardSwitchPill(
                  label: l10n.mobileDashboard_switchToFreelance,
                  icon: Icons.swap_horiz_rounded,
                  onPressed: () => context.go(RoutePaths.dashboard),
                  tone: DashboardTone.sapin(context),
                ),
              ),
              const SizedBox(height: 24),
              DashboardSearchActions(
                actions: [
                  DashboardSearchAction(
                    label: l10n.findFreelancers,
                    icon: Icons.person_search_rounded,
                    type: 'freelancer',
                    tone: DashboardTone.corail(context),
                  ),
                ],
              ),
              const SizedBox(height: 28),
              DashboardStatGrid(
                cards: [
                  DashboardStatCard(
                    icon: Icons.handshake_rounded,
                    title: l10n.referrals,
                    value: '0',
                    tone: DashboardTone.corail(context),
                  ),
                  DashboardStatCard(
                    icon: Icons.schedule_rounded,
                    title: l10n.activeMissions,
                    value: '0',
                    tone: DashboardTone.pink(context),
                  ),
                  DashboardStatCard(
                    icon: Icons.check_circle_outline_rounded,
                    title: l10n.completedMissions,
                    value: '0',
                    tone: DashboardTone.sapin(context),
                  ),
                  DashboardStatCard(
                    icon: Icons.trending_up_rounded,
                    title: l10n.commissions,
                    value: '0 EUR',
                    tone: DashboardTone.amber(context),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
