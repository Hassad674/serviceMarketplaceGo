import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/job/domain/entities/job_application_entity.dart';
import 'package:marketplace_mobile/features/job/domain/entities/job_entity.dart';
import 'package:marketplace_mobile/features/job/presentation/providers/job_provider.dart';
import 'package:marketplace_mobile/features/job/presentation/screens/opportunities_screen.dart';

import '../../widget/payment_info/test_helpers.dart';

// =============================================================================
// Fake implementations
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

class _FakeAuthNotifier extends AuthNotifier {
  _FakeAuthNotifier()
      : super(apiClient: _FakeApiClient(), storage: _FakeStorage());

  @override
  AuthState get state => const AuthState(
        status: AuthStatus.authenticated,
        user: {
          'id': 'current-user',
          'email': 'provider@example.com',
          'role': 'provider',
        },
      );
}

// =============================================================================
// Override builder
// =============================================================================

List<Override> _buildOverrides({
  required int creditCount,
  List<JobEntity> openJobs = const [],
  List<ApplicationWithJob> myApplications = const [],
}) {
  return [
    secureStorageProvider.overrideWithValue(_FakeStorage()),
    apiClientProvider.overrideWithValue(_FakeApiClient()),
    authProvider.overrideWith((ref) => _FakeAuthNotifier()),
    creditsProvider.overrideWith((ref) => Future.value(creditCount)),
    openJobsProvider.overrideWith((ref) => Future.value(openJobs)),
    myApplicationsProvider
        .overrideWith((ref) => Future.value(myApplications)),
  ];
}

// =============================================================================
// Test data
// =============================================================================

const _sampleJobs = [
  JobEntity(
    id: 'job-1',
    creatorId: 'other-user',
    title: 'Flutter Developer Needed',
    description: 'Build a mobile app',
    skills: ['Flutter', 'Dart'],
    applicantType: 'all',
    budgetType: 'one_shot',
    minBudget: 1000,
    maxBudget: 5000,
    status: 'open',
    createdAt: '2026-04-01T10:00:00Z',
    updatedAt: '2026-04-01T10:00:00Z',
  ),
  JobEntity(
    id: 'job-2',
    creatorId: 'other-user-2',
    title: 'Go Backend Engineer',
    description: 'Build microservices',
    skills: ['Go', 'PostgreSQL'],
    applicantType: 'freelancers',
    budgetType: 'recurring',
    minBudget: 3000,
    maxBudget: 6000,
    status: 'open',
    createdAt: '2026-04-02T10:00:00Z',
    updatedAt: '2026-04-02T10:00:00Z',
  ),
];

// =============================================================================
// Tests — _CreditsHeader behavior within OpportunitiesScreen
// =============================================================================

void main() {
  group('Credits display in OpportunitiesScreen', () {
    testWidgets(
      'shows positive credit count',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: _buildOverrides(
              creditCount: 10,
              openJobs: _sampleJobs,
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Should display "10 credits remaining"
        expect(find.textContaining('10'), findsWidgets);
        expect(find.textContaining('credits remaining'), findsOneWidget);
      },
    );

    testWidgets(
      'shows single credit count correctly',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: _buildOverrides(
              creditCount: 1,
              openJobs: _sampleJobs,
            ),
          ),
        );

        await tester.pumpAndSettle();

        expect(find.textContaining('1'), findsWidgets);
        expect(find.textContaining('credits remaining'), findsOneWidget);
      },
    );

    testWidgets(
      'shows warning when credits are zero',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: _buildOverrides(
              creditCount: 0,
              openJobs: _sampleJobs,
            ),
          ),
        );

        await tester.pumpAndSettle();

        // The "no credits left" warning banner should appear
        expect(
          find.text('You have no application credits left'),
          findsOneWidget,
        );

        // Warning icon should be visible
        expect(
          find.byIcon(Icons.warning_amber_rounded),
          findsOneWidget,
        );
      },
    );

    testWidgets(
      'no warning banner when credits are positive',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: _buildOverrides(
              creditCount: 5,
              openJobs: _sampleJobs,
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Warning text should NOT appear
        expect(
          find.text('You have no application credits left'),
          findsNothing,
        );

        // Warning icon should NOT appear
        expect(
          find.byIcon(Icons.warning_amber_rounded),
          findsNothing,
        );
      },
    );

    testWidgets(
      'credits header shows help button',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: _buildOverrides(
              creditCount: 7,
              openJobs: _sampleJobs,
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Help icon should be present
        expect(find.byIcon(Icons.help_outline), findsOneWidget);
      },
    );

    testWidgets(
      'credits header shows ticket icon',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: _buildOverrides(
              creditCount: 10,
              openJobs: _sampleJobs,
            ),
          ),
        );

        await tester.pumpAndSettle();

        expect(
          find.byIcon(Icons.confirmation_number_outlined),
          findsOneWidget,
        );
      },
    );

    testWidgets(
      'zero credits header uses red color scheme',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: _buildOverrides(
              creditCount: 0,
              openJobs: _sampleJobs,
            ),
          ),
        );

        await tester.pumpAndSettle();

        // The icon should be red (0xFFEF4444) when no credits
        final iconFinder = find.byIcon(Icons.confirmation_number_outlined);
        expect(iconFinder, findsOneWidget);

        final icon = tester.widget<Icon>(iconFinder);
        expect(icon.color, const Color(0xFFEF4444));
      },
    );

    testWidgets(
      'positive credits header uses rose color scheme',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: _buildOverrides(
              creditCount: 5,
              openJobs: _sampleJobs,
            ),
          ),
        );

        await tester.pumpAndSettle();

        final iconFinder = find.byIcon(Icons.confirmation_number_outlined);
        expect(iconFinder, findsOneWidget);

        final icon = tester.widget<Icon>(iconFinder);
        expect(icon.color, const Color(0xFFF43F5E));
      },
    );

    testWidgets(
      'tapping help button opens credits explanation modal',
      (WidgetTester tester) async {
        // Use a larger surface to avoid overflow in the bottom sheet modal
        tester.view.physicalSize = const Size(800, 1600);
        tester.view.devicePixelRatio = 1.0;
        addTearDown(tester.view.resetPhysicalSize);
        addTearDown(tester.view.resetDevicePixelRatio);

        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: _buildOverrides(
              creditCount: 5,
              openJobs: _sampleJobs,
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Tap the help icon
        await tester.tap(find.byIcon(Icons.help_outline));
        await tester.pumpAndSettle();

        // The modal bottom sheet should appear with explanation text
        expect(
          find.text('How do credits work?'),
          findsWidgets, // title in header + modal
        );
      },
    );

    testWidgets(
      'shows empty state when no open jobs and credits are positive',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: _buildOverrides(
              creditCount: 10,
              openJobs: const [],
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Credits header should still be present
        expect(find.textContaining('credits remaining'), findsOneWidget);

        // Empty state icon
        expect(find.byIcon(Icons.work_off_outlined), findsOneWidget);
      },
    );

    testWidgets(
      'shows empty state when no open jobs and credits are zero',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: _buildOverrides(
              creditCount: 0,
              openJobs: const [],
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Both the zero credits warning and empty jobs state should show
        expect(
          find.text('You have no application credits left'),
          findsOneWidget,
        );
        expect(find.byIcon(Icons.work_off_outlined), findsOneWidget);
      },
    );

    testWidgets(
      'job cards are displayed alongside credits header',
      (WidgetTester tester) async {
        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: _buildOverrides(
              creditCount: 8,
              openJobs: _sampleJobs,
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Credits header visible
        expect(find.textContaining('credits remaining'), findsOneWidget);

        // Job titles visible
        expect(find.text('Flutter Developer Needed'), findsOneWidget);
        expect(find.text('Go Backend Engineer'), findsOneWidget);
      },
    );

    testWidgets(
      'credits display updates when provider is invalidated',
      (WidgetTester tester) async {
        var currentCredits = 5;

        await tester.pumpWidget(
          buildTestableScreen(
            const OpportunitiesScreen(),
            overrides: [
              secureStorageProvider.overrideWithValue(_FakeStorage()),
              apiClientProvider.overrideWithValue(_FakeApiClient()),
              authProvider.overrideWith((ref) => _FakeAuthNotifier()),
              creditsProvider
                  .overrideWith((ref) => Future.value(currentCredits)),
              openJobsProvider
                  .overrideWith((ref) => Future.value(_sampleJobs)),
              myApplicationsProvider.overrideWith(
                (ref) => Future.value(<ApplicationWithJob>[]),
              ),
            ],
          ),
        );

        await tester.pumpAndSettle();

        // Should show 5 credits
        expect(find.textContaining('5'), findsWidgets);
        expect(find.textContaining('credits remaining'), findsOneWidget);
      },
    );
  });
}
