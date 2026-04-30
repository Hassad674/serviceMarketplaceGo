import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:marketplace_mobile/core/router/app_router.dart';
import 'package:marketplace_mobile/core/router/routes/auth_routes.dart';
import 'package:marketplace_mobile/core/router/routes/dashboard_routes.dart';
import 'package:marketplace_mobile/core/router/routes/payment_routes.dart';
import 'package:marketplace_mobile/core/router/routes/profile_routes.dart';
import 'package:marketplace_mobile/core/router/routes/team_routes.dart';

/// Helper that returns the path string of every [GoRoute] returned by
/// a route-list builder.
List<String> _paths(List<RouteBase> routes) =>
    routes.whereType<GoRoute>().map((r) => r.path).toList();

void main() {
  group('buildAuthRoutes', () {
    test('returns 5 routes covering login + 4 register screens', () {
      final paths = _paths(buildAuthRoutes());
      expect(paths, [
        RoutePaths.login,
        RoutePaths.register,
        RoutePaths.registerAgency,
        RoutePaths.registerProvider,
        RoutePaths.registerEnterprise,
      ]);
    });
  });

  group('buildProfileRoutes', () {
    test('returns the public profile + chat + search routes', () {
      final paths = _paths(buildProfileRoutes());
      expect(paths, contains('/profiles/:id'));
      expect(paths, contains('/freelancers/:id'));
      expect(paths, contains('/referrers/:id'));
      expect(paths, contains('/clients/:id'));
      expect(paths, contains('/search/:type'));
      expect(paths, contains('/chat/:id'));
      expect(paths, contains('${RoutePaths.newChat}/:recipientOrgId'));
    });
  });

  group('buildPaymentRoutes', () {
    test('exposes proposal, dispute, referral, subscription, invoicing', () {
      final paths = _paths(buildPaymentRoutes());
      expect(paths, contains(RoutePaths.candidateDetail));
      expect(paths, contains(RoutePaths.projectsNew));
      expect(paths, contains('/projects/pay/:id'));
      expect(paths, contains('/projects/detail/:id'));
      expect(paths, contains(RoutePaths.disputeOpen));
      expect(paths, contains(RoutePaths.disputeCounter));
      expect(paths, contains(RoutePaths.referralsDashboard));
      expect(paths, contains(RoutePaths.referralCreate));
      expect(paths, contains('/referrals/:id'));
      expect(paths, contains(RoutePaths.pricing));
      expect(paths, contains(RoutePaths.billingSuccess));
      expect(paths, contains(RoutePaths.billingCancel));
      expect(paths, contains(RoutePaths.checkoutWebview));
      expect(paths, contains(RoutePaths.billingProfile));
      expect(paths, contains(RoutePaths.invoices));
    });
  });

  group('buildDashboardShellRoutes', () {
    test('exposes dashboard + messaging + missions + jobs', () {
      final paths = _paths(buildDashboardShellRoutes());
      expect(paths, contains(RoutePaths.dashboard));
      expect(paths, contains(RoutePaths.dashboardReferrer));
      expect(paths, contains(RoutePaths.messaging));
      expect(paths, contains(RoutePaths.missions));
      expect(paths, contains(RoutePaths.jobs));
      expect(paths, contains(RoutePaths.jobsCreate));
      expect(paths, contains(RoutePaths.opportunities));
      expect(paths, contains(RoutePaths.opportunityDetail));
      expect(paths, contains(RoutePaths.myApplications));
      expect(paths, contains(RoutePaths.jobCandidates));
      expect(paths, contains(RoutePaths.jobDetail));
      expect(paths, contains(RoutePaths.jobEdit));
    });
  });

  group('buildTeamShellRoutes', () {
    test('exposes profile + wallet + payment-info + notifications + team',
        () {
      final paths = _paths(buildTeamShellRoutes());
      expect(paths, contains(RoutePaths.profile));
      expect(paths, contains(RoutePaths.clientProfile));
      expect(paths, contains(RoutePaths.referralProfile));
      expect(paths, contains(RoutePaths.paymentInfo));
      expect(paths, contains(RoutePaths.wallet));
      expect(paths, contains(RoutePaths.notifications));
      expect(paths, contains(RoutePaths.team));
    });
  });

  group('RoutePaths consistency', () {
    test('login + register paths are stable strings', () {
      // FCM regression suite imports these constants verbatim.
      expect(RoutePaths.login, '/login');
      expect(RoutePaths.register, '/register');
      expect(RoutePaths.dashboard, '/dashboard');
      expect(RoutePaths.notifications, '/notifications');
      expect(RoutePaths.profile, '/profile');
      expect(RoutePaths.proposalDetail, '/projects/detail');
    });
  });
}
