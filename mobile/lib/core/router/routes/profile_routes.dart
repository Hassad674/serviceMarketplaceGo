import 'package:go_router/go_router.dart';

import '../../../features/client_profile/presentation/screens/public_client_profile_screen.dart';
import '../../../features/freelance_profile/presentation/screens/freelance_public_profile_screen.dart';
import '../../../features/messaging/presentation/screens/chat_screen.dart';
import '../../../features/messaging/presentation/screens/new_chat_screen.dart';
import '../../../features/referrer_profile/presentation/screens/referrer_public_profile_screen.dart';
import '../../../features/search/presentation/screens/public_profile_screen.dart';
import '../../../features/search/presentation/screens/search_screen.dart';
import '../app_router.dart';

/// Public-facing profile and discovery routes — accessible without the
/// bottom navigation shell. The legacy `/profiles/:id` is still used by
/// agencies; the freelance/referrer split lives at `/freelancers/:id`
/// and `/referrers/:id`. Public client profiles at `/clients/:id`.
/// Search lives outside the shell because it is reachable from public
/// profile pages and chat headers.
List<RouteBase> buildProfileRoutes() => [
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
      GoRoute(
        path: '/clients/:id',
        builder: (context, state) {
          return PublicClientProfileScreen(
            organizationId: state.pathParameters['id'] ?? '',
          );
        },
      ),
      GoRoute(
        path: '/search/:type',
        builder: (context, state) => SearchScreen(
          type: state.pathParameters['type'] ?? 'freelancer',
        ),
      ),
      GoRoute(
        path: '/chat/:id',
        builder: (context, state) => ChatScreen(
          conversationId: state.pathParameters['id'] ?? '',
        ),
      ),
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
    ];
