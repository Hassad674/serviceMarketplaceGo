import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/presentation/screens/login_screen.dart';
import '../../features/auth/presentation/screens/register_screen.dart';
import '../../features/auth/presentation/screens/test_screen.dart';
import '../../features/auth/presentation/providers/auth_provider.dart';

// ---------------------------------------------------------------------------
// Route path constants
// ---------------------------------------------------------------------------

/// Centralized route paths to avoid magic strings.
class RoutePaths {
  RoutePaths._();

  static const String test = '/test';
  static const String login = '/login';
  static const String register = '/register';
  static const String dashboard = '/dashboard';
  static const String messaging = '/messaging';
  static const String missions = '/missions';
  static const String profile = '/profile';
  static const String settings = '/settings';
}

// ---------------------------------------------------------------------------
// Router provider
// ---------------------------------------------------------------------------

/// GoRouter with authentication-based redirects.
///
/// Watches [authStateProvider] to determine whether the user is authenticated
/// and redirects to /login or /dashboard accordingly.
final appRouterProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authProvider);

  return GoRouter(
    initialLocation: RoutePaths.login,
    redirect: (context, state) {
      final isAuthenticated = authState.status == AuthStatus.authenticated;
      final isLoading = authState.status == AuthStatus.loading;
      final isAuthRoute = state.matchedLocation == RoutePaths.login ||
          state.matchedLocation == RoutePaths.register ||
          state.matchedLocation == RoutePaths.test;

      // Still loading — stay on current route.
      if (isLoading) return null;

      // Not authenticated — force to login (unless already on an auth route).
      if (!isAuthenticated && !isAuthRoute) return RoutePaths.login;

      // Authenticated — redirect away from auth routes.
      if (isAuthenticated && isAuthRoute) return RoutePaths.dashboard;

      return null;
    },
    routes: [
      // --- Test route (temporary, public) ---
      GoRoute(
        path: RoutePaths.test,
        builder: (context, state) => const TestScreen(),
      ),

      // --- Auth routes (public) ---
      GoRoute(
        path: RoutePaths.login,
        builder: (context, state) => const LoginScreen(),
      ),
      GoRoute(
        path: RoutePaths.register,
        builder: (context, state) => const RegisterScreen(),
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
            path: RoutePaths.messaging,
            builder: (context, state) =>
                const _PlaceholderScreen(title: 'Messagerie'),
          ),
          GoRoute(
            path: RoutePaths.missions,
            builder: (context, state) =>
                const _PlaceholderScreen(title: 'Mes Missions'),
          ),
          GoRoute(
            path: RoutePaths.profile,
            builder: (context, state) =>
                const _PlaceholderScreen(title: 'Mon Profil'),
          ),
          GoRoute(
            path: RoutePaths.settings,
            builder: (context, state) =>
                const _PlaceholderScreen(title: 'Parametres'),
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
            label: 'Accueil',
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
            label: 'Profil',
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
// Dashboard screen
// ---------------------------------------------------------------------------

/// Main dashboard / home screen (stub — to be implemented per feature).
class DashboardScreen extends ConsumerWidget {
  const DashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final theme = Theme.of(context);
    final userName = authState.user?['name'] as String? ?? '';

    return Scaffold(
      appBar: AppBar(
        title: const Text('Marketplace'),
        actions: [
          IconButton(
            icon: const Icon(Icons.notifications_outlined),
            onPressed: () {
              // TODO: navigate to notifications
            },
          ),
          IconButton(
            icon: const Icon(Icons.settings_outlined),
            onPressed: () => context.go(RoutePaths.settings),
          ),
        ],
      ),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Bonjour${userName.isNotEmpty ? ', $userName' : ''} !',
                style: theme.textTheme.headlineMedium,
              ),
              const SizedBox(height: 8),
              Text(
                'Bienvenue sur votre tableau de bord',
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurface.withOpacity(0.6),
                ),
              ),
              const SizedBox(height: 32),
              const Expanded(
                child: Center(
                  child: Text('Tableau de bord en construction'),
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
              'Bientot disponible',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurface.withOpacity(0.6),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
