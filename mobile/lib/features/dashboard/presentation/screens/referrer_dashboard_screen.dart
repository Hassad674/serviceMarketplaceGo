import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../auth/presentation/providers/auth_provider.dart';

/// Referrer (apporteur d'affaires) dashboard for providers with referrer mode.
///
/// Shows referrer-specific stats and a button to switch back to freelance mode.
class ReferrerDashboardScreen extends ConsumerWidget {
  const ReferrerDashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    final user = authState.user;
    final displayName =
        user?['first_name'] as String? ??
        user?['display_name'] as String? ??
        'Apporteur';

    return Scaffold(
      appBar: AppBar(
        title: const Text('Mode Apporteur'),
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
              // Greeting
              Text(
                'Bonjour, $displayName',
                style: theme.textTheme.headlineMedium,
              ),
              const SizedBox(height: 4),
              Text(
                'Gerez vos apports d\'affaires et vos commissions',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: appColors?.mutedForeground,
                ),
              ),
              const SizedBox(height: 24),

              // Switch to freelance mode
              SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  onPressed: () => context.go(RoutePaths.dashboard),
                  icon: const Icon(Icons.swap_horiz),
                  label: const Text('Dashboard Freelance'),
                ),
              ),
              const SizedBox(height: 24),

              // Stats grid
              const _StatCard(
                icon: Icons.handshake_outlined,
                title: 'Mises en relation',
                value: '0',
                subtitle: 'En attente de reponse',
                color: Color(0xFF14B8A6), // teal-500
              ),
              const SizedBox(height: 12),

              const _StatCard(
                icon: Icons.schedule,
                title: 'Missions en cours',
                value: '0',
                subtitle: 'Contrats actifs',
                color: Color(0xFFF59E0B), // amber-500
              ),
              const SizedBox(height: 12),

              const _StatCard(
                icon: Icons.check_circle_outline,
                title: 'Missions terminees',
                value: '0',
                subtitle: 'Total historique',
                color: Color(0xFF22C55E), // emerald-500
              ),
              const SizedBox(height: 12),

              const _StatCard(
                icon: Icons.trending_up,
                title: 'Commissions',
                value: '0 EUR',
                subtitle: 'Total gagne',
                color: Color(0xFFF43F5E), // rose-500
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// A stat card matching the web dashboard design.
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
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Row(
        children: [
          Container(
            width: 48,
            height: 48,
            decoration: BoxDecoration(
              color: color.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(24),
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
