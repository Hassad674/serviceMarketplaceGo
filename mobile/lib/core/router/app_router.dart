import 'dart:async';

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
import '../../features/messaging/data/messaging_ws_service.dart';
import '../../features/messaging/presentation/providers/messaging_provider.dart';
import '../../features/messaging/presentation/screens/chat_screen.dart';
import '../../features/messaging/presentation/screens/messaging_screen.dart';
import '../../features/messaging/presentation/screens/new_chat_screen.dart';
import '../../features/freelance_profile/presentation/screens/freelance_profile_screen.dart';
import '../../features/freelance_profile/presentation/screens/freelance_public_profile_screen.dart';
import '../../features/profile/presentation/screens/profile_screen.dart';
import '../../features/referrer_profile/presentation/screens/referrer_profile_screen.dart';
import '../../features/referrer_profile/presentation/screens/referrer_public_profile_screen.dart';
import '../../features/job/presentation/screens/create_job_screen.dart';
import '../../features/job/presentation/screens/jobs_screen.dart';
import '../../features/job/presentation/screens/opportunities_screen.dart';
import '../../features/job/presentation/screens/opportunity_detail_screen.dart';
import '../../features/job/presentation/screens/my_applications_screen.dart';
import '../../features/job/domain/entities/job_application_entity.dart';
import '../../features/job/presentation/screens/candidate_detail_screen.dart';
import '../../features/job/presentation/screens/candidates_screen.dart';
import '../../features/job/presentation/screens/job_detail_screen.dart';
import '../../features/proposal/domain/entities/proposal_entity.dart';
import '../../features/proposal/presentation/screens/create_proposal_screen.dart';
import '../../features/proposal/presentation/screens/payment_simulation_screen.dart';
import '../../features/proposal/presentation/screens/projects_list_screen.dart';
import '../../features/dispute/presentation/screens/counter_propose_screen.dart';
import '../../features/dispute/presentation/screens/open_dispute_screen.dart';
import '../../features/referral/presentation/screens/referral_creation_screen.dart';
import '../../features/referral/presentation/screens/referral_dashboard_screen.dart';
import '../../features/referral/presentation/screens/referral_detail_screen.dart';
import '../../features/proposal/presentation/screens/proposal_detail_screen.dart';
import '../../features/search/presentation/screens/public_profile_screen.dart';
import '../../features/team/presentation/screens/team_screen.dart';
import '../../features/notification/presentation/screens/notification_screen.dart';
import '../../features/notification/presentation/widgets/notification_badge.dart';
import '../../features/payment_info/presentation/screens/payment_info_screen.dart';
import '../../features/search/presentation/screens/search_screen.dart';
import '../../features/wallet/presentation/screens/wallet_screen.dart';
import '../../l10n/app_localizations.dart';
import '../../shared/widgets/app_drawer.dart';
import '../../shared/widgets/kyc_banner.dart';
import '../notifications/fcm_service.dart';
import '../theme/app_theme.dart';

/// Global navigator key — used by [CallEventListener] to push modal
/// screens from above the GoRouter navigator in the widget tree.
final rootNavigatorKey = GlobalKey<NavigatorState>();

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
  static const String projects = '/projects';
  static const String projectsNew = '/projects/new';
  static const String jobs = '/jobs';
  static const String jobsCreate = '/jobs/create';
  static const String projectsPay = '/projects/pay';
  static const String projectsList = '/projects/list';
  static const String profile = '/profile';
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
      location.startsWith('/search/');
}

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

      // --- Public profile route (accessible without bottom nav) ---
      // The path param is an org id — profiles describe the team's
      // shared marketplace identity since phase R2.
      GoRoute(
        path: '/profiles/:id',
        builder: (context, state) {
          final extras = state.extra as Map<String, dynamic>?;
          return PublicProfileScreen(
            orgId: state.pathParameters['id'] ?? '',
            displayName: extras?['display_name'] as String?,
            orgType: extras?['org_type'] as String?,
          );
        },
      ),

      // --- Split-profile public routes (accessible without bottom nav) ---
      // Introduced with the freelance/referrer split so each persona
      // has its own shareable URL. The path param is an organization
      // id. The extras map may carry display_name to seed the header
      // while the payload is in flight.
      GoRoute(
        path: '/freelancers/:id',
        builder: (context, state) {
          final extras = state.extra as Map<String, dynamic>?;
          return FreelancePublicProfileScreen(
            organizationId: state.pathParameters['id'] ?? '',
            displayName: extras?['display_name'] as String?,
          );
        },
      ),
      GoRoute(
        path: '/referrers/:id',
        builder: (context, state) {
          final extras = state.extra as Map<String, dynamic>?;
          return ReferrerPublicProfileScreen(
            organizationId: state.pathParameters['id'] ?? '',
            displayName: extras?['display_name'] as String?,
          );
        },
      ),

      // --- Search route (accessible without bottom nav) ---
      GoRoute(
        path: '/search/:type',
        builder: (context, state) => SearchScreen(
          type: state.pathParameters['type'] ?? 'freelancer',
        ),
      ),

      // --- Chat route (full-screen, no bottom nav) ---
      GoRoute(
        path: '/chat/:id',
        builder: (context, state) => ChatScreen(
          conversationId: state.pathParameters['id'] ?? '',
        ),
      ),

      // --- New chat route (lazy conversation — no conversation until first message) ---
      // The path param is an organization id: public profile URLs
      // surface the org, and the messaging backend resolves the Owner
      // user id internally.
      GoRoute(
        path: '${RoutePaths.newChat}/:recipientOrgId',
        builder: (context, state) {
          final extras = state.extra as Map<String, String>? ?? {};
          return NewChatScreen(
            recipientOrgId: state.pathParameters['recipientOrgId'] ?? '',
            recipientName: extras['name'] ?? '',
          );
        },
      ),

      // --- Candidate detail (full-screen, no bottom nav) ---
      GoRoute(
        path: RoutePaths.candidateDetail,
        builder: (context, state) {
          final extras = state.extra as Map<String, dynamic>;
          return CandidateDetailScreen(
            item: extras['item'] as ApplicationWithProfile,
            jobId: extras['jobId'] as String,
            candidates: extras['candidates'] as List<ApplicationWithProfile>?,
            candidateIndex: extras['candidateIndex'] as int?,
          );
        },
      ),

      // --- Proposal creation / modification (full-screen, no bottom nav) ---
      GoRoute(
        path: RoutePaths.projectsNew,
        builder: (context, state) {
          final extras = state.extra as Map<String, dynamic>?;
          return CreateProposalScreen(
            recipientId: extras?['recipientId'] as String? ?? '',
            conversationId:
                extras?['conversationId'] as String? ?? '',
            recipientName:
                extras?['recipientName'] as String? ?? '',
            existingProposal:
                extras?['existingProposal'] as ProposalEntity?,
          );
        },
      ),

      // --- Payment simulation (full-screen, no bottom nav) ---
      GoRoute(
        path: '/projects/pay/:id',
        builder: (context, state) => PaymentSimulationScreen(
          proposalId: state.pathParameters['id'] ?? '',
        ),
      ),

      // --- Proposal detail (full-screen, no bottom nav) ---
      GoRoute(
        path: '/projects/detail/:id',
        builder: (context, state) => ProposalDetailScreen(
          proposalId: state.pathParameters['id'] ?? '',
        ),
      ),

      // --- Open dispute (full-screen, no bottom nav) ---
      GoRoute(
        path: RoutePaths.disputeOpen,
        builder: (context, state) {
          final extras = state.extra as Map<String, dynamic>? ?? const {};
          return OpenDisputeScreen(
            proposalId: extras['proposalId'] as String? ?? '',
            proposalAmount: extras['proposalAmount'] as int? ?? 0,
            userRole: extras['userRole'] as String? ?? 'client',
          );
        },
      ),

      // --- Counter-propose dispute (full-screen, no bottom nav) ---
      GoRoute(
        path: RoutePaths.disputeCounter,
        builder: (context, state) {
          final extras = state.extra as Map<String, dynamic>? ?? const {};
          return CounterProposeScreen(
            disputeId: extras['disputeId'] as String? ?? '',
            proposalAmount: extras['proposalAmount'] as int? ?? 0,
          );
        },
      ),

      // --- Referral feature (full-screen, no bottom nav) ---
      // /referrals          → apporteur dashboard
      // /referrals/new      → creation form
      // /referrals/:id      → detail (adapts to viewer role)
      GoRoute(
        path: RoutePaths.referralsDashboard,
        builder: (context, state) => const ReferralDashboardScreen(),
      ),
      GoRoute(
        path: RoutePaths.referralCreate,
        builder: (context, state) => const ReferralCreationScreen(),
      ),
      GoRoute(
        path: '/referrals/:id',
        builder: (context, state) => ReferralDetailScreen(
          referralId: state.pathParameters['id'] ?? '',
        ),
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
            builder: (context, state) => const MessagingScreen(),
          ),
          GoRoute(
            path: RoutePaths.missions,
            builder: (context, state) => const ProjectsListScreen(),
          ),
          GoRoute(
            path: RoutePaths.jobs,
            builder: (context, state) => const JobsScreen(),
          ),
          GoRoute(
            path: RoutePaths.jobsCreate,
            builder: (context, state) => const CreateJobScreen(),
          ),
          GoRoute(
            path: RoutePaths.opportunities,
            builder: (context, state) => const OpportunitiesScreen(),
          ),
          GoRoute(
            path: RoutePaths.opportunityDetail,
            builder: (context, state) => OpportunityDetailScreen(jobId: state.extra as String),
          ),
          GoRoute(
            path: RoutePaths.myApplications,
            builder: (context, state) => const MyApplicationsScreen(),
          ),
          GoRoute(
            path: RoutePaths.jobCandidates,
            builder: (context, state) => CandidatesScreen(jobId: state.extra as String),
          ),
          GoRoute(
            path: RoutePaths.jobDetail,
            builder: (context, state) => JobDetailScreen(jobId: state.extra as String),
          ),
          GoRoute(
            path: RoutePaths.jobEdit,
            builder: (context, state) => CreateJobScreen(jobId: state.extra as String),
          ),
          GoRoute(
            path: RoutePaths.profile,
            // Dispatches based on org type: provider_personal users
            // see the new FreelanceProfileScreen; agency users still
            // see the legacy ProfileScreen (the legacy /api/v1/profile
            // endpoint is agency-only now and the refactor is tracked
            // as a separate follow-up).
            builder: (context, state) => const _ProfileDispatcher(),
          ),
          GoRoute(
            path: RoutePaths.referralProfile,
            builder: (context, state) => const ReferrerProfileScreen(),
          ),
          GoRoute(
            path: RoutePaths.paymentInfo,
            builder: (context, state) => const PaymentInfoScreen(),
          ),
          GoRoute(
            path: RoutePaths.wallet,
            builder: (context, state) => const WalletScreen(),
          ),
          GoRoute(
            path: RoutePaths.notifications,
            builder: (context, state) => const NotificationScreen(),
          ),
          GoRoute(
            path: RoutePaths.team,
            builder: (context, state) => const TeamScreen(),
          ),
        ],
      ),
    ],
  );
});

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
  bool _fcmInitialized = false;
  StreamSubscription<Map<String, dynamic>>? _notifSub;

  @override
  void initState() {
    super.initState();
    // Listen to WS notification events and show a SnackBar toast
    final wsSvc = ref.read(messagingWsServiceProvider);
    _notifSub = wsSvc.events.listen((event) {
      if (event['type'] == 'notification' && mounted) {
        final payload = event['payload'];
        if (payload is Map && payload['title'] != null) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    payload['title'] as String,
                    style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14),
                  ),
                  if (payload['body'] != null && (payload['body'] as String).isNotEmpty)
                    Text(
                      payload['body'] as String,
                      style: const TextStyle(fontSize: 12),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                ],
              ),
              behavior: SnackBarBehavior.floating,
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
              margin: const EdgeInsets.fromLTRB(16, 0, 16, 16),
              duration: const Duration(seconds: 3),
            ),
          );
        }
      }
    });
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
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final totalUnread = ref.watch(totalUnreadProvider);

    // Initialize FCM push notifications once after auth
    if (!_fcmInitialized) {
      _fcmInitialized = true;
      Future.microtask(() => FCMService.initialize(ref));
    }

    return Scaffold(
      key: DashboardShell.scaffoldKey,
      drawer: const AppDrawer(),
      body: Column(
        children: [
          const KYCBanner(),
          Expanded(child: widget.child),
        ],
      ),
      bottomNavigationBar: Container(
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
          selectedIndex: _currentIndex(context),
          destinations: [
            NavigationDestination(
              icon: const Icon(Icons.dashboard_outlined),
              selectedIcon: const Icon(Icons.dashboard),
              label: l10n.home,
            ),
            NavigationDestination(
              icon: totalUnread > 0
                  ? Badge(
                      label: Text(
                        totalUnread > 99 ? '99+' : '$totalUnread',
                        style: const TextStyle(
                          fontSize: 10,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      backgroundColor: const Color(0xFFF43F5E),
                      child: const Icon(Icons.chat_outlined),
                    )
                  : const Icon(Icons.chat_outlined),
              selectedIcon: totalUnread > 0
                  ? Badge(
                      label: Text(
                        totalUnread > 99 ? '99+' : '$totalUnread',
                        style: const TextStyle(
                          fontSize: 10,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      backgroundColor: const Color(0xFFF43F5E),
                      child: const Icon(Icons.chat),
                    )
                  : const Icon(Icons.chat),
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
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Profile dispatcher — selects freelance vs legacy based on org type
// ---------------------------------------------------------------------------

/// Thin wrapper that picks the right profile screen for the signed-in
/// operator. `provider_personal` users get the new split-profile
/// [FreelanceProfileScreen]; any other org type (currently just
/// `agency`) keeps rendering the legacy [ProfileScreen] until the
/// agency refactor ships.
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
// Welcome banner — gradient rose to purple
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
            Color(0xFFF43F5E), // rose-500
            Color(0xFF8B5CF6), // violet-500
          ],
        ),
        boxShadow: [
          BoxShadow(
            color: const Color(0xFFF43F5E).withValues(alpha: 0.3),
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
// Agency dashboard
// ---------------------------------------------------------------------------

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
        leading: IconButton(
          icon: const Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: const Text('Marketplace'),
        actions: [
          NotificationBadge(
            onTap: () => GoRouter.of(context).go(RoutePaths.notifications),
          ),
        ],
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _WelcomeBanner(
                displayName: displayName,
                subtitle: l10n.roleAgencyDesc,
              ),
              const SizedBox(height: 24),
              _SearchActions(actions: [
                _SearchAction(
                  label: l10n.findFreelancers,
                  icon: Icons.person_search,
                  type: 'freelancer',
                  color: const Color(0xFFF43F5E),
                ),
                _SearchAction(
                  label: l10n.findReferrers,
                  icon: Icons.handshake_outlined,
                  type: 'referrer',
                  color: const Color(0xFFF59E0B),
                ),
              ],),
              const SizedBox(height: 24),
              _buildStatCards(context, l10n),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildStatCards(BuildContext context, AppLocalizations l10n) {
    return Column(
      children: [
        _StatCard(
          icon: Icons.work_outline,
          title: l10n.activeMissions,
          value: '0',
          subtitle: l10n.activeContracts,
          color: const Color(0xFF2563EB),
        ),
        const SizedBox(height: 12),
        _StatCard(
          icon: Icons.chat_outlined,
          title: l10n.unreadMessages,
          value: '0',
          subtitle: l10n.conversations,
          color: const Color(0xFF8B5CF6),
        ),
        const SizedBox(height: 12),
        _StatCard(
          icon: Icons.trending_up,
          title: l10n.monthlyRevenue,
          value: '0 EUR',
          subtitle: l10n.thisMonth,
          color: const Color(0xFF22C55E),
        ),
      ],
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
    final l10n = AppLocalizations.of(context)!;
    final displayName =
        authState.user?['display_name'] as String? ?? 'Enterprise';

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: const Text('Marketplace'),
        actions: [
          NotificationBadge(
            onTap: () => GoRouter.of(context).go(RoutePaths.notifications),
          ),
        ],
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _WelcomeBanner(
                displayName: displayName,
                subtitle: l10n.roleEnterpriseDesc,
              ),
              const SizedBox(height: 24),
              _SearchActions(actions: [
                _SearchAction(
                  label: l10n.findFreelancers,
                  icon: Icons.person_search,
                  type: 'freelancer',
                  color: const Color(0xFFF43F5E),
                ),
                _SearchAction(
                  label: l10n.findAgencies,
                  icon: Icons.business,
                  type: 'agency',
                  color: const Color(0xFF2563EB),
                ),
                _SearchAction(
                  label: l10n.findReferrers,
                  icon: Icons.handshake_outlined,
                  type: 'referrer',
                  color: const Color(0xFFF59E0B),
                ),
              ],),
              const SizedBox(height: 24),
              _buildStatCards(context, l10n),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildStatCards(BuildContext context, AppLocalizations l10n) {
    return Column(
      children: [
        _StatCard(
          icon: Icons.folder_open_outlined,
          title: l10n.activeProjects,
          value: '0',
          subtitle: l10n.activeProjects,
          color: const Color(0xFF2563EB),
        ),
        const SizedBox(height: 12),
        _StatCard(
          icon: Icons.chat_outlined,
          title: l10n.unreadMessages,
          value: '0',
          subtitle: l10n.conversations,
          color: const Color(0xFF8B5CF6),
        ),
        const SizedBox(height: 12),
        _StatCard(
          icon: Icons.account_balance_wallet_outlined,
          title: l10n.totalBudget,
          value: '0 EUR',
          subtitle: l10n.spentThisMonth,
          color: const Color(0xFF22C55E),
        ),
      ],
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
    final l10n = AppLocalizations.of(context)!;
    final displayName =
        authState.user?['first_name'] as String? ??
        authState.user?['display_name'] as String? ??
        'Provider';

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: const Text('Marketplace'),
        actions: [
          NotificationBadge(
            onTap: () => GoRouter.of(context).go(RoutePaths.notifications),
          ),
        ],
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _WelcomeBanner(
                displayName: displayName,
                subtitle: l10n.roleFreelanceDesc,
              ),
              const SizedBox(height: 16),

              // Switch to referrer mode
              SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  onPressed: () => context.go(RoutePaths.dashboardReferrer),
                  icon: const Icon(Icons.swap_horiz),
                  label: Text(l10n.businessReferrerMode),
                ),
              ),
              const SizedBox(height: 24),
              _buildStatCards(context, l10n),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildStatCards(BuildContext context, AppLocalizations l10n) {
    return Column(
      children: [
        _StatCard(
          icon: Icons.work_outline,
          title: l10n.activeMissions,
          value: '0',
          subtitle: l10n.activeContracts,
          color: const Color(0xFF2563EB),
        ),
        const SizedBox(height: 12),
        _StatCard(
          icon: Icons.chat_outlined,
          title: l10n.unreadMessages,
          value: '0',
          subtitle: l10n.conversations,
          color: const Color(0xFF8B5CF6),
        ),
        const SizedBox(height: 12),
        _StatCard(
          icon: Icons.trending_up,
          title: l10n.monthlyRevenue,
          value: '0 EUR',
          subtitle: l10n.thisMonth,
          color: const Color(0xFF22C55E),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Search action data class
// ---------------------------------------------------------------------------

/// Describes a single search action button on the dashboard.
class _SearchAction {
  _SearchAction({
    required this.label,
    required this.icon,
    required this.type,
    required this.color,
  });

  final String label;
  final IconData icon;
  final String type;
  final Color color;
}

// ---------------------------------------------------------------------------
// Search actions row — quick-access search buttons on dashboards
// ---------------------------------------------------------------------------

class _SearchActions extends StatelessWidget {
  _SearchActions({required this.actions});

  final List<_SearchAction> actions;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: actions
          .map((action) => _SearchActionChip(action: action))
          .toList(),
    );
  }
}

class _SearchActionChip extends StatelessWidget {
  const _SearchActionChip({required this.action});

  final _SearchAction action;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return ActionChip(
      avatar: Icon(action.icon, size: 18, color: action.color),
      label: Text(
        action.label,
        style: TextStyle(
          color: theme.colorScheme.onSurface,
          fontWeight: FontWeight.w500,
          fontSize: 13,
        ),
      ),
      backgroundColor: action.color.withValues(alpha: 0.08),
      side: BorderSide(color: action.color.withValues(alpha: 0.2)),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      onPressed: () => GoRouter.of(context).push('/search/${action.type}'),
    );
  }
}

// ---------------------------------------------------------------------------
// Shared stat card widget — premium design
// ---------------------------------------------------------------------------

/// A stat card matching the web premium dashboard design.
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

// Placeholder screens removed -- replaced by ProjectsListScreen.
