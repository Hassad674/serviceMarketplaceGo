import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/router/app_router.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/profile_completion/domain/entities/profile_completion_report.dart';
import 'package:marketplace_mobile/features/profile_completion/domain/repositories/profile_completion_repository.dart';
import 'package:marketplace_mobile/features/profile_completion/presentation/providers/profile_completion_providers.dart';
import 'package:marketplace_mobile/features/profile_completion/presentation/widgets/profile_completion_bar.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

// =============================================================================
// Repository doubles
// =============================================================================

class _FakeRepo implements ProfileCompletionRepository {
  _FakeRepo(this._report);
  final ProfileCompletionReport _report;
  @override
  Future<ProfileCompletionReport> getMy({String? persona}) async => _report;
}

class _ThrowingRepo implements ProfileCompletionRepository {
  @override
  Future<ProfileCompletionReport> getMy({String? persona}) async =>
      throw Exception('boom');
}

// =============================================================================
// Auth doubles — bypass real ApiClient + SecureStorage so the widget
// test does not have to spin up platform plugins
// =============================================================================

class _FakeStorage extends Fake implements SecureStorageService {
  @override
  Future<String?> getAccessToken() async => null;

  @override
  Future<String?> getRefreshToken() async => null;

  @override
  Future<bool> hasTokens() async => false;

  @override
  Future<void> saveTokens(String access, String refresh) async {}

  @override
  Future<void> clearTokens() async {}

  @override
  Future<void> clearAll() async {}

  @override
  Future<void> saveUser(Map<String, dynamic> user) async {}

  @override
  Future<Map<String, dynamic>?> getUser() async => null;
}

class _FakeApiClient extends ApiClient {
  _FakeApiClient() : super(storage: _FakeStorage());
}

/// Builds a real [AuthNotifier] with fake deps then overrides its
/// state synchronously via the protected setter — same trick as
/// `pricing_screen_test.dart`. Avoids the async session-restore in
/// the parent constructor that would otherwise overwrite our seeded
/// state on the first frame.
AuthNotifier _buildAuthNotifier({String? role, String? orgType}) {
  final notifier = AuthNotifier(
    apiClient: _FakeApiClient(),
    storage: _FakeStorage(),
  );
  // ignore: invalid_use_of_protected_member
  notifier.state = AuthState(
    status:
        role == null ? AuthStatus.unauthenticated : AuthStatus.authenticated,
    user: role != null ? {'role': role} : null,
    organization: orgType != null ? {'type': orgType} : null,
  );
  return notifier;
}

// =============================================================================
// Wrap helper
// =============================================================================

Widget _wrap({
  required ProfileCompletionRepository repo,
  bool hideWhenComplete = false,
  String? role,
  String? orgType,
  Map<String, String>? capturedRoutes,
}) {
  // Minimal go_router with two routes (/profile + /client-profile) so
  // the bar's `context.go(...)` lands somewhere observable. The body
  // of each route writes its path into [capturedRoutes] so the test
  // can assert which destination was reached.
  final router = GoRouter(
    initialLocation: '/start',
    routes: [
      GoRoute(
        path: '/start',
        // Wrap the bar in a Scaffold so the embedded InkWell has the
        // Material ancestor it needs (production mounts the bar
        // inside a Column under a Scaffold; the test mirrors that).
        builder: (_, __) => Scaffold(
          body: Padding(
            padding: const EdgeInsets.all(16),
            child: ProfileCompletionBar(hideWhenComplete: hideWhenComplete),
          ),
        ),
      ),
      GoRoute(
        path: RoutePaths.profile,
        builder: (_, __) {
          capturedRoutes?['target'] = RoutePaths.profile;
          return const Scaffold(body: Text('PROFILE'));
        },
      ),
      GoRoute(
        path: RoutePaths.clientProfile,
        builder: (_, __) {
          capturedRoutes?['target'] = RoutePaths.clientProfile;
          return const Scaffold(body: Text('CLIENT_PROFILE'));
        },
      ),
    ],
  );

  return ProviderScope(
    overrides: [
      profileCompletionRepositoryProvider.overrideWithValue(repo),
      secureStorageProvider.overrideWithValue(_FakeStorage()),
      apiClientProvider.overrideWithValue(_FakeApiClient()),
      authProvider.overrideWith(
        (ref) => _buildAuthNotifier(role: role, orgType: orgType),
      ),
    ],
    child: MaterialApp.router(
      theme: AppTheme.light,
      routerConfig: router,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      locale: const Locale('fr'),
    ),
  );
}

const _baseReport = ProfileCompletionReport(
  role: 'provider',
  persona: 'freelance',
  percent: 50,
  totalSections: 11,
  filledSections: 5,
  sections: [
    ProfileCompletionSection(
      key: 'title',
      filled: true,
      labelKey: 'profile.completion.section.title',
      completionPath: '/dashboard/profile/edit',
    ),
    ProfileCompletionSection(
      key: 'about',
      filled: false,
      labelKey: 'profile.completion.section.about',
      completionPath: '/dashboard/profile/edit',
    ),
  ],
);

void main() {
  testWidgets('renders the percent and filled/total subtitle',
      (tester) async {
    await tester.pumpWidget(_wrap(repo: _FakeRepo(_baseReport)));
    await tester.pumpAndSettle();

    expect(find.text('Profil rempli à 50%'), findsOneWidget);
    expect(find.text('5/11 sections complétées'), findsOneWidget);
  });

  testWidgets(
      'navigates to /profile for provider users on tap (no bottom sheet)',
      (tester) async {
    final captured = <String, String>{};
    await tester.pumpWidget(
      _wrap(
        repo: _FakeRepo(_baseReport),
        role: 'provider',
        orgType: 'provider_personal',
        capturedRoutes: captured,
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Profil rempli à 50%'));
    await tester.pumpAndSettle();

    // Bottom sheet must NOT have opened — bar is now a navigation
    // affordance, not a popup launcher.
    expect(find.text('Profil complété à 50%'), findsNothing);
    // Landed on /profile — the freelance/agency dispatcher route.
    expect(find.text('PROFILE'), findsOneWidget);
    expect(captured['target'], RoutePaths.profile);
  });

  testWidgets('navigates to /client-profile for enterprise users on tap',
      (tester) async {
    final captured = <String, String>{};
    await tester.pumpWidget(
      _wrap(
        repo: _FakeRepo(_baseReport),
        role: 'enterprise',
        orgType: 'enterprise',
        capturedRoutes: captured,
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Profil rempli à 50%'));
    await tester.pumpAndSettle();

    expect(find.text('CLIENT_PROFILE'), findsOneWidget);
    expect(captured['target'], RoutePaths.clientProfile);
  });

  testWidgets('hides itself at 100 percent when hideWhenComplete is true',
      (tester) async {
    const completed = ProfileCompletionReport(
      role: 'provider',
      persona: 'freelance',
      percent: 100,
      totalSections: 5,
      filledSections: 5,
      sections: [],
    );
    await tester.pumpWidget(
      _wrap(repo: _FakeRepo(completed), hideWhenComplete: true),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('Profil rempli'), findsNothing);
  });

  testWidgets('renders nothing on repository error (silent fallback)',
      (tester) async {
    await tester.pumpWidget(_wrap(repo: _ThrowingRepo()));
    await tester.pumpAndSettle();

    expect(find.textContaining('Profil rempli'), findsNothing);
  });
}
