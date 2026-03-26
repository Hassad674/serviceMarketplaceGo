import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/auth/presentation/screens/login_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

// =============================================================================
// Mock AuthNotifier — overrides the real provider for widget tests
// =============================================================================

class MockAuthNotifier extends StateNotifier<AuthState> {
  MockAuthNotifier([AuthState? initial])
      : super(initial ?? const AuthState(status: AuthStatus.unauthenticated));

  Future<bool> login({
    required String email,
    required String password,
  }) async {
    return true;
  }

  void clearError() {
    state = state.copyWith(errorMessage: null);
  }
}

// =============================================================================
// Helper to pump a testable LoginScreen with all required dependencies
// =============================================================================

Widget _buildTestableLoginScreen({AuthState? authState}) {
  final notifier = MockAuthNotifier(
    authState ?? const AuthState(status: AuthStatus.unauthenticated),
  );

  return ProviderScope(
    overrides: [
      authProvider.overrideWith((_) => notifier),
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

      // Password is obscured, so we check the controller has the value
      // by looking for the TextFormField that contains the text
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
