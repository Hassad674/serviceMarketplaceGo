import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/main.dart' as app;

// ---------------------------------------------------------------------------
// App initialization
// ---------------------------------------------------------------------------

/// Initialize the app for integration testing.
///
/// Calls [app.main] and waits for animations and async startup
/// (session restore, router initialization) to settle.
Future<void> initApp(WidgetTester tester) async {
  app.main();
  await tester.pumpAndSettle(const Duration(seconds: 5));
}

// ---------------------------------------------------------------------------
// Text field interaction
// ---------------------------------------------------------------------------

/// Fill a [TextFormField] or [TextField] at the given [index] among all
/// text fields currently rendered on screen.
///
/// Uses `find.byType(TextFormField)` first; falls back to `find.byType(TextField)`
/// when [useTextField] is true (needed for bottom sheets that use plain TextField).
Future<void> fillField(
  WidgetTester tester,
  int index,
  String text, {
  bool useTextField = false,
}) async {
  final finder = useTextField
      ? find.byType(TextField)
      : find.byType(TextFormField);
  expect(
    finder,
    findsWidgets,
    reason: 'Expected at least one text field on screen',
  );
  await tester.enterText(finder.at(index), text);
  await tester.pumpAndSettle();
}

// ---------------------------------------------------------------------------
// Tap helpers
// ---------------------------------------------------------------------------

/// Tap the first widget whose text content matches [text].
Future<void> tapText(WidgetTester tester, String text) async {
  final finder = find.text(text);
  expect(finder, findsWidgets, reason: 'Expected to find "$text" on screen');
  await tester.tap(finder.first);
  await tester.pumpAndSettle(const Duration(seconds: 2));
}

/// Tap the first widget of type [T] found on screen.
Future<void> tapByType<T extends Widget>(WidgetTester tester) async {
  final finder = find.byType(T);
  expect(finder, findsWidgets, reason: 'Expected to find ${T.toString()} on screen');
  await tester.tap(finder.first);
  await tester.pumpAndSettle();
}

/// Tap an [Icon] by its [IconData] value.
Future<void> tapIcon(WidgetTester tester, IconData icon) async {
  final finder = find.byIcon(icon);
  expect(finder, findsWidgets, reason: 'Expected to find icon $icon on screen');
  await tester.tap(finder.first);
  await tester.pumpAndSettle(const Duration(seconds: 2));
}

// ---------------------------------------------------------------------------
// Navigation helpers
// ---------------------------------------------------------------------------

/// Tap the bottom navigation bar item at [index] (0-based).
///
/// The shell uses [NavigationBar] with [NavigationDestination] children.
Future<void> tapBottomNavItem(WidgetTester tester, int index) async {
  final destinations = find.byType(NavigationDestination);
  expect(
    destinations,
    findsNWidgets(4),
    reason: 'Expected 4 bottom navigation items',
  );
  await tester.tap(destinations.at(index));
  await tester.pumpAndSettle(const Duration(seconds: 2));
}

// ---------------------------------------------------------------------------
// Wait helpers
// ---------------------------------------------------------------------------

/// Wait for the app to settle with a configurable timeout.
Future<void> waitForSettle(
  WidgetTester tester, {
  int seconds = 5,
}) async {
  await tester.pumpAndSettle(Duration(seconds: seconds));
}

// ---------------------------------------------------------------------------
// Assertion helpers
// ---------------------------------------------------------------------------

/// Assert that [text] is visible on the current screen.
void expectText(String text) {
  expect(
    find.text(text),
    findsWidgets,
    reason: 'Expected to find "$text" on screen',
  );
}

/// Assert that [text] is NOT visible on the current screen.
void expectNoText(String text) {
  expect(
    find.text(text),
    findsNothing,
    reason: 'Expected "$text" to NOT be on screen',
  );
}

/// Assert that a widget of type [T] exists on screen.
void expectWidget<T extends Widget>() {
  expect(
    find.byType(T),
    findsWidgets,
    reason: 'Expected to find ${T.toString()} on screen',
  );
}

/// Assert that exactly [count] widgets of type [T] exist.
void expectWidgetCount<T extends Widget>(int count) {
  expect(
    find.byType(T),
    findsNWidgets(count),
    reason: 'Expected $count ${T.toString()} widgets on screen',
  );
}

/// Assert that a widget containing [icon] is visible.
void expectIcon(IconData icon) {
  expect(
    find.byIcon(icon),
    findsWidgets,
    reason: 'Expected to find icon $icon on screen',
  );
}

// ---------------------------------------------------------------------------
// Scroll helpers
// ---------------------------------------------------------------------------

/// Scroll down on the first [SingleChildScrollView] or [ListView] found.
Future<void> scrollDown(WidgetTester tester, {double dy = -300}) async {
  final scrollable = find.byType(Scrollable).first;
  await tester.drag(scrollable, Offset(0, dy));
  await tester.pumpAndSettle();
}

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

/// Generate a unique email address for test isolation.
///
/// Uses the current timestamp in milliseconds to ensure no collisions
/// between test runs hitting the same backend.
String uniqueEmail() =>
    'test-${DateTime.now().millisecondsSinceEpoch}@integration-test.com';

/// Standard password that meets backend requirements:
/// minimum 8 characters, uppercase, lowercase, digit, special character.
const testPassword = 'TestPass123!';
