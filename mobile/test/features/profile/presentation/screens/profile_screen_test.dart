import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/core/theme/theme_provider.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/profile/presentation/providers/profile_provider.dart';
import 'package:marketplace_mobile/features/profile/presentation/screens/profile_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

// =============================================================================
// Fake SecureStorageService — in-memory, no platform plugins
// =============================================================================

class FakeSecureStorage extends SecureStorageService {
  @override
  Future<void> saveTokens(String accessToken, String refreshToken) async {}

  @override
  Future<String?> getAccessToken() async => null;

  @override
  Future<String?> getRefreshToken() async => null;

  @override
  Future<void> clearTokens() async {}

  @override
  Future<bool> hasTokens() async => false;

  @override
  Future<void> saveUser(Map<String, dynamic> userJson) async {}

  @override
  Future<Map<String, dynamic>?> getUser() async => null;

  @override
  Future<void> clearAll() async {}
}

// =============================================================================
// Fake ApiClient — all calls error by default
// =============================================================================

class FakeApiClient extends ApiClient {
  FakeApiClient() : super(storage: FakeSecureStorage());

  @override
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
  }) async {
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }

  @override
  Future<Response<T>> post<T>(String path, {dynamic data}) async {
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }
}

// =============================================================================
// Helper — create a real AuthNotifier with controlled initial state
// =============================================================================

AuthNotifier _createAuthNotifier(Map<String, dynamic> user) {
  final storage = FakeSecureStorage();
  final api = FakeApiClient();
  final notifier = AuthNotifier(apiClient: api, storage: storage);
  // Force the desired state (overrides the async _tryRestoreSession)
  // ignore: invalid_use_of_protected_member
  notifier.state = AuthState(
    status: AuthStatus.authenticated,
    user: user,
  );
  return notifier;
}

// =============================================================================
// FakeThemeModeNotifier — avoids FlutterSecureStorage platform calls
// =============================================================================

class FakeThemeModeNotifier extends ThemeModeNotifier {
  FakeThemeModeNotifier() : super() {
    // Override the super constructor's _loadTheme() call by setting state
    // ignore: invalid_use_of_protected_member
    state = ThemeMode.light;
  }

  @override
  Future<void> toggle() async {
    // ignore: invalid_use_of_protected_member
    state = state == ThemeMode.light ? ThemeMode.dark : ThemeMode.light;
  }

  @override
  Future<void> setThemeMode(ThemeMode mode) async {
    // ignore: invalid_use_of_protected_member
    state = mode;
  }
}

// =============================================================================
// Helper — pump a testable ProfileScreen with overridden providers
// =============================================================================

Widget _buildTestableProfileScreen({
  required Map<String, dynamic> user,
  Map<String, dynamic>? profileData,
}) {
  return ProviderScope(
    overrides: [
      authProvider.overrideWith((_) => _createAuthNotifier(user)),
      themeModeProvider.overrideWith((_) => FakeThemeModeNotifier()),
      profileProvider.overrideWith((ref) async {
        if (profileData != null) return profileData;
        return <String, dynamic>{};
      }),
    ],
    child: MaterialApp(
      theme: AppTheme.light,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en')],
      home: const ProfileScreen(),
    ),
  );
}

// =============================================================================
// Test data
// =============================================================================

const _providerUser = {
  'id': 'user-123',
  'email': 'john@example.com',
  'first_name': 'John',
  'last_name': 'Doe',
  'display_name': 'John Doe',
  'role': 'provider',
};

const _agencyUser = {
  'id': 'user-456',
  'email': 'agency@example.com',
  'display_name': 'Acme Agency',
  'role': 'agency',
};

const _enterpriseUser = {
  'id': 'user-789',
  'email': 'corp@example.com',
  'display_name': 'Big Corp',
  'role': 'enterprise',
};

// =============================================================================
// Tests
// =============================================================================

void main() {
  // ---------------------------------------------------------------------------
  // App bar
  // ---------------------------------------------------------------------------

  group('ProfileScreen app bar', () {
    testWidgets('renders "My Profile" in the app bar', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.text('My Profile'), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // User info display
  // ---------------------------------------------------------------------------

  group('ProfileScreen user info', () {
    testWidgets('shows user display name', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.text('John Doe'), findsOneWidget);
    });

    testWidgets('shows user email', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.text('john@example.com'), findsOneWidget);
    });

    testWidgets('shows initials when no photo', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      // "John Doe" -> initials "JD" in CircleAvatar
      expect(find.text('JD'), findsOneWidget);
    });

    testWidgets('shows "User" when display name is empty', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: const {
          'id': 'user-000',
          'email': 'nobody@example.com',
          'role': 'provider',
        }),
      );
      await tester.pumpAndSettle();

      expect(find.text('User'), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // Role badge
  // ---------------------------------------------------------------------------

  group('ProfileScreen role badge', () {
    testWidgets('shows "Freelance" for provider role', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.text('Freelance'), findsOneWidget);
    });

    testWidgets('shows "Agency" for agency role', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _agencyUser),
      );
      await tester.pumpAndSettle();

      expect(find.text('Agency'), findsOneWidget);
    });

    testWidgets('shows "Enterprise" for enterprise role', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _enterpriseUser),
      );
      await tester.pumpAndSettle();

      expect(find.text('Enterprise'), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // Sign out button
  // ---------------------------------------------------------------------------

  group('ProfileScreen sign out', () {
    testWidgets('shows Sign Out button', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.text('Sign Out'), findsOneWidget);
    });

    testWidgets('Sign Out button has logout icon', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.byIcon(Icons.logout), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // Dark mode toggle
  // ---------------------------------------------------------------------------

  group('ProfileScreen dark mode toggle', () {
    testWidgets('shows Dark Mode label', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.text('Dark Mode'), findsOneWidget);
    });

    testWidgets('shows a Switch widget', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.byType(Switch), findsOneWidget);
    });

    testWidgets('shows light mode icon by default', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.byIcon(Icons.light_mode), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // Profile sections
  // ---------------------------------------------------------------------------

  group('ProfileScreen sections', () {
    testWidgets('shows Professional Title section', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.text('Professional Title'), findsOneWidget);
    });

    testWidgets('shows About section', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.text('About'), findsOneWidget);
    });

    testWidgets('shows Presentation Video section', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.text('Presentation Video'), findsAtLeastNWidgets(1));
    });

    testWidgets(
      'shows placeholder text when no professional title',
      (tester) async {
        await tester.pumpWidget(
          _buildTestableProfileScreen(
            user: _providerUser,
            profileData: const {'title': null, 'about': null},
          ),
        );
        await tester.pumpAndSettle();

        expect(find.text('Add your professional title'), findsOneWidget);
      },
    );

    testWidgets('shows placeholder text when no about', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(
          user: _providerUser,
          profileData: const {'title': null, 'about': null},
        ),
      );
      await tester.pumpAndSettle();

      expect(
        find.text('Tell others about yourself and your expertise'),
        findsOneWidget,
      );
    });

    testWidgets('shows title text when profile has title', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(
          user: _providerUser,
          profileData: const {
            'title': 'Senior Flutter Developer',
            'about': null,
          },
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Senior Flutter Developer'), findsOneWidget);
    });

    testWidgets('shows about text when profile has about', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(
          user: _providerUser,
          profileData: const {
            'title': null,
            'about': 'I build mobile apps with Flutter.',
          },
        ),
      );
      await tester.pumpAndSettle();

      expect(
        find.text('I build mobile apps with Flutter.'),
        findsOneWidget,
      );
    });
  });

  // ---------------------------------------------------------------------------
  // Camera icon for photo upload
  // ---------------------------------------------------------------------------

  group('ProfileScreen photo upload', () {
    testWidgets('shows camera icon on avatar', (tester) async {
      await tester.pumpWidget(
        _buildTestableProfileScreen(user: _providerUser),
      );
      await tester.pumpAndSettle();

      expect(find.byIcon(Icons.camera_alt), findsOneWidget);
    });
  });
}
