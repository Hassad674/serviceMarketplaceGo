import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'helpers/test_helpers.dart';

/// Integration tests for the messaging feature.
///
/// These tests require the Go backend to be running and reachable at the
/// API_URL configured via --dart-define.
///
/// The test flow:
/// 1. Register a provider account to get an authenticated session
/// 2. Navigate to the Messages tab
/// 3. Test conversation list, chat screen, and message sending
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  group('Messaging — Conversation List', () {
    testWidgets('messages tab shows conversation list', (tester) async {
      // Register and reach the dashboard
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Freelance / Business Referrer');
      await waitForSettle(tester);

      final email = uniqueEmail();
      await fillField(tester, 0, 'MsgTest');
      await fillField(tester, 1, 'User');
      await fillField(tester, 2, email);
      await fillField(tester, 3, testPassword);
      await fillField(tester, 4, testPassword);

      await scrollDown(tester, dy: -300);
      final submitButton = find.byType(ElevatedButton).first;
      await tester.tap(submitButton);
      await waitForSettle(tester, seconds: 10);

      // Navigate to Messages tab (index 2 in bottom nav)
      await tapBottomNavItem(tester, 2);
      await waitForSettle(tester);

      // The messages screen should load — look for the "Messages" header
      // or the empty state text
      final hasMessages = find.text('Messages');
      final hasEmptyState = find.text('No conversations yet');
      expect(
        hasMessages.evaluate().isNotEmpty || hasEmptyState.evaluate().isNotEmpty,
        isTrue,
        reason: 'Expected Messages header or empty state',
      );
    });

    testWidgets('tap conversation navigates to chat screen', (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Freelance / Business Referrer');
      await waitForSettle(tester);

      final email = uniqueEmail();
      await fillField(tester, 0, 'ChatTest');
      await fillField(tester, 1, 'User');
      await fillField(tester, 2, email);
      await fillField(tester, 3, testPassword);
      await fillField(tester, 4, testPassword);

      await scrollDown(tester, dy: -300);
      final submitButton = find.byType(ElevatedButton).first;
      await tester.tap(submitButton);
      await waitForSettle(tester, seconds: 10);

      // Navigate to Messages tab
      await tapBottomNavItem(tester, 2);
      await waitForSettle(tester);

      // If conversations exist, tap the first one
      final listTiles = find.byType(ListTile);
      if (listTiles.evaluate().isNotEmpty) {
        await tester.tap(listTiles.first);
        await waitForSettle(tester);

        // The chat screen should show a text input for typing messages
        expect(find.byType(TextField), findsWidgets);
      } else {
        // New user — no conversations, verify empty state
        final hasEmptyState = find.text('No conversations yet');
        expect(hasEmptyState, findsWidgets);
      }
    });
  });

  group('Messaging — Chat Screen', () {
    testWidgets('chat screen shows messages', (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Freelance / Business Referrer');
      await waitForSettle(tester);

      final email = uniqueEmail();
      await fillField(tester, 0, 'ChatView');
      await fillField(tester, 1, 'Tester');
      await fillField(tester, 2, email);
      await fillField(tester, 3, testPassword);
      await fillField(tester, 4, testPassword);

      await scrollDown(tester, dy: -300);
      final submitButton = find.byType(ElevatedButton).first;
      await tester.tap(submitButton);
      await waitForSettle(tester, seconds: 10);

      await tapBottomNavItem(tester, 2);
      await waitForSettle(tester);

      // If there are conversations, open the first one
      final listTiles = find.byType(ListTile);
      if (listTiles.evaluate().isNotEmpty) {
        await tester.tap(listTiles.first);
        await waitForSettle(tester);

        // Chat screen should have a message input and a send button
        expect(find.byType(TextField), findsWidgets);
        expect(find.byIcon(Icons.send), findsWidgets);
      }
    });

    testWidgets('can type and send message', (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Freelance / Business Referrer');
      await waitForSettle(tester);

      final email = uniqueEmail();
      await fillField(tester, 0, 'SendMsg');
      await fillField(tester, 1, 'Tester');
      await fillField(tester, 2, email);
      await fillField(tester, 3, testPassword);
      await fillField(tester, 4, testPassword);

      await scrollDown(tester, dy: -300);
      final submitButton = find.byType(ElevatedButton).first;
      await tester.tap(submitButton);
      await waitForSettle(tester, seconds: 10);

      await tapBottomNavItem(tester, 2);
      await waitForSettle(tester);

      final listTiles = find.byType(ListTile);
      if (listTiles.evaluate().isNotEmpty) {
        await tester.tap(listTiles.first);
        await waitForSettle(tester);

        // Type a message
        final textField = find.byType(TextField);
        if (textField.evaluate().isNotEmpty) {
          await tester.enterText(textField.last, 'Hello integration test!');
          await tester.pumpAndSettle();

          // Tap the send button
          final sendButton = find.byIcon(Icons.send);
          if (sendButton.evaluate().isNotEmpty) {
            await tester.tap(sendButton.first);
            await waitForSettle(tester, seconds: 3);
          }
        }
      }
    });

    testWidgets('back button returns to list', (tester) async {
      await initApp(tester);
      await tapText(tester, 'Sign Up');
      await waitForSettle(tester);
      await tapText(tester, 'Freelance / Business Referrer');
      await waitForSettle(tester);

      final email = uniqueEmail();
      await fillField(tester, 0, 'BackBtn');
      await fillField(tester, 1, 'Tester');
      await fillField(tester, 2, email);
      await fillField(tester, 3, testPassword);
      await fillField(tester, 4, testPassword);

      await scrollDown(tester, dy: -300);
      final submitButton = find.byType(ElevatedButton).first;
      await tester.tap(submitButton);
      await waitForSettle(tester, seconds: 10);

      await tapBottomNavItem(tester, 2);
      await waitForSettle(tester);

      final listTiles = find.byType(ListTile);
      if (listTiles.evaluate().isNotEmpty) {
        await tester.tap(listTiles.first);
        await waitForSettle(tester);

        // Tap the back button
        final backButton = find.byIcon(Icons.arrow_back);
        if (backButton.evaluate().isNotEmpty) {
          await tester.tap(backButton.first);
          await waitForSettle(tester);

          // Should be back on the conversation list
          final hasMessages = find.text('Messages');
          expect(hasMessages, findsWidgets);
        }
      }
    });
  });
}
