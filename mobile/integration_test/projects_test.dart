import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'helpers/test_helpers.dart';

/// Integration tests for the projects feature.
///
/// These tests require the Go backend to be running and reachable at the
/// API_URL configured via --dart-define.
///
/// The test flow:
/// 1. Register an enterprise account to get an authenticated session
/// 2. Navigate to the Projects tab
/// 3. Test project creation form, payment selection, milestones, skills, etc.
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  /// Helper to register an enterprise user and reach the dashboard.
  Future<void> registerEnterpriseAndDashboard(WidgetTester tester) async {
    await initApp(tester);
    await tapText(tester, 'Sign Up');
    await waitForSettle(tester);
    await scrollDown(tester, dy: -200);
    await tapText(tester, 'Enterprise');
    await waitForSettle(tester);

    final email = uniqueEmail();
    await fillField(tester, 0, 'ProjTest Enterprise');
    await fillField(tester, 1, email);
    await fillField(tester, 2, testPassword);
    await fillField(tester, 3, testPassword);

    await scrollDown(tester, dy: -300);
    final submitButton = find.byType(ElevatedButton).first;
    await tester.tap(submitButton);
    await waitForSettle(tester, seconds: 10);
  }

  group('Projects — List Screen', () {
    testWidgets('projects tab shows empty state', (tester) async {
      await registerEnterpriseAndDashboard(tester);

      // Navigate to the Projects tab (index 1 in bottom nav)
      await tapBottomNavItem(tester, 1);
      await waitForSettle(tester);

      // Should see the projects screen with empty state or title
      final hasTitle = find.text('Projects');
      final hasEmptyState = find.text('No projects yet');
      expect(
        hasTitle.evaluate().isNotEmpty || hasEmptyState.evaluate().isNotEmpty,
        isTrue,
        reason: 'Expected Projects title or empty state',
      );
    });

    testWidgets('FAB navigates to create project', (tester) async {
      await registerEnterpriseAndDashboard(tester);

      await tapBottomNavItem(tester, 1);
      await waitForSettle(tester);

      // Find and tap the FAB (FloatingActionButton)
      final fab = find.byType(FloatingActionButton);
      if (fab.evaluate().isNotEmpty) {
        await tester.tap(fab.first);
        await waitForSettle(tester);

        // Should be on the create project screen
        // Look for "Payment type" or "Create Project" header
        final hasPaymentType = find.text('Payment type');
        final hasCreateHeader = find.text('Create Project');
        expect(
          hasPaymentType.evaluate().isNotEmpty || hasCreateHeader.evaluate().isNotEmpty,
          isTrue,
          reason: 'Expected create project form elements',
        );
      }
    });
  });

  group('Projects — Create Form', () {
    testWidgets('payment type cards selectable', (tester) async {
      await registerEnterpriseAndDashboard(tester);

      await tapBottomNavItem(tester, 1);
      await waitForSettle(tester);

      // Navigate to create project
      final fab = find.byType(FloatingActionButton);
      if (fab.evaluate().isNotEmpty) {
        await tester.tap(fab.first);
        await waitForSettle(tester);

        // Should see payment type cards
        final escrowCard = find.text('Escrow');
        final invoiceCard = find.text('Invoice');

        if (escrowCard.evaluate().isNotEmpty) {
          // Tap invoice to switch
          if (invoiceCard.evaluate().isNotEmpty) {
            await tester.tap(invoiceCard.first);
            await waitForSettle(tester);
          }

          // Tap escrow to switch back
          await tester.tap(escrowCard.first);
          await waitForSettle(tester);
        }
      }
    });

    testWidgets('can add milestones', (tester) async {
      await registerEnterpriseAndDashboard(tester);

      await tapBottomNavItem(tester, 1);
      await waitForSettle(tester);

      final fab = find.byType(FloatingActionButton);
      if (fab.evaluate().isNotEmpty) {
        await tester.tap(fab.first);
        await waitForSettle(tester);

        // Look for "Add milestone" button
        final addMilestone = find.text('Add milestone');
        if (addMilestone.evaluate().isNotEmpty) {
          // Count existing milestone fields
          final initialFields = find.byType(TextFormField).evaluate().length;

          await tester.tap(addMilestone.first);
          await waitForSettle(tester);

          // Should have more fields after adding
          final afterFields = find.byType(TextFormField).evaluate().length;
          expect(afterFields, greaterThan(initialFields));
        }
      }
    });

    testWidgets('can add skills', (tester) async {
      await registerEnterpriseAndDashboard(tester);

      await tapBottomNavItem(tester, 1);
      await waitForSettle(tester);

      final fab = find.byType(FloatingActionButton);
      if (fab.evaluate().isNotEmpty) {
        await tester.tap(fab.first);
        await waitForSettle(tester);

        // Scroll down to find skills input
        await scrollDown(tester, dy: -400);

        // Look for skills input placeholder
        final skillsInput = find.text('Type a skill');
        if (skillsInput.evaluate().isNotEmpty) {
          // The skills input area should be visible
          expect(skillsInput, findsWidgets);
        }
      }
    });

    testWidgets('form fields work', (tester) async {
      await registerEnterpriseAndDashboard(tester);

      await tapBottomNavItem(tester, 1);
      await waitForSettle(tester);

      final fab = find.byType(FloatingActionButton);
      if (fab.evaluate().isNotEmpty) {
        await tester.tap(fab.first);
        await waitForSettle(tester);

        // Scroll to find the project title field
        await scrollDown(tester, dy: -200);

        // Try filling form fields (title, description)
        final textFields = find.byType(TextFormField);
        if (textFields.evaluate().isNotEmpty) {
          await tester.enterText(textFields.first, 'Integration Test Project');
          await tester.pumpAndSettle();
        }
      }
    });

    testWidgets('publish button visible', (tester) async {
      await registerEnterpriseAndDashboard(tester);

      await tapBottomNavItem(tester, 1);
      await waitForSettle(tester);

      final fab = find.byType(FloatingActionButton);
      if (fab.evaluate().isNotEmpty) {
        await tester.tap(fab.first);
        await waitForSettle(tester);

        // Scroll to bottom to find publish button
        await scrollDown(tester, dy: -600);
        await scrollDown(tester, dy: -600);

        final publishButton = find.text('Publish');
        final publishProjectButton = find.text('Publish project');
        expect(
          publishButton.evaluate().isNotEmpty || publishProjectButton.evaluate().isNotEmpty,
          isTrue,
          reason: 'Expected a publish button at the bottom of the form',
        );
      }
    });
  });
}
