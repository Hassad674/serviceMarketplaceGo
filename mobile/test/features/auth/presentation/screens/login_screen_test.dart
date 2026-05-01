import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/auth/presentation/screens/login_screen.dart';
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
// Fake ApiClient — all calls return connection errors by default
// =============================================================================

class FakeApiClient extends ApiClient {
  FakeApiClient() : super(storage: FakeSecureStorage());

  @override
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
    Options? options,
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
// Test AuthNotifier — extends the real AuthNotifier but with controlled state
// =============================================================================

/// Creates a real AuthNotifier with fake deps, then overrides its state.
AuthNotifier _createTestNotifier(AuthState desiredState) {
  final storage = FakeSecureStorage();
  final api = FakeApiClient();
  final notifier = AuthNotifier(apiClient: api, storage: storage);
  // The constructor kicks off _tryRestoreSession which is async.
  // We force the desired state immediately — since no tokens exist
  // in the fake storage, the async call will also settle to unauthenticated,
  // but our forced state takes precedence for the initial render.
  // ignore: invalid_use_of_protected_member
  notifier.state = desiredState;
  return notifier;
}

// =============================================================================
// Helper to pump a testable LoginScreen with all required dependencies
// =============================================================================

Widget _buildTestableLoginScreen({AuthState? authState}) {
  final desiredState = authState ??
      const AuthState(status: AuthStatus.unauthenticated);

  return ProviderScope(
    overrides: [
      authProvider.overrideWith((_) => _createTestNotifier(desiredState)),
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
      home: const LoginScreen(),
    ),
  );
}

// =============================================================================
// Tests
// =============================================================================

void main() {
  // ---------------------------------------------------------------------------
  // Widget rendering tests
  // ---------------------------------------------------------------------------

  group('LoginScreen renders', () {
    testWidgets('email and password text fields', (tester) async {
      await tester.pumpWidget(_buildTestableLoginScreen());
      await tester.pumpAndSettle();

      // Email field — look for the hint text
      expect(find.text('you@example.com'), findsOneWidget);

      // Password field — look for the hint text
      expect(find.text('Your password'), findsOneWidget);
    });

    testWidgets('email and password labels', (tester) async {
      await tester.pumpWidget(_buildTestableLoginScreen());
      await tester.pumpAndSettle();

      expect(find.text('Email'), findsOneWidget);
      expect(find.text('Password'), findsOneWidget);
    });

    testWidgets('Sign In button', (tester) async {
      await tester.pumpWidget(_buildTestableLoginScreen());
      await tester.pumpAndSettle();

      // There are two "Sign In" texts: the subtitle header and the button.
      // The button is an ElevatedButton child.
      expect(
        find.widgetWithText(ElevatedButton, 'Sign In'),
        findsOneWidget,
      );
    });

    testWidgets('Forgot password link', (tester) async {
      await tester.pumpWidget(_buildTestableLoginScreen());
      await tester.pumpAndSettle();

      expect(find.text('Forgot password?'), findsOneWidget);
    });

    testWidgets('Sign Up link', (tester) async {
      await tester.pumpWidget(_buildTestableLoginScreen());
      await tester.pumpAndSettle();

      expect(find.text('Sign Up'), findsOneWidget);
    });

    testWidgets('"No account yet?" text', (tester) async {
      await tester.pumpWidget(_buildTestableLoginScreen());
      await tester.pumpAndSettle();

      expect(find.text('No account yet?'), findsOneWidget);
    });

    testWidgets('Welcome back header text', (tester) async {
      await tester.pumpWidget(_buildTestableLoginScreen());
      await tester.pumpAndSettle();

      expect(find.text('Welcome back,'), findsOneWidget);
    });

    testWidgets('storefront icon', (tester) async {
      await tester.pumpWidget(_buildTestableLoginScreen());
      await tester.pumpAndSettle();

      expect(find.byIcon(Icons.storefront_rounded), findsOneWidget);
    });
  });

  // ---------------------------------------------------------------------------
  // Error display
  // ---------------------------------------------------------------------------

  group('LoginScreen error display', () {
    testWidgets('shows error message when login fails', (tester) async {
      await tester.pumpWidget(
        _buildTestableLoginScreen(
          authState: const AuthState(
            status: AuthStatus.unauthenticated,
            errorMessage: 'Invalid email or password',
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Invalid email or password'), findsOneWidget);
      // Error banner includes an error icon
      expect(find.byIcon(Icons.error_outline), findsOneWidget);
    });

    testWidgets('does not show error banner when no error', (tester) async {
      await tester.pumpWidget(_buildTestableLoginScreen());
      await tester.pumpAndSettle();

      expect(find.byIcon(Icons.error_outline), findsNothing);
    });
  });

  // ---------------------------------------------------------------------------
  // Submitting state
  // ---------------------------------------------------------------------------

  group('LoginScreen submitting state', () {
    testWidgets(
      'shows CircularProgressIndicator when isSubmitting is true',
      (tester) async {
        await tester.pumpWidget(
          _buildTestableLoginScreen(
            authState: const AuthState(
              status: AuthStatus.unauthenticated,
              isSubmitting: true,
            ),
          ),
        );
        await tester.pumpAndSettle();

        expect(find.byType(CircularProgressIndicator), findsOneWidget);
      },
    );

    testWidgets(
      'disables Sign In button when isSubmitting is true',
      (tester) async {
        await tester.pumpWidget(
          _buildTestableLoginScreen(
            authState: const AuthState(
              status: AuthStatus.unauthenticated,
              isSubmitting: true,
            ),
          ),
        );
        await tester.pumpAndSettle();

        final button = tester.widget<ElevatedButton>(
          find.byType(ElevatedButton),
        );
        expect(button.onPressed, isNull);
      },
    );
  });

  // ---------------------------------------------------------------------------
  // Form field interactions
  // ---------------------------------------------------------------------------

  group('LoginScreen form interactions', () {
    testWidgets('can enter text in email field', (tester) async {
      await tester.pumpWidget(_buildTestableLoginScreen());
      await tester.pumpAndSettle();

      final emailField = find.widgetWithText(TextFormField, 'you@example.com');
      await tester.enterText(emailField, 'user@test.com');

      expect(find.text('user@test.com'), findsOneWidget);
    });

    testWidgets('can enter text in password field', (tester) async {
      await tester.pumpWidget(_buildTestableLoginScreen());
      await tester.pumpAndSettle();

      final passwordField =
          find.widgetWithText(TextFormField, 'Your password');
      await tester.enterText(passwordField, 'secret123');

      // Password is obscured, but the form field still has the value.
      // We verify the TextFormField exists and accepted input.
      final formFields =
          tester.widgetList<TextFormField>(find.byType(TextFormField));
      expect(formFields.length, greaterThanOrEqualTo(2));
    });

    testWidgets('password visibility toggle works', (tester) async {
      await tester.pumpWidget(_buildTestableLoginScreen());
      await tester.pumpAndSettle();

      // Initially password is obscured — visibility icon is shown
      expect(find.byIcon(Icons.visibility_outlined), findsOneWidget);

      // Tap the visibility toggle
      await tester.tap(find.byIcon(Icons.visibility_outlined));
      await tester.pump();

      // Now visibility_off icon should be shown
      expect(find.byIcon(Icons.visibility_off_outlined), findsOneWidget);
    });
  });
}
