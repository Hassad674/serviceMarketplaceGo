import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';

// Rose color constant for the search chip.
const Color _rose500 = Color(0xFFF43F5E);

/// Referrer (business referrer) dashboard for providers with referrer mode.
///
/// Shows referrer-specific stats and a button to switch back to freelance mode.
class ReferrerDashboardScreen extends ConsumerWidget {
  const ReferrerDashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final l10n = AppLocalizations.of(context)!;

    final user = authState.user;
    final displayName =
        user?['first_name'] as String? ??
        user?['display_name'] as String? ??
        'Referrer';

    return Scaffold(
      appBar: AppBar(
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.referrerMode),
        actions: [
          IconButton(
            icon: const Icon(Icons.notifications_outlined),
            onPressed: () {},
          ),
        ],
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Welcome banner with gradient
              _WelcomeBanner(
                displayName: displayName,
                subtitle: l10n.roleFreelanceDesc,
              ),
              const SizedBox(height: 16),

              // Switch to freelance mode
              SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  onPressed: () => context.go(RoutePaths.dashboard),
                  icon: const Icon(Icons.swap_horiz),
                  label: Text(l10n.freelanceDashboard),
                ),
              ),
              const SizedBox(height: 24),

              // Search action
              ActionChip(
                avatar: const Icon(Icons.person_search, size: 18, color: _rose500),
                label: Text(
                  l10n.findFreelancers,
                  style: TextStyle(
                    color: Theme.of(context).colorScheme.onSurface,
                    fontWeight: FontWeight.w500,
                    fontSize: 13,
                  ),
                ),
                backgroundColor: _rose500.withValues(alpha: 0.08),
                side: BorderSide(color: _rose500.withValues(alpha: 0.2)),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
                onPressed: () => context.push('/search/freelancer'),
              ),
              const SizedBox(height: 24),

              // Stats grid — 4 cards for referrer
              _StatCard(
                icon: Icons.handshake_outlined,
                title: l10n.referrals,
                value: '0',
                subtitle: l10n.pendingResponse,
                color: const Color(0xFF14B8A6), // teal-500
              ),
              const SizedBox(height: 12),

              _StatCard(
                icon: Icons.schedule,
                title: l10n.activeMissions,
                value: '0',
                subtitle: l10n.activeContracts,
                color: const Color(0xFFF59E0B), // amber-500
              ),
              const SizedBox(height: 12),

              _StatCard(
                icon: Icons.check_circle_outline,
                title: l10n.completedMissions,
                value: '0',
                subtitle: l10n.totalHistory,
                color: const Color(0xFF22C55E), // emerald-500
              ),
              const SizedBox(height: 12),

              _StatCard(
                icon: Icons.trending_up,
                title: l10n.commissions,
                value: '0 EUR',
                subtitle: l10n.totalEarned,
                color: const Color(0xFFF43F5E), // rose-500
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Welcome banner — gradient rose to teal (referrer variant)
// ---------------------------------------------------------------------------

class _WelcomeBanner extends StatelessWidget {
  const _WelcomeBanner({
    required this.displayName,
    required this.subtitle,
  });

  final String displayName;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        gradient: const LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [
            Color(0xFF14B8A6), // teal-500
            Color(0xFF8B5CF6), // violet-500
          ],
        ),
        boxShadow: [
          BoxShadow(
            color: const Color(0xFF14B8A6).withValues(alpha: 0.3),
            blurRadius: 20,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: Builder(
        builder: (context) {
          final l10n = AppLocalizations.of(context)!;
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                l10n.welcomeBack,
                style: TextStyle(
                  color: Colors.white.withValues(alpha: 0.85),
                  fontSize: 15,
                  fontWeight: FontWeight.w400,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                displayName,
                style: const TextStyle(
                  color: Colors.white,
                  fontSize: 24,
                  fontWeight: FontWeight.bold,
                  letterSpacing: -0.3,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                subtitle,
                style: TextStyle(
                  color: Colors.white.withValues(alpha: 0.8),
                  fontSize: 14,
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Stat card — premium design
// ---------------------------------------------------------------------------

class _StatCard extends StatelessWidget {
  const _StatCard({
    required this.icon,
    required this.title,
    required this.value,
    required this.subtitle,
    required this.color,
  });

  final IconData icon;
  final String title;
  final String value;
  final String subtitle;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Row(
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: color.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
            child: Icon(icon, color: color, size: 22),
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight: FontWeight.w500,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  value,
                  style: theme.textTheme.headlineMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                Text(
                  subtitle,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
