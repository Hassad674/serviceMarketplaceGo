import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'helpers/test_helpers.dart';

/// Integration tests for the authentication flows:
///
/// - Login screen rendering
/// - Navigation from login to role selection
/// - Navigation from role selection to each registration form
/// - Login with invalid credentials (error handling)
/// - Full registration flow for each role (provider, agency, enterprise)
///
/// These tests require the Go backend to be running and reachable at the
/// API_URL configured via --dart-define.
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  group('Authentication — Login Screen', () {
    testWidgets('app starts on login screen with email and password fields',
        (tester) async {
      await initApp(tester);

      // The login screen shows the localized "Welcome back," header
      expectText('Welcome back,');

      // Two form fields: email and password
      expect(find.byType(TextFormField), findsNWidgets(2));

      // Sign In button is present
      expectText('Sign In');

      // Link to registration
      expectText('Sign Up');
    });

    testWidgets('password visibility toggle works', (tester) async {
      await initApp(tester);

      // Initially the password field is obscured — visibility icon is shown
      expectIcon(Icons.visibility_outlined);

      // Tap the visibility toggle
      await tapIcon(tester, Icons.visibility_outlined);

      // Now the icon should switch to visibility_off
      expectIcon(Icons.visibility_off_outlined);
    });

    testWidgets('empty form submission shows validation errors', (tester) async {
      await initApp(tester);

      // Tap Sign In without filling any fields
      await tapText(tester, 'Sign In');

      // Validation error messages should appear
      // The email validator returns l10n.fieldRequired ("This field is required")
      expectText('This field is required');
    });

    testWidgets('login with invalid credentials stays on login screen',
        (tester) async {
      await initApp(tester);

      // Enter invalid credentials
      await fillField(tester, 0, 'nonexistent@fake-domain.com');
      await fillField(tester, 1, 'WrongPassword123!');

      // Submit
      await tapText(tester, 'Sign In');
      await waitForSettle(tester, seconds: 8);

      // Should remain on login screen — "Welcome back," still visible
      expectText('Welcome back,');

      // The auth provider should have set an errorMessage, shown in the banner
      // (The exact message comes from the backend API error)
    });

    testWidgets('can navigate to role selection via Sign Up link',
        (tester) async {
      await initApp(tester);

      // Tap the "Sign Up" text button
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);

      // Role selection screen header
      expectText('Join the marketplace');
      expectText('Choose your profile to get started');

      // Three role cards visible
      expectText('Agency / IT Services');
      expectText('Freelance / Business Referrer');
      expectText('Enterprise');
    });
  });

  group('Authentication — Role Selection', () {
    testWidgets('role selection shows all three role cards', (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);

      // Verify each card text and description
      expectText('Agency / IT Services');
      expectText('Manage your agency, providers, and missions');

      expectText('Freelance / Business Referrer');
      expectText('Offer your services or connect professionals together');

      expectText('Enterprise');
      expectText('Post your projects and find the best providers');

      // "Already registered?" link back to login
      expectText('Already registered?');
    });

    testWidgets('tapping Agency card navigates to agency registration',
        (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);

      // Tap the Agency card
      await tapText(tester, 'Agency / IT Services');
      await waitForSettle(tester);

      // Agency register screen
      expectText('Agency Sign Up');
      // Role badge
      expectText('Agency / IT Services');
      // Form fields: agency name, email, password, confirm password
      expect(find.byType(TextFormField), findsNWidgets(4));
    });

    testWidgets('tapping Freelance card navigates to provider registration',
        (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);

      // Tap the Freelance card
      await tapText(tester, 'Freelance / Business Referrer');
      await waitForSettle(tester);

      // Provider register screen
      expectText('Freelance Sign Up');
      // Role badge
      expectText('Freelance / Business Referrer');
      // Form fields: first name, last name, email, password, confirm password
      expect(find.byType(TextFormField), findsNWidgets(5));
    });

    testWidgets('tapping Enterprise card navigates to enterprise registration',
        (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);

      // Scroll down to reveal the Enterprise card if off-screen
      await scrollDown(tester, dy: -200);

      // Tap the Enterprise card
      await tapText(tester, 'Enterprise');
      await waitForSettle(tester);

      // Enterprise register screen
      expectText('Enterprise Sign Up');
      // Form fields: company name, email, password, confirm password
      expect(find.byType(TextFormField), findsNWidgets(4));
    });

    testWidgets('Sign In link on role selection returns to login',
        (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);

      // Scroll down to find "Sign In" link
      await scrollDown(tester, dy: -200);
      await tapText(tester, 'Sign In');
      await waitForSettle(tester);

      // Should be back on login
      expectText('Welcome back,');
    });
  });

  group('Authentication — Provider Registration', () {
    testWidgets('empty provider form shows validation errors', (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Freelance / Business Referrer');
      await waitForSettle(tester);

      // Scroll down to the submit button and tap
      await scrollDown(tester, dy: -300);
      await tapText(tester, 'Create Account');
      await waitForSettle(tester);

      // Validation errors should be visible
      expectText('First name is required');
    });

    testWidgets('password mismatch shows error', (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Freelance / Business Referrer');
      await waitForSettle(tester);

      await fillField(tester, 0, 'John');
      await fillField(tester, 1, 'Doe');
      await fillField(tester, 2, uniqueEmail());
      await fillField(tester, 3, testPassword);
      await fillField(tester, 4, 'DifferentPassword123!');

      await scrollDown(tester, dy: -300);
      await tapText(tester, 'Create Account');
      await waitForSettle(tester);

      expectText('Passwords do not match');
    });

    testWidgets('register provider and reach dashboard', (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Freelance / Business Referrer');
      await waitForSettle(tester);

      final email = uniqueEmail();
      await fillField(tester, 0, 'IntegTest');
      await fillField(tester, 1, 'Provider');
      await fillField(tester, 2, email);
      await fillField(tester, 3, testPassword);
      await fillField(tester, 4, testPassword);

      // Scroll to submit
      await scrollDown(tester, dy: -300);

      // Tap "Create Account"
      final submitButton = find.byType(ElevatedButton).first;
      await tester.tap(submitButton);
      await waitForSettle(tester, seconds: 10);

      // Should land on the provider dashboard
      // The welcome banner shows "Welcome back,"
      expectText('Welcome back,');

      // Dashboard should show Marketplace in app bar
      expectText('Marketplace');

      // Provider dashboard shows the "Business Referrer Mode" button
      expectText('Business Referrer Mode');

      // Bottom navigation should be visible with 4 items
      expect(find.byType(NavigationDestination), findsNWidgets(4));
    });

    testWidgets('back button on provider register returns to role selection',
        (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Freelance / Business Referrer');
      await waitForSettle(tester);

      // Tap the back arrow
      await tapIcon(tester, Icons.arrow_back);
      await waitForSettle(tester);

      // Should be back on role selection
      expectText('Join the marketplace');
    });
  });

  group('Authentication — Agency Registration', () {
    testWidgets('empty agency form shows validation errors', (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Agency / IT Services');
      await waitForSettle(tester);

      await scrollDown(tester, dy: -300);
      await tapText(tester, 'Create Account');
      await waitForSettle(tester);

      expectText('Agency name is required');
    });

    testWidgets('register agency and reach dashboard', (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Agency / IT Services');
      await waitForSettle(tester);

      final email = uniqueEmail();
      await fillField(tester, 0, 'IntegTest Agency');
      await fillField(tester, 1, email);
      await fillField(tester, 2, testPassword);
      await fillField(tester, 3, testPassword);

      await scrollDown(tester, dy: -300);

      final submitButton = find.byType(ElevatedButton).first;
      await tester.tap(submitButton);
      await waitForSettle(tester, seconds: 10);

      // Should land on the agency dashboard
      expectText('Welcome back,');
      expectText('Marketplace');

      // Agency dashboard shows "Find Freelancers" and "Find Referrers" chips
      expectText('Find Freelancers');
      expectText('Find Referrers');
    });

    testWidgets('back button on agency register returns to role selection',
        (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Agency / IT Services');
      await waitForSettle(tester);

      await tapIcon(tester, Icons.arrow_back);
      await waitForSettle(tester);

      expectText('Join the marketplace');
    });
  });

  group('Authentication — Enterprise Registration', () {
    testWidgets('empty enterprise form shows validation errors',
        (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await scrollDown(tester, dy: -200);
      await tapText(tester, 'Enterprise');
      await waitForSettle(tester);

      await scrollDown(tester, dy: -300);
      await tapText(tester, 'Create Account');
      await waitForSettle(tester);

      expectText('Company name is required');
    });

    testWidgets('register enterprise and reach dashboard', (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await scrollDown(tester, dy: -200);
      await tapText(tester, 'Enterprise');
      await waitForSettle(tester);

      final email = uniqueEmail();
      await fillField(tester, 0, 'IntegTest Enterprise');
      await fillField(tester, 1, email);
      await fillField(tester, 2, testPassword);
      await fillField(tester, 3, testPassword);

      await scrollDown(tester, dy: -300);

      final submitButton = find.byType(ElevatedButton).first;
      await tester.tap(submitButton);
      await waitForSettle(tester, seconds: 10);

      // Should land on enterprise dashboard
      expectText('Welcome back,');
      expectText('Marketplace');

      // Enterprise dashboard shows all three search chips
      expectText('Find Freelancers');
      expectText('Find Agencies');
      expectText('Find Referrers');
    });

    testWidgets('back button on enterprise register returns to role selection',
        (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await scrollDown(tester, dy: -200);
      await tapText(tester, 'Enterprise');
      await waitForSettle(tester);

      await tapIcon(tester, Icons.arrow_back);
      await waitForSettle(tester);

      expectText('Join the marketplace');
    });
  });
}
