import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../notification/presentation/widgets/notification_badge.dart';
import '../widgets/dashboard_atoms.dart';
import '../widgets/role_layouts.dart';

/// Main dashboard / home screen — Soleil v2.
///
/// Role-aware dispatch (provider / agency / enterprise / referrer-mode)
/// over a single editorial layout: corail mono eyebrow, Fraunces
/// italic-corail welcome title, role-specific subtitle, then the
/// role-specific stat tiles + actions todo card. (D2)
///
/// Provider/Agency: tied to /me/stats/visibility (D1 endpoint).
/// Enterprise: tied to /me/stats/enterprise-applications + my-jobs.
/// Referrer (when `referrer_enabled === true`): pure placeholders for
/// now — referral stats hooks not on mobile yet (D3+).
///
/// Out-of-scope flagged:
///   - "Pipeline" section (no clean mission-feed hook on mobile)
///   - Wallet/monthly revenue tile (no monthly-revenue hook on mobile)
///   - "Cette semaine" editorial card / "Atelier Premium" CTA — same
///     scope flags as before (W-11 batch).
class DashboardScreen extends ConsumerWidget {
  const DashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final user = authState.user;
    final role = user?['role'] as String?;
    final referrerEnabled = (user?['referrer_enabled'] as bool?) ?? false;

    final l10n = AppLocalizations.of(context)!;
    final displayName = user?['first_name'] as String? ??
        user?['display_name'] as String? ??
        '';

    Widget layout;
    Widget greeting;
    Widget? referrerSwitch;

    switch (role) {
      case 'enterprise':
        layout = const EnterpriseRoleLayout();
        greeting = DashboardWelcomeBanner(
          displayName: displayName,
          subtitle: l10n.mobileDashboard_enterpriseSubtitle,
        );
        break;
      case 'agency':
        layout = const ProviderRoleLayout();
        greeting = DashboardWelcomeBanner(
          displayName: displayName,
          subtitle: l10n.mobileDashboard_agencySubtitle,
        );
        break;
      case 'provider':
      default:
        layout = const ProviderRoleLayout();
        greeting = DashboardWelcomeBanner(
          displayName: displayName,
          subtitle: l10n.mobileDashboard_providerSubtitle,
        );
        if (referrerEnabled) {
          referrerSwitch = Align(
            alignment: Alignment.centerLeft,
            child: DashboardSwitchPill(
              label: l10n.mobileDashboard_switchToReferrer,
              icon: Icons.swap_horiz_rounded,
              onPressed: () => context.go(RoutePaths.dashboardReferrer),
              tone: DashboardTone.corail(context),
            ),
          );
        }
        break;
    }

    return _DashboardShell(
      greeting: greeting,
      referrerSwitch: referrerSwitch,
      body: layout,
    );
  }
}

class _DashboardShell extends StatelessWidget {
  const _DashboardShell({
    required this.greeting,
    required this.body,
    this.referrerSwitch,
  });

  final Widget greeting;
  final Widget body;
  final Widget? referrerSwitch;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
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
          l10n.dashboard,
          style: SoleilTextStyles.titleMedium.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
        actions: [
          NotificationBadge(
            onTap: () => GoRouter.of(context).go(RoutePaths.notifications),
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
              const SizedBox(height: 28),
              body,
            ],
          ),
        ),
      ),
    );
  }
}
