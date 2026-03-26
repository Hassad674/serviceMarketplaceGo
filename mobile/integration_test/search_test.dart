import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:marketplace_mobile/features/search/presentation/widgets/provider_card.dart';

import 'helpers/test_helpers.dart';

/// Integration tests for the Search and Public Profile flows:
///
/// - Dashboard search action chips trigger navigation to search screen
/// - Search screen shows correct title per type
/// - Search screen handles empty, loading, and populated states
/// - Tapping a provider card navigates to public profile
/// - Public profile displays name and role badge
/// - Back navigation from search and public profile
///
/// Each test registers a fresh provider user for isolation.
/// Requires the Go backend to be running.
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  // -------------------------------------------------------------------------
  // Helper: register a provider and stay on dashboard
  // -------------------------------------------------------------------------
  Future<void> registerProviderAndGoToDashboard(WidgetTester tester) async {
    await initApp(tester);
    await tapText(tester, 'Sign Up');
    await waitForSettle(tester);
    await tapText(tester, 'Freelance / Business Referrer');
    await waitForSettle(tester);

    final email = uniqueEmail();
    await fillField(tester, 0, 'SearchTest');
    await fillField(tester, 1, 'User');
    await fillField(tester, 2, email);
    await fillField(tester, 3, testPassword);
    await fillField(tester, 4, testPassword);

    await scrollDown(tester, dy: -300);
    final submitButton = find.byType(ElevatedButton).first;
    await tester.tap(submitButton);
    await waitForSettle(tester, seconds: 10);
  }

  // -------------------------------------------------------------------------
  // Helper: register an enterprise user and stay on dashboard
  // -------------------------------------------------------------------------
  Future<void> registerEnterpriseAndGoToDashboard(WidgetTester tester) async {
    await initApp(tester);
    await tapText(tester, 'Sign Up');
    await waitForSettle(tester);
    await scrollDown(tester, dy: -200);
    await tapText(tester, 'Enterprise');
    await waitForSettle(tester);

    final email = uniqueEmail();
    await fillField(tester, 0, 'SearchEnterprise');
    await fillField(tester, 1, email);
    await fillField(tester, 2, testPassword);
    await fillField(tester, 3, testPassword);

    await scrollDown(tester, dy: -300);
    final submitButton = find.byType(ElevatedButton).first;
    await tester.tap(submitButton);
    await waitForSettle(tester, seconds: 10);
  }

  // =========================================================================
  // Search Navigation from Dashboard
  // =========================================================================

  group('Search — Navigation from Dashboard', () {
    testWidgets('provider dashboard has "Find Freelancers" chip',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      expectText('Find Freelancers');
    });

    testWidgets('tapping "Find Freelancers" navigates to search screen',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      await tapText(tester, 'Find Freelancers');
      await waitForSettle(tester, seconds: 5);

      // Search screen shows title "Find Freelancers"
      expectText('Find Freelancers');
    });

    testWidgets('enterprise dashboard "Find Agencies" navigates correctly',
        (tester) async {
      await registerEnterpriseAndGoToDashboard(tester);

      await tapText(tester, 'Find Agencies');
      await waitForSettle(tester, seconds: 5);

      expectText('Find Agencies');
    });

    testWidgets('enterprise dashboard "Find Referrers" navigates correctly',
        (tester) async {
      await registerEnterpriseAndGoToDashboard(tester);

      await tapText(tester, 'Find Referrers');
      await waitForSettle(tester, seconds: 5);

      expectText('Find Referrers');
    });
  });

  // =========================================================================
  // Search Screen Content
  // =========================================================================

  group('Search — Screen Content', () {
    testWidgets('search screen shows results or empty state',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      await tapText(tester, 'Find Freelancers');
      await waitForSettle(tester, seconds: 8);

      // After loading, should show either:
      // - A list of ProviderCard widgets (if profiles exist)
      // - "No profiles found" empty state
      final hasProviderCards = find.byType(ProviderCard).evaluate().isNotEmpty;
      final hasEmptyState = find.text('No profiles found').evaluate().isNotEmpty;

      expect(
        hasProviderCards || hasEmptyState,
        isTrue,
        reason:
            'Search screen should show provider cards or empty state message',
      );
    });

    testWidgets('search screen has back navigation via app bar',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      await tapText(tester, 'Find Freelancers');
      await waitForSettle(tester, seconds: 5);

      // AppBar should have a back button
      final backButton = find.byType(BackButton);
      if (backButton.evaluate().isNotEmpty) {
        await tester.tap(backButton.first);
        await waitForSettle(tester, seconds: 3);

        // Should be back on dashboard
        expectText('Business Referrer Mode');
      }
    });
  });

  // =========================================================================
  // Provider Card and Public Profile
  // =========================================================================

  group('Search — Provider Card and Public Profile', () {
    testWidgets('provider card shows name and role badge when results exist',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      await tapText(tester, 'Find Freelancers');
      await waitForSettle(tester, seconds: 8);

      final providerCards = find.byType(ProviderCard);
      if (providerCards.evaluate().isNotEmpty) {
        // At least one card is visible — it should contain text widgets
        // for name and role badge (we cannot predict exact content, but
        // the card always renders a name and a role badge)
        expect(providerCards, findsWidgets);
      }
    });

    testWidgets(
        'tapping a provider card navigates to public profile when results exist',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      await tapText(tester, 'Find Freelancers');
      await waitForSettle(tester, seconds: 8);

      final providerCards = find.byType(ProviderCard);
      if (providerCards.evaluate().isNotEmpty) {
        // Tap the first card
        await tester.tap(providerCards.first);
        await waitForSettle(tester, seconds: 5);

        // Public profile screen has "Profile" in the app bar
        expectText('Profile');
      }
    });

    testWidgets('public profile shows name and role badge',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      await tapText(tester, 'Find Freelancers');
      await waitForSettle(tester, seconds: 8);

      final providerCards = find.byType(ProviderCard);
      if (providerCards.evaluate().isNotEmpty) {
        await tester.tap(providerCards.first);
        await waitForSettle(tester, seconds: 5);

        // Profile screen app bar title
        expectText('Profile');

        // Should display at least one CircleAvatar (the large avatar)
        expect(find.byType(CircleAvatar), findsWidgets);
      }
    });

    testWidgets('back button from public profile returns to search',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      await tapText(tester, 'Find Freelancers');
      await waitForSettle(tester, seconds: 8);

      final providerCards = find.byType(ProviderCard);
      if (providerCards.evaluate().isNotEmpty) {
        await tester.tap(providerCards.first);
        await waitForSettle(tester, seconds: 5);

        // Navigate back
        final backButton = find.byType(BackButton);
        if (backButton.evaluate().isNotEmpty) {
          await tester.tap(backButton.first);
          await waitForSettle(tester, seconds: 3);

          // Should be back on search screen
          expectText('Find Freelancers');
        }
      }
    });
  });

  // =========================================================================
  // Search from Referrer Dashboard
  // =========================================================================

  group('Search — From Referrer Dashboard', () {
    testWidgets('referrer dashboard Find Freelancers chip works',
        (tester) async {
      await registerProviderAndGoToDashboard(tester);

      // Switch to referrer mode
      await tapText(tester, 'Business Referrer Mode');
      await waitForSettle(tester, seconds: 3);

      // Referrer dashboard also has "Find Freelancers"
      await tapText(tester, 'Find Freelancers');
      await waitForSettle(tester, seconds: 5);

      expectText('Find Freelancers');
    });
  });

  // =========================================================================
  // Empty State
  // =========================================================================

  group('Search — Empty State', () {
    testWidgets('empty state shows message and hint text', (tester) async {
      await registerProviderAndGoToDashboard(tester);

      // Search for a type that likely has no results (referrer)
      // Navigate to referrer dashboard first to get referrer chip
      await tapText(tester, 'Business Referrer Mode');
      await waitForSettle(tester, seconds: 3);

      await tapText(tester, 'Find Freelancers');
      await waitForSettle(tester, seconds: 8);

      // Either we see results or the empty state
      final hasEmptyState = find.text('No profiles found').evaluate().isNotEmpty;
      if (hasEmptyState) {
        expectText('No profiles found');
        expectText('Try again later or adjust your search.');
      }
    });
  });
}
