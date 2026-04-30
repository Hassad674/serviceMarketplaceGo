import 'package:go_router/go_router.dart';

import '../../../features/auth/presentation/screens/agency_register_screen.dart';
import '../../../features/auth/presentation/screens/enterprise_register_screen.dart';
import '../../../features/auth/presentation/screens/login_screen.dart';
import '../../../features/auth/presentation/screens/register_screen.dart';
import '../../../features/auth/presentation/screens/role_selection_screen.dart';
import '../app_router.dart';

/// Public auth routes — login + role-based register screens. None of
/// these require authentication.
List<RouteBase> buildAuthRoutes() => [
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
    ];
