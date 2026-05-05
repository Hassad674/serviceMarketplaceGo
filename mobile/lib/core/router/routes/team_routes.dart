import 'package:go_router/go_router.dart';

import '../../../features/account/presentation/screens/account_screen.dart';
import '../../../features/account/presentation/screens/cancel_deletion_screen.dart';
import '../../../features/account/presentation/screens/delete_account_screen.dart';
import '../../../features/client_profile/presentation/screens/client_profile_screen.dart';
import '../../../features/notification/presentation/screens/notification_screen.dart';
import '../../../features/payment_info/presentation/screens/payment_info_screen.dart';
import '../../../features/referrer_profile/presentation/screens/referrer_profile_screen.dart';
import '../../../features/team/presentation/screens/team_screen.dart';
import '../../../features/wallet/presentation/screens/wallet_screen.dart';
import '../app_router.dart';

/// Profile, team, wallet, payment-info, notifications, account — the
/// "membership" routes that sit inside the [DashboardShell]. The
/// `/profile` builder dispatches to the freelance vs legacy profile
/// screen based on the org type via [profileDispatcherBuilder].
List<RouteBase> buildTeamShellRoutes() => [
      GoRoute(
        path: RoutePaths.profile,
        builder: profileDispatcherBuilder,
      ),
      GoRoute(
        path: RoutePaths.clientProfile,
        builder: (context, state) => const ClientProfileScreen(),
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
      GoRoute(
        path: RoutePaths.account,
        builder: (context, state) => const AccountScreen(),
      ),
      GoRoute(
        path: RoutePaths.accountDelete,
        builder: (context, state) => const DeleteAccountScreen(),
      ),
      GoRoute(
        path: RoutePaths.accountCancelDeletion,
        builder: (context, state) => const CancelDeletionScreen(),
      ),
    ];
