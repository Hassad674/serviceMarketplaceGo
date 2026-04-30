import 'package:go_router/go_router.dart';

import '../../../features/dispute/presentation/screens/counter_propose_screen.dart';
import '../../../features/dispute/presentation/screens/open_dispute_screen.dart';
import '../../../features/invoicing/presentation/screens/billing_profile_screen.dart';
import '../../../features/invoicing/presentation/screens/invoices_screen.dart';
import '../../../features/job/domain/entities/job_application_entity.dart';
import '../../../features/job/presentation/screens/candidate_detail_screen.dart';
import '../../../features/proposal/domain/entities/proposal_entity.dart';
import '../../../features/proposal/presentation/screens/create_proposal_screen.dart';
import '../../../features/proposal/presentation/screens/payment_simulation_screen.dart';
import '../../../features/proposal/presentation/screens/proposal_detail_screen.dart';
import '../../../features/referral/presentation/screens/referral_creation_screen.dart';
import '../../../features/referral/presentation/screens/referral_dashboard_screen.dart';
import '../../../features/referral/presentation/screens/referral_detail_screen.dart';
import '../../../features/subscription/presentation/screens/billing_cancel_screen.dart';
import '../../../features/subscription/presentation/screens/billing_success_screen.dart';
import '../../../features/subscription/presentation/screens/checkout_webview_screen.dart';
import '../../../features/subscription/presentation/screens/pricing_screen.dart';
import '../app_router.dart';

/// Full-screen routes for the proposal / dispute / referral / subscription /
/// invoicing flows — none of these are wrapped in the bottom navigation
/// shell. The candidate-detail screen also lives here because it is reached
/// from the candidates list with a non-trivial extras payload.
List<RouteBase> buildPaymentRoutes() => [
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
      GoRoute(
        path: RoutePaths.projectsNew,
        builder: (context, state) {
          final extras = state.extra as Map<String, dynamic>?;
          return CreateProposalScreen(
            recipientId: extras?['recipientId'] as String? ?? '',
            conversationId: extras?['conversationId'] as String? ?? '',
            recipientName: extras?['recipientName'] as String? ?? '',
            existingProposal: extras?['existingProposal'] as ProposalEntity?,
          );
        },
      ),
      GoRoute(
        path: '/projects/pay/:id',
        builder: (context, state) => PaymentSimulationScreen(
          proposalId: state.pathParameters['id'] ?? '',
        ),
      ),
      GoRoute(
        path: '/projects/detail/:id',
        builder: (context, state) => ProposalDetailScreen(
          proposalId: state.pathParameters['id'] ?? '',
        ),
      ),
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
      GoRoute(
        path: RoutePaths.pricing,
        builder: (context, state) => const PricingScreen(),
      ),
      GoRoute(
        path: RoutePaths.billingSuccess,
        builder: (context, state) => const BillingSuccessScreen(),
      ),
      GoRoute(
        path: RoutePaths.billingCancel,
        builder: (context, state) => const BillingCancelScreen(),
      ),
      GoRoute(
        path: RoutePaths.checkoutWebview,
        builder: (context, state) {
          final url = state.extra;
          if (url is! String || url.isEmpty) {
            return const BillingCancelScreen();
          }
          return CheckoutWebViewScreen(url: url);
        },
      ),
      GoRoute(
        path: RoutePaths.billingProfile,
        builder: (context, state) => BillingProfileScreen(
          returnTo: state.uri.queryParameters['return_to'],
        ),
      ),
      GoRoute(
        path: RoutePaths.invoices,
        builder: (context, state) => const InvoicesScreen(),
      ),
    ];
