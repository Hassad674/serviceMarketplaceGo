import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/presentation/providers/auth_provider.dart';
import '../../features/auth/presentation/screens/agency_register_screen.dart';
import '../../features/auth/presentation/screens/enterprise_register_screen.dart';
import '../../features/auth/presentation/screens/login_screen.dart';
import '../../features/auth/presentation/screens/register_screen.dart';
import '../../features/auth/presentation/screens/role_selection_screen.dart';
import '../../features/dashboard/presentation/screens/referrer_dashboard_screen.dart';
import '../../features/profile/presentation/screens/profile_screen.dart';
import '../theme/app_theme.dart';

// ---------------------------------------------------------------------------
// Route path constants
// ---------------------------------------------------------------------------

/// Centralized route paths to avoid magic strings.
class RoutePaths {
  RoutePaths._();

  static const String login = '/login';
  static const String register = '/register';
  static const String registerAgency = '/register/agency';
  static const String registerProvider = '/register/provider';
  static const String registerEnterprise = '/register/enterprise';
  static const String dashboard = '/dashboard';
  static const String dashboardReferrer = '/dashboard/referrer';
  static const String messaging = '/messaging';
  static const String missions = '/missions';
  static const String profile = '/profile';
}

// ---------------------------------------------------------------------------
// Auth route list (used by redirect logic)
// ---------------------------------------------------------------------------

const _authRoutes = [
  RoutePaths.login,
  RoutePaths.register,
  RoutePaths.registerAgency,
  RoutePaths.registerProvider,
  RoutePaths.registerEnterprise,
];

// ---------------------------------------------------------------------------
// Router provider
// ---------------------------------------------------------------------------

/// GoRouter with authentication-based redirects.
///
/// Watches [authProvider] to determine whether the user is authenticated
/// and redirects to /login or /dashboard accordingly.
final appRouterProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authProvider);

  return GoRouter(
    initialLocation: RoutePaths.login,
    redirect: (context, state) {
      final isAuthenticated = authState.status == AuthStatus.authenticated;
      final isLoading = authState.status == AuthStatus.loading;
      final isAuthRoute = _authRoutes.contains(state.matchedLocation);

      // Still loading — stay on current route.
      if (isLoading) return null;

      // Not authenticated — force to login (unless already on an auth route).
      if (!isAuthenticated && !isAuthRoute) return RoutePaths.login;

      // Authenticated — redirect away from auth routes to dashboard.
      if (isAuthenticated && isAuthRoute) return RoutePaths.dashboard;

      return null;
    },
    routes: [
      // --- Auth routes (public) ---
      GoRoute(
        path: RoutePaths.login,
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: RoutePaths.register,
        builder: (context, state) => const RoleSelectionScreen(),
      ),
      GoRoute(
        path: RoutePaths.registerAgency,
        builder: (context, state) => const AgencyRegisterScreen(),
      ),
      GoRoute(
        path: RoutePaths.registerProvider,
        builder: (context, state) => const RegisterScreen(),
      ),
      GoRoute(
        path: RoutePaths.registerEnterprise,
        builder: (context, state) => const EnterpriseRegisterScreen(),
      ),

      // --- Authenticated routes (with bottom navigation shell) ---
      ShellRoute(
        builder: (context, state, child) {
          return DashboardShell(child: child);
        },
        routes: [
          GoRoute(
            path: RoutePaths.dashboard,
            builder: (context, state) => const DashboardScreen(),
          ),
          GoRoute(
            path: RoutePaths.dashboardReferrer,
            builder: (context, state) =>
                const ReferrerDashboardScreen(),
          ),
          GoRoute(
            path: RoutePaths.messaging,
            builder: (context, state) =>
                const _PlaceholderScreen(title: 'Messages'),
          ),
          GoRoute(
            path: RoutePaths.missions,
            builder: (context, state) =>
                const _PlaceholderScreen(title: 'My Missions'),
          ),
          GoRoute(
            path: RoutePaths.profile,
            builder: (context, state) => const ProfileScreen(),
          ),
        ],
      ),
    ],
  );
});

// ---------------------------------------------------------------------------
// Dashboard shell with bottom navigation
// ---------------------------------------------------------------------------

/// Wraps authenticated screens with a persistent bottom navigation bar.
class DashboardShell extends StatelessWidget {
  final Widget child;
  const DashboardShell({super.key, required this.child});

  int _currentIndex(BuildContext context) {
    final location = GoRouterState.of(context).matchedLocation;
    if (location.startsWith(RoutePaths.messaging)) return 1;
    if (location.startsWith(RoutePaths.missions)) return 2;
    if (location.startsWith(RoutePaths.profile)) return 3;
    return 0;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: child,
      bottomNavigationBar: NavigationBar(
        selectedIndex: _currentIndex(context),
        destinations: const [
          NavigationDestination(
            icon: Icon(Icons.dashboard_outlined),
            selectedIcon: Icon(Icons.dashboard),
            label: 'Home',
          ),
          NavigationDestination(
            icon: Icon(Icons.chat_outlined),
            selectedIcon: Icon(Icons.chat),
            label: 'Messages',
          ),
          NavigationDestination(
            icon: Icon(Icons.work_outline),
            selectedIcon: Icon(Icons.work),
            label: 'Missions',
          ),
          NavigationDestination(
            icon: Icon(Icons.person_outline),
            selectedIcon: Icon(Icons.person),
            label: 'Profile',
          ),
        ],
        onDestinationSelected: (index) {
          final routes = [
            RoutePaths.dashboard,
            RoutePaths.messaging,
            RoutePaths.missions,
            RoutePaths.profile,
          ];
          GoRouter.of(context).go(routes[index]);
        },
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Dashboard screen — role-based content
// ---------------------------------------------------------------------------

/// Main dashboard / home screen with role-based stats cards.
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
// Agency dashboard
// ---------------------------------------------------------------------------

class _AgencyDashboard extends ConsumerWidget {
  const _AgencyDashboard();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final displayName =
        authState.user?['display_name'] as String? ?? 'Agency';

    return Scaffold(
      appBar: AppBar(
        title: const Text('Marketplace'),
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
              Text(
                'Hello, $displayName',
                style: theme.textTheme.headlineMedium,
              ),
              const SizedBox(height: 4),
              Text(
                'Manage your agency and missions',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: appColors?.mutedForeground,
                ),
              ),
              const SizedBox(height: 24),
              const _StatCard(
                icon: Icons.work_outline,
                title: 'Active Missions',
                value: '0',
                subtitle: 'Active contracts',
                color: Color(0xFF2563EB),
              ),
              const SizedBox(height: 12),
              const _StatCard(
                icon: Icons.chat_outlined,
                title: 'Unread Messages',
                value: '0',
                subtitle: 'Conversations',
                color: Color(0xFF8B5CF6),
              ),
              const SizedBox(height: 12),
              const _StatCard(
                icon: Icons.trending_up,
                title: 'Monthly Revenue',
                value: '0 EUR',
                subtitle: 'This month',
                color: Color(0xFF22C55E),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Enterprise dashboard
// ---------------------------------------------------------------------------

class _EnterpriseDashboard extends ConsumerWidget {
  const _EnterpriseDashboard();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final displayName =
        authState.user?['display_name'] as String? ?? 'Enterprise';

    return Scaffold(
      appBar: AppBar(
        title: const Text('Marketplace'),
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
              Text(
                'Hello, $displayName',
                style: theme.textTheme.headlineMedium,
              ),
              const SizedBox(height: 4),
              Text(
                'Find the best providers for your projects',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: appColors?.mutedForeground,
                ),
              ),
              const SizedBox(height: 24),
              const _StatCard(
                icon: Icons.folder_open_outlined,
                title: 'Active Projects',
                value: '0',
                subtitle: 'Active projects',
                color: Color(0xFF2563EB),
              ),
              const SizedBox(height: 12),
              const _StatCard(
                icon: Icons.chat_outlined,
                title: 'Unread Messages',
                value: '0',
                subtitle: 'Conversations',
                color: Color(0xFF8B5CF6),
              ),
              const SizedBox(height: 12),
              const _StatCard(
                icon: Icons.account_balance_wallet_outlined,
                title: 'Total Budget',
                value: '0 EUR',
                subtitle: 'Spent this month',
                color: Color(0xFF22C55E),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Provider (freelance) dashboard
// ---------------------------------------------------------------------------

class _ProviderDashboard extends ConsumerWidget {
  const _ProviderDashboard();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final displayName =
        authState.user?['first_name'] as String? ??
        authState.user?['display_name'] as String? ??
        'Provider';

    return Scaffold(
      appBar: AppBar(
        title: const Text('Marketplace'),
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
              Text(
                'Hello, $displayName',
                style: theme.textTheme.headlineMedium,
              ),
              const SizedBox(height: 4),
              Text(
                'Manage your missions and grow your business',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: appColors?.mutedForeground,
                ),
              ),
              const SizedBox(height: 16),

              // Switch to referrer mode
              SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  onPressed: () => context.go(RoutePaths.dashboardReferrer),
                  icon: const Icon(Icons.swap_horiz),
                  label: const Text('Business Referrer Mode'),
                ),
              ),
              const SizedBox(height: 24),

              const _StatCard(
                icon: Icons.work_outline,
                title: 'Active Missions',
                value: '0',
                subtitle: 'Active contracts',
                color: Color(0xFF2563EB),
              ),
              const SizedBox(height: 12),
              const _StatCard(
                icon: Icons.chat_outlined,
                title: 'Unread Messages',
                value: '0',
                subtitle: 'Conversations',
                color: Color(0xFF8B5CF6),
              ),
              const SizedBox(height: 12),
              const _StatCard(
                icon: Icons.trending_up,
                title: 'Monthly Revenue',
                value: '0 EUR',
                subtitle: 'This month',
                color: Color(0xFF22C55E),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Shared stat card widget
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Placeholder screen
// ---------------------------------------------------------------------------

/// Temporary placeholder for screens that have not been implemented yet.
class _PlaceholderScreen extends StatelessWidget {
  const _PlaceholderScreen({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: Text(title)),
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.construction,
              size: 48,
              color: theme.colorScheme.primary,
            ),
            const SizedBox(height: 16),
            Text(title, style: theme.textTheme.headlineMedium),
            const SizedBox(height: 8),
            Text(
              'Coming soon',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
