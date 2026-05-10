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

/// Referrer (business referrer / "apporteur d'affaire") dashboard — Soleil v2.
///
/// Same editorial anatomy as [DashboardScreen] but with the referrer
/// 4-tile layout (active referrals / pending commissions / paid out /
/// lifetime). Tiles render em-dashes until the referral stats hooks ship
/// (D3+).
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
              const SizedBox(height: 28),
              const ReferrerRoleLayout(),
            ],
          ),
        ),
      ),
    );
  }
}
