import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'helpers/test_helpers.dart';

/// Integration tests for the Profile screen:
///
/// - User info display (name, email, role badge)
/// - Photo placeholder and upload trigger
/// - Professional title section (empty state)
/// - Presentation video section (empty state, upload trigger)
/// - About section (empty state, edit trigger)
/// - Dark mode toggle
/// - Sign out button and logout flow
///
/// Each test registers a fresh provider user for isolation.
/// Requires the Go backend to be running.
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  // -------------------------------------------------------------------------
  // Helper: register a provider and navigate to the Profile screen
  // -------------------------------------------------------------------------
  Future<void> registerAndGoToProfile(WidgetTester tester) async {
    await initApp(tester);
    await tapText(tester, 'Sign Up');
    await waitForSettle(tester);
    await tapText(tester, 'Freelance / Business Referrer');
    await waitForSettle(tester);

    final email = uniqueEmail();
    await fillField(tester, 0, 'ProfileTest');
    await fillField(tester, 1, 'User');
    await fillField(tester, 2, email);
    await fillField(tester, 3, testPassword);
    await fillField(tester, 4, testPassword);

    await scrollDown(tester, dy: -300);
    final submitButton = find.byType(ElevatedButton).first;
    await tester.tap(submitButton);
    await waitForSettle(tester, seconds: 10);

    // Navigate to profile via bottom nav
    await tapBottomNavItem(tester, 3);
    await waitForSettle(tester, seconds: 3);
  }

  // =========================================================================
  // Profile Header
  // =========================================================================

  group('Profile — Header Card', () {
    testWidgets('profile screen shows app bar with title', (tester) async {
      await registerAndGoToProfile(tester);

      expectText('My Profile');
    });

    testWidgets('profile screen shows user display name', (tester) async {
      await registerAndGoToProfile(tester);

      // The provider's first_name is used as display_name fallback
      expectText('ProfileTest');
    });

    testWidgets('profile screen shows role badge for provider',
        (tester) async {
      await registerAndGoToProfile(tester);

      // Provider role badge shows "Freelance"
      expectText('Freelance');
    });

    testWidgets('profile shows avatar with initials when no photo',
        (tester) async {
      await registerAndGoToProfile(tester);

      // The avatar should show a CircleAvatar
      expect(find.byType(CircleAvatar), findsWidgets);

      // Camera badge icon is visible on the avatar
      expectIcon(Icons.camera_alt);
    });

    testWidgets('tapping photo opens upload bottom sheet', (tester) async {
      await registerAndGoToProfile(tester);

      // Tap the camera icon overlay on the avatar
      await tapIcon(tester, Icons.camera_alt);
      await waitForSettle(tester, seconds: 2);

      // Upload bottom sheet should appear with title and options
      expectText('Add a photo');
      expectText('Take a photo');
      expectText('Choose from gallery');
      expectText('Cancel');
    });

    testWidgets('dismissing photo upload bottom sheet returns to profile',
        (tester) async {
      await registerAndGoToProfile(tester);

      await tapIcon(tester, Icons.camera_alt);
      await waitForSettle(tester, seconds: 2);

      // Tap Cancel
      await tapText(tester, 'Cancel');
      await waitForSettle(tester);

      // Still on profile screen
      expectText('My Profile');
    });
  });

  // =========================================================================
  // Professional Title Section
  // =========================================================================

  group('Profile — Professional Title', () {
    testWidgets('professional title section shows empty state',
        (tester) async {
      await registerAndGoToProfile(tester);

      expectText('Professional Title');
      expectText('Add your professional title');
    });
  });

  // =========================================================================
  // Presentation Video Section
  // =========================================================================

  group('Profile — Presentation Video', () {
    testWidgets('video section shows empty state for new user',
        (tester) async {
      await registerAndGoToProfile(tester);

      // Scroll down to see the video section
      await scrollDown(tester, dy: -200);

      expectText('Presentation Video');
      expectText('No presentation video');
    });

    testWidgets('video section has "Add a video" button', (tester) async {
      await registerAndGoToProfile(tester);

      await scrollDown(tester, dy: -200);

      expectText('Add a video');
    });

    testWidgets('tapping "Add a video" opens upload bottom sheet',
        (tester) async {
      await registerAndGoToProfile(tester);

      await scrollDown(tester, dy: -200);

      await tapText(tester, 'Add a video');
      await waitForSettle(tester, seconds: 2);

      // Video upload bottom sheet
      expectText('Add a video');
      expectText('Choose from gallery');
      expectText('Cancel');
    });
  });

  // =========================================================================
  // About Section
  // =========================================================================

  group('Profile — About', () {
    testWidgets('about section shows empty state for new user',
        (tester) async {
      await registerAndGoToProfile(tester);

      await scrollDown(tester, dy: -300);

      expectText('About');
      expectText('Tell others about yourself and your expertise');
    });

    testWidgets('tapping about section opens edit bottom sheet',
        (tester) async {
      await registerAndGoToProfile(tester);

      await scrollDown(tester, dy: -300);

      // Tap the about section card (the entire card is wrapped in GestureDetector)
      await tapText(tester, 'Tell others about yourself and your expertise');
      await waitForSettle(tester, seconds: 2);

      // Edit about bottom sheet should appear
      // Title in the sheet
      expectText('About');
      // Hint text
      expectText('Tell others about yourself...');
      // Save button
      expectText('Save');
    });
  });

  // =========================================================================
  // Dark Mode Toggle
  // =========================================================================

  group('Profile — Dark Mode', () {
    testWidgets('dark mode toggle is visible', (tester) async {
      await registerAndGoToProfile(tester);

      await scrollDown(tester, dy: -400);

      expectText('Dark Mode');
      expect(find.byType(Switch), findsOneWidget);
    });

    testWidgets('tapping dark mode toggle switches theme', (tester) async {
      await registerAndGoToProfile(tester);

      await scrollDown(tester, dy: -400);

      // Initially light mode icon is shown
      expectIcon(Icons.light_mode);

      // Toggle the switch
      await tapByType<Switch>(tester);
      await waitForSettle(tester);

      // Now dark mode icon should be shown
      expectIcon(Icons.dark_mode);
    });
  });

  // =========================================================================
  // Logout
  // =========================================================================

  group('Profile — Logout', () {
    testWidgets('sign out button is visible on profile screen',
        (tester) async {
      await registerAndGoToProfile(tester);

      await scrollDown(tester, dy: -500);

      expectText('Sign Out');
      expectIcon(Icons.logout);
    });

    testWidgets('tapping Sign Out returns to login screen', (tester) async {
      await registerAndGoToProfile(tester);

      await scrollDown(tester, dy: -500);

      await tapText(tester, 'Sign Out');
      await waitForSettle(tester, seconds: 5);

      // Should be back on the login screen
      expectText('Welcome back,');
      expectText('Sign In');
      expect(find.byType(TextFormField), findsNWidgets(2));
    });
  });

  // =========================================================================
  // Agency Profile
  // =========================================================================

  group('Profile — Agency Role Badge', () {
    testWidgets('agency user profile shows Agency badge', (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Agency / IT Services');
      await waitForSettle(tester);

      final email = uniqueEmail();
      await fillField(tester, 0, 'ProfileAgency');
      await fillField(tester, 1, email);
      await fillField(tester, 2, testPassword);
      await fillField(tester, 3, testPassword);

      await scrollDown(tester, dy: -300);
      final submitButton = find.byType(ElevatedButton).first;
      await tester.tap(submitButton);
      await waitForSettle(tester, seconds: 10);

      // Navigate to profile
      await tapBottomNavItem(tester, 3);
      await waitForSettle(tester, seconds: 3);

      // Role badge shows "Agency"
      expectText('Agency');
      expectText('My Profile');
    });
  });

  group('Profile — Enterprise Role Badge', () {
    testWidgets('enterprise user profile shows Enterprise badge',
        (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await scrollDown(tester, dy: -200);
      await tapText(tester, 'Enterprise');
      await waitForSettle(tester);

      final email = uniqueEmail();
      await fillField(tester, 0, 'ProfileEnterprise');
      await fillField(tester, 1, email);
      await fillField(tester, 2, testPassword);
      await fillField(tester, 3, testPassword);

      await scrollDown(tester, dy: -300);
      final submitButton = find.byType(ElevatedButton).first;
      await tester.tap(submitButton);
      await waitForSettle(tester, seconds: 10);

      // Navigate to profile
      await tapBottomNavItem(tester, 3);
      await waitForSettle(tester, seconds: 3);

      // Role badge shows "Enterprise"
      expectText('Enterprise');
      expectText('My Profile');
    });
  });
}
