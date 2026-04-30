import 'package:go_router/go_router.dart';

import '../../../features/dashboard/presentation/screens/referrer_dashboard_screen.dart';
import '../../../features/job/presentation/screens/candidates_screen.dart';
import '../../../features/job/presentation/screens/create_job_screen.dart';
import '../../../features/job/presentation/screens/job_detail_screen.dart';
import '../../../features/job/presentation/screens/jobs_screen.dart';
import '../../../features/job/presentation/screens/my_applications_screen.dart';
import '../../../features/job/presentation/screens/opportunities_screen.dart';
import '../../../features/job/presentation/screens/opportunity_detail_screen.dart';
import '../../../features/messaging/presentation/screens/messaging_screen.dart';
import '../../../features/proposal/presentation/screens/projects_list_screen.dart';
import '../app_router.dart';

/// Authenticated routes wrapped by the [DashboardShell] bottom-navigation
/// shell — dashboard, messaging, missions, jobs, opportunities. Each
/// route renders its full screen above the persistent bottom nav.
///
/// Profile, team, payment-info, wallet, and notifications also live in
/// the shell; they are kept in [team_routes.dart] for grouping.
List<RouteBase> buildDashboardShellRoutes() => [
      GoRoute(
        path: RoutePaths.dashboard,
        builder: (context, state) => const DashboardScreen(),
      ),
      GoRoute(
        path: RoutePaths.dashboardReferrer,
        builder: (context, state) => const ReferrerDashboardScreen(),
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
        builder: (context, state) =>
            OpportunityDetailScreen(jobId: state.extra as String),
      ),
      GoRoute(
        path: RoutePaths.myApplications,
        builder: (context, state) => const MyApplicationsScreen(),
      ),
      GoRoute(
        path: RoutePaths.jobCandidates,
        builder: (context, state) =>
            CandidatesScreen(jobId: state.extra as String),
      ),
      GoRoute(
        path: RoutePaths.jobDetail,
        builder: (context, state) =>
            JobDetailScreen(jobId: state.extra as String),
      ),
      GoRoute(
        path: RoutePaths.jobEdit,
        builder: (context, state) =>
            CreateJobScreen(jobId: state.extra as String),
      ),
    ];
