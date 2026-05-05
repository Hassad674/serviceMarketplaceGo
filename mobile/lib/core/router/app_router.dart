import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/presentation/providers/auth_provider.dart';
import '../../features/freelance_profile/presentation/screens/freelance_profile_screen.dart';
import '../../features/messaging/data/messaging_ws_service.dart';
import '../../features/messaging/presentation/providers/messaging_provider.dart';
import '../../features/profile/presentation/screens/profile_screen.dart';
import '../../l10n/app_localizations.dart';
import '../../shared/widgets/app_drawer.dart';
import '../../shared/widgets/kyc_banner.dart';
import '../notifications/fcm_service.dart';
import '../theme/app_theme.dart';
import 'routes/auth_routes.dart';
import 'routes/dashboard_routes.dart';
import 'routes/payment_routes.dart';
import 'routes/profile_routes.dart';
import 'routes/team_routes.dart';
import '../theme/app_palette.dart';

// Re-export DashboardScreen so existing imports of `app_router.dart` for
// the symbol keep compiling without requiring a callsite change.
export '../../features/dashboard/presentation/screens/dashboard_screen.dart'
    show DashboardScreen;

/// Global navigator key — used by [CallEventListener] and [FCMService]
/// to push modal screens from above the GoRouter navigator in the
/// widget tree. **CRITICAL: do not move this constant** — the FCM tap
/// navigation regression suite depends on the verbatim symbol.
final rootNavigatorKey = GlobalKey<NavigatorState>();

// ---------------------------------------------------------------------------
// Route path constants
// ---------------------------------------------------------------------------

/// Centralized route paths to avoid magic strings.
///
/// **CRITICAL: tests at `test/core/notifications/fcm_route_test.dart`
/// import these constants verbatim** — the FCM tap navigation regression
/// suite must keep importing the same symbols.
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
  static const String projects = '/projects';
  static const String projectsNew = '/projects/new';
  static const String jobs = '/jobs';
  static const String jobsCreate = '/jobs/create';
  static const String projectsPay = '/projects/pay';
  static const String projectsList = '/projects/list';
  static const String profile = '/profile';
  static const String clientProfile = '/client-profile';
  static const String clientPublic = '/clients';
  static const String referralProfile = '/referral';

  /// Read-only split-profile public routes introduced with the
  /// freelance/referrer split. The legacy `/profiles/:id` path is
  /// still used by agencies and as a fallback when the caller has
  /// no hint about the persona they are opening.
  static const String freelancerPublic = '/freelancers';
  static const String referrerPublic = '/referrers';
  static const String paymentInfo = '/payment-info';
  static const String wallet = '/wallet';
  static const String team = '/team';
  static const String notifications = '/notifications';
  static const String search = '/search';
  static const String publicProfile = '/profiles';
  static const String chat = '/chat';
  static const String newChat = '/new-chat';
  static const String proposalDetail = '/projects/detail';
  static const String opportunities = '/opportunities';
  static const String opportunityDetail = '/opportunities/detail';
  static const String myApplications = '/my-applications';
  static const String jobCandidates = '/jobs/candidates';
  static const String jobDetail = '/jobs/detail';
  static const String jobEdit = '/jobs/edit';
  static const String candidateDetail = '/candidates/detail';
  static const String disputeOpen = '/disputes/open';
  static const String disputeCounter = '/disputes/counter';
  static const String referralsDashboard = '/referrals';
  static const String referralCreate = '/referrals/new';

  // Subscription (Premium) — pricing page + Stripe Checkout landings.
  static const String pricing = '/pricing';
  static const String billingSuccess = '/billing/success';
  static const String billingCancel = '/billing/cancel';
  // In-app WebView host for Stripe Checkout / Customer Portal URLs.
  // Reached by push-navigation from the subscription flow, not a
  // direct URL — the target Stripe URL is passed via `extra`.
  static const String checkoutWebview = '/billing/checkout';

  // Invoicing — full-screen settings/utility routes reached from the
  // drawer or from the gate modal. NOT inside the bottom-nav shell.
  static const String billingProfile = '/settings/billing-profile';
  static const String invoices = '/invoices';

  // Account preferences — mirror of web /account?section=…. The mobile
  // equivalent is a single screen surfacing the sections sequentially
  // (no sidebar tabs on a 390-wide viewport). Reached from the drawer.
  static const String account = '/account';
  static const String accountDelete = '/account/delete';
  static const String accountCancelDeletion = '/account/cancel-deletion';
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

/// Returns true if the given [location] is a publicly accessible route
/// that does not require authentication (search results, public profiles).
bool _isPublicRoute(String location) {
  return location.startsWith('/profiles/') ||
      location.startsWith('/freelancers/') ||
      location.startsWith('/referrers/') ||
      location.startsWith('/clients/') ||
      location.startsWith('/search/');
}

// ---------------------------------------------------------------------------
// Router provider
// ---------------------------------------------------------------------------

/// GoRouter with authentication-based redirects.
///
/// Watches [authProvider] to determine whether the user is authenticated
/// and redirects to /login or /dashboard accordingly. The route table
/// is composed from focused builders in `lib/core/router/routes/`.
final appRouterProvider = Provider<GoRouter>((ref) {
  final authState = ref.watch(authProvider);

  return GoRouter(
    navigatorKey: rootNavigatorKey,
    initialLocation: RoutePaths.login,
    redirect: (context, state) {
      final isAuthenticated = authState.status == AuthStatus.authenticated;
      final isLoading = authState.status == AuthStatus.loading;
      final isAuthRoute = _authRoutes.contains(state.matchedLocation);
      final isPublicRoute = _isPublicRoute(state.matchedLocation);

      // Still loading — stay on current route.
      if (isLoading) return null;

      // Not authenticated — force to login (unless on auth or public route).
      if (!isAuthenticated && !isAuthRoute && !isPublicRoute) {
        return RoutePaths.login;
      }

      // Authenticated — redirect away from auth routes to dashboard.
      if (isAuthenticated && isAuthRoute) return RoutePaths.dashboard;

      return null;
    },
    routes: [
      // --- Auth routes (public) ---
      ...buildAuthRoutes(),

      // --- Public profile + search + chat (no bottom nav) ---
      ...buildProfileRoutes(),

      // --- Proposal / dispute / referral / subscription / invoicing ---
      ...buildPaymentRoutes(),

      // --- Authenticated routes (with bottom navigation shell) ---
      ShellRoute(
        builder: (context, state, child) => DashboardShell(child: child),
        routes: [
          ...buildDashboardShellRoutes(),
          ...buildTeamShellRoutes(),
        ],
      ),
    ],
  );
});

// ---------------------------------------------------------------------------
// Profile dispatcher — selects freelance vs legacy based on org type
// ---------------------------------------------------------------------------

/// Builder used by `/profile` to dispatch between the freelance and
/// legacy profile screens based on the signed-in user's org type.
/// `provider_personal` users get the new split-profile
/// [FreelanceProfileScreen]; any other org type (currently just
/// `agency`) keeps rendering the legacy [ProfileScreen] until the
/// agency refactor ships.
Widget profileDispatcherBuilder(BuildContext context, GoRouterState state) {
  return const _ProfileDispatcher();
}

class _ProfileDispatcher extends ConsumerWidget {
  const _ProfileDispatcher();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final orgType = authState.organization?['type'] as String?;
    if (orgType == 'provider_personal') {
      return const FreelanceProfileScreen();
    }
    return const ProfileScreen();
  }
}

// ---------------------------------------------------------------------------
// Drawer open helper — used by inner screens to trigger the shell drawer
// ---------------------------------------------------------------------------

/// Opens the [DashboardShell] drawer from any inner screen's AppBar.
void openShellDrawer() {
  DashboardShell.scaffoldKey.currentState?.openDrawer();
}

// ---------------------------------------------------------------------------
// Dashboard shell with bottom navigation
// ---------------------------------------------------------------------------

/// Wraps authenticated screens with a persistent bottom navigation bar
/// and a navigation drawer accessible via hamburger icon.
///
/// Reads [totalUnreadProvider] to display a badge on the Messages tab.
/// Call event handling is done globally by [CallEventListener] in main.dart.
class DashboardShell extends ConsumerStatefulWidget {
  /// Key for the outer Scaffold — inner screens use this to open the drawer.
  static final scaffoldKey = GlobalKey<ScaffoldState>();

  final Widget child;
  const DashboardShell({super.key, required this.child});

  @override
  ConsumerState<DashboardShell> createState() => _DashboardShellState();
}

class _DashboardShellState extends ConsumerState<DashboardShell> {
  StreamSubscription<Map<String, dynamic>>? _notifSub;

  @override
  void initState() {
    super.initState();
    final wsSvc = ref.read(messagingWsServiceProvider);
    _notifSub = wsSvc.events.listen(_handleWsNotification);

    // FCM init is heavy (channel setup + permission prompt). Schedule
    // it after the first frame so the shell renders interactively
    // before the platform plugin handshake begins. `addPostFrameCallback`
    // (rather than `Future.microtask`) ensures the user actually sees
    // the dashboard before the OS-level permission sheet appears.
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      FCMService.initialize(ref);
    });
  }

  void _handleWsNotification(Map<String, dynamic> event) {
    if (event['type'] != 'notification' || !mounted) return;
    final payload = event['payload'];
    if (payload is! Map || payload['title'] == null) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              payload['title'] as String,
              style: const TextStyle(
                fontWeight: FontWeight.w600,
                fontSize: 14,
              ),
            ),
            if (payload['body'] != null &&
                (payload['body'] as String).isNotEmpty)
              Text(
                payload['body'] as String,
                style: const TextStyle(fontSize: 12),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
          ],
        ),
        behavior: SnackBarBehavior.floating,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(10),
        ),
        margin: const EdgeInsets.fromLTRB(16, 0, 16, 16),
        duration: const Duration(seconds: 3),
      ),
    );
  }

  @override
  void dispose() {
    _notifSub?.cancel();
    super.dispose();
  }

  int _currentIndex(BuildContext context) {
    final location = GoRouterState.of(context).matchedLocation;
    if (location.startsWith(RoutePaths.messaging)) return 1;
    if (location.startsWith(RoutePaths.missions)) return 2;
    if (location.startsWith(RoutePaths.profile)) return 3;
    return 0;
  }

  @override
  Widget build(BuildContext context) {
    // Build is intentionally lean — we only depend on the location
    // (for the selected tab index) and pass the unread badge down
    // to a ConsumerWidget leaf that watches `totalUnreadProvider`
    // independently. A WS push that changes the unread count no
    // longer rebuilds KYCBanner / child / drawer / scaffold chrome
    // (PERF-M-01 / PERF-M-08).
    return Scaffold(
      key: DashboardShell.scaffoldKey,
      drawer: const AppDrawer(),
      body: Column(
        children: [
          const KYCBanner(),
          Expanded(child: widget.child),
        ],
      ),
      bottomNavigationBar: _ShellBottomNav(
        selectedIndex: _currentIndex(context),
      ),
    );
  }
}

/// Bottom navigation bar isolated from the [DashboardShell] so the
/// `totalUnreadProvider` watch only invalidates the navbar subtree,
/// not the entire shell + child.
class _ShellBottomNav extends ConsumerWidget {
  const _ShellBottomNav({required this.selectedIndex});

  final int selectedIndex;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final totalUnread = ref.watch(totalUnreadProvider);

    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(
          top: BorderSide(
            color: appColors?.border ?? theme.dividerColor,
            width: 1,
          ),
        ),
      ),
      child: NavigationBar(
        selectedIndex: selectedIndex,
        destinations: _buildDestinations(l10n, totalUnread),
        onDestinationSelected: (index) {
          const routes = [
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

  List<NavigationDestination> _buildDestinations(
    AppLocalizations l10n,
    int totalUnread,
  ) {
    final messagesIcon = totalUnread > 0
        ? Badge(
            label: Text(
              totalUnread > 99 ? '99+' : '$totalUnread',
              style: const TextStyle(
                fontSize: 10,
                fontWeight: FontWeight.bold,
              ),
            ),
            backgroundColor: AppPalette.rose500,
            child: const Icon(Icons.chat_outlined),
          )
        : const Icon(Icons.chat_outlined);
    final messagesSelectedIcon = totalUnread > 0
        ? Badge(
            label: Text(
              totalUnread > 99 ? '99+' : '$totalUnread',
              style: const TextStyle(
                fontSize: 10,
                fontWeight: FontWeight.bold,
              ),
            ),
            backgroundColor: AppPalette.rose500,
            child: const Icon(Icons.chat),
          )
        : const Icon(Icons.chat);

    return [
      NavigationDestination(
        icon: const Icon(Icons.dashboard_outlined),
        selectedIcon: const Icon(Icons.dashboard),
        label: l10n.home,
      ),
      NavigationDestination(
        icon: messagesIcon,
        selectedIcon: messagesSelectedIcon,
        label: l10n.messages,
      ),
      NavigationDestination(
        icon: const Icon(Icons.work_outline),
        selectedIcon: const Icon(Icons.work),
        label: l10n.missions,
      ),
      NavigationDestination(
        icon: const Icon(Icons.person_outline),
        selectedIcon: const Icon(Icons.person),
        label: l10n.profile,
      ),
    ];
  }
}
