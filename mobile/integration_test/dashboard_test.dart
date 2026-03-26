import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'helpers/test_helpers.dart';

/// Integration tests for the dashboard screens:
///
/// - Provider dashboard: stat cards, search chips, referrer mode toggle
/// - Agency dashboard: stat cards, search chips
/// - Enterprise dashboard: stat cards, search chips
/// - Referrer dashboard: referrer-specific stats, back to freelance toggle
/// - Bottom navigation: 4 items, tab switching
///
/// Each test registers a fresh user to ensure test isolation.
/// Requires the Go backend to be running.
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  // -------------------------------------------------------------------------
  // Helper: register a provider and arrive at dashboard
  // -------------------------------------------------------------------------
  Future<void> registerProviderAndGoToDashboard(WidgetTester tester) async {
    await initApp(tester);
    await tapText(tester, 'Sign Up');
    await waitForSettle(tester);
    await tapText(tester, 'Freelance / Business Referrer');
    await waitForSettle(tester);

    final email = uniqueEmail();
    await fillField(tester, 0, 'DashTest');
    await fillField(tester, 1, 'Provider');
    await fillField(tester, 2, email);
    await fillField(tester, 3, testPassword);
    await fillField(tester, 4, testPassword);

    await scrollDown(tester, dy: -300);
    final submitButton = find.byType(ElevatedButton).first;
    await tester.tap(submitButton);
    await waitForSettle(tester, seconds: 10);
  }

  // -------------------------------------------------------------------------
  // Helper: register an agency and arrive at dashboard
  // -------------------------------------------------------------------------
  Future<void> registerAgencyAndGoToDashboard(WidgetTester tester) async {
    await initApp(tester);
    await tapText(tester, 'Sign Up');
    await waitForSettle(tester);
    await tapText(tester, 'Agency / IT Services');
    await waitForSettle(tester);

    final email = uniqueEmail();
    await fillField(tester, 0, 'DashTest Agency');
    await fillField(tester, 1, email);
    await fillField(tester, 2, testPassword);
    await fillField(tester, 3, testPassword);

    await scrollDown(tester, dy: -300);
    final submitButton = find.byType(ElevatedButton).first;
    await tester.tap(submitButton);
    await waitForSettle(tester, seconds: 10);
  }

  // -------------------------------------------------------------------------
  // Helper: register an enterprise and arrive at dashboard
  // -------------------------------------------------------------------------
  Future<void> registerEnterpriseAndGoToDashboard(WidgetTester tester) async {
    await initApp(tester);
    await tapText(tester, 'Sign Up');
    await waitForSettle(tester);
    await scrollDown(tester, dy: -200);
    await tapText(tester, 'Enterprise');
    await waitForSettle(tester);

    final email = uniqueEmail();
    await fillField(tester, 0, 'DashTest Enterprise');
    await fillField(tester, 1, email);
    await fillField(tester, 2, testPassword);
    await fillField(tester, 3, testPassword);

    await scrollDown(tester, dy: -300);
    final submitButton = find.byType(ElevatedButton).first;
    await tester.tap(submitButton);
    await waitForSettle(tester, seconds: 10);
  }

  // =========================================================================
  // Provider Dashboard
  // =========================================================================

  group('Dashboard — Provider (Freelance)', () {
    testWidgets('provider dashboard shows welcome banner and stat cards',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      // Welcome banner
      expectText('Welcome back,');

      // App bar title
      expectText('Marketplace');

      // Stat cards (values show "0" by default for new users)
      expectText('Active Missions');
      expectText('Unread Messages');
      expectText('Monthly Revenue');
      expectText('0 EUR');
    });

    testWidgets('provider dashboard shows search action chip',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      // Provider only gets "Find Freelancers"
      expectText('Find Freelancers');
    });

    testWidgets('provider dashboard shows Business Referrer Mode button',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      expectText('Business Referrer Mode');
      expectIcon(Icons.swap_horiz);
    });

    testWidgets('provider dashboard has 4 bottom navigation items',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      expect(find.byType(NavigationDestination), findsNWidgets(4));

      // Verify labels (localized)
      expectText('Home');
      expectText('Messages');
      expectText('Missions');
      expectText('Profile');
    });

    testWidgets('tapping Business Referrer Mode navigates to referrer dashboard',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      await tapText(tester, 'Business Referrer Mode');
      await waitForSettle(tester, seconds: 3);

      // Referrer dashboard app bar
      expectText('Referrer Mode');

      // Referrer-specific stat cards
      expectText('Referrals');
      expectText('Commissions');
      expectText('Completed Missions');

      // "Freelance Dashboard" button to switch back
      expectText('Freelance Dashboard');
    });
  });

  // =========================================================================
  // Referrer Dashboard
  // =========================================================================

  group('Dashboard — Referrer Mode', () {
    testWidgets('referrer dashboard shows 4 stat cards', (tester) async {
      await registerProviderAndGoToDashboard(tester);
      await tapText(tester, 'Business Referrer Mode');
      await waitForSettle(tester, seconds: 3);

      expectText('Referrals');
      expectText('Pending response');
      expectText('Active Missions');
      expectText('Active contracts');
      expectText('Completed Missions');
      expectText('Total history');
      expectText('Commissions');
      expectText('Total earned');
    });

    testWidgets('referrer dashboard shows welcome banner', (tester) async {
      await registerProviderAndGoToDashboard(tester);
      await tapText(tester, 'Business Referrer Mode');
      await waitForSettle(tester, seconds: 3);

      expectText('Welcome back,');
    });

    testWidgets('referrer dashboard has Find Freelancers search chip',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);
      await tapText(tester, 'Business Referrer Mode');
      await waitForSettle(tester, seconds: 3);

      expectText('Find Freelancers');
    });

    testWidgets('Freelance Dashboard button returns to provider dashboard',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);
      await tapText(tester, 'Business Referrer Mode');
      await waitForSettle(tester, seconds: 3);

      // Tap "Freelance Dashboard" to switch back
      await tapText(tester, 'Freelance Dashboard');
      await waitForSettle(tester, seconds: 3);

      // Back on provider dashboard — Business Referrer Mode visible again
      expectText('Business Referrer Mode');
      expectText('Active Missions');
    });
  });

  // =========================================================================
  // Agency Dashboard
  // =========================================================================

  group('Dashboard — Agency', () {
    testWidgets('agency dashboard shows welcome banner and stat cards',
        (tester) async {
      await registerAgencyAndGoToDashboard(tester);

      expectText('Welcome back,');
      expectText('Marketplace');

      // Agency-specific stat labels
      expectText('Active Missions');
      expectText('Unread Messages');
      expectText('Monthly Revenue');
      expectText('0 EUR');
    });

    testWidgets('agency dashboard shows search chips', (tester) async {
      await registerAgencyAndGoToDashboard(tester);

      expectText('Find Freelancers');
      expectText('Find Referrers');
    });

    testWidgets('agency dashboard does NOT show Business Referrer Mode',
        (tester) async {
      await registerAgencyAndGoToDashboard(tester);

      expectNoText('Business Referrer Mode');
    });

    testWidgets('agency dashboard has notification bell icon',
        (tester) async {
      await registerAgencyAndGoToDashboard(tester);

      expectIcon(Icons.notifications_outlined);
    });
  });

  // =========================================================================
  // Enterprise Dashboard
  // =========================================================================

  group('Dashboard — Enterprise', () {
    testWidgets('enterprise dashboard shows welcome banner and stat cards',
        (tester) async {
      await registerEnterpriseAndGoToDashboard(tester);

      expectText('Welcome back,');
      expectText('Marketplace');

      // Enterprise-specific stat labels
      expectText('Active Projects');
      expectText('Unread Messages');
      expectText('Total Budget');
      expectText('0 EUR');
    });

    testWidgets('enterprise dashboard shows all three search chips',
        (tester) async {
      await registerEnterpriseAndGoToDashboard(tester);

      expectText('Find Freelancers');
      expectText('Find Agencies');
      expectText('Find Referrers');
    });

    testWidgets('enterprise dashboard does NOT show Business Referrer Mode',
        (tester) async {
      await registerEnterpriseAndGoToDashboard(tester);

      expectNoText('Business Referrer Mode');
    });
  });

  // =========================================================================
  // Bottom Navigation
  // =========================================================================

  group('Dashboard — Bottom Navigation', () {
    testWidgets('tapping Profile tab navigates to profile screen',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      // Tap the 4th nav item (Profile, index 3)
      await tapBottomNavItem(tester, 3);

      // Profile screen app bar
      expectText('My Profile');
    });

    testWidgets('tapping Messages tab navigates to messages placeholder',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      // Tap Messages (index 1)
      await tapBottomNavItem(tester, 1);

      expectText('Messages');
      expectText('Coming soon');
    });

    testWidgets('tapping Missions tab navigates to missions placeholder',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      // Tap Missions (index 2)
      await tapBottomNavItem(tester, 2);

      expectText('My Missions');
      expectText('Coming soon');
    });

    testWidgets('tapping Home tab returns to dashboard', (tester) async {
      await registerProviderAndGoToDashboard(tester);

      // Navigate away first
      await tapBottomNavItem(tester, 3);
      expectText('My Profile');

      // Then back to Home (index 0)
      await tapBottomNavItem(tester, 0);

      // Should see the dashboard again
      expectText('Business Referrer Mode');
      expectText('Active Missions');
    });
  });
}
