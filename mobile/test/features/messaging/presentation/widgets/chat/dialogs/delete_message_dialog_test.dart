import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/presentation/widgets/chat/dialogs/delete_message_dialog.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child) => MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en')],
      locale: const Locale('en'),
      home: child,
    );

void main() {
  testWidgets('renders the delete confirmation message', (tester) async {
    await tester.pumpWidget(
      _wrap(
        Scaffold(
          body: Builder(
            builder: (ctx) => TextButton(
              onPressed: () => showDeleteMessageDialog(
                context: ctx,
                onConfirm: () async {},
              ),
              child: const Text('Open'),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();

    expect(find.byType(AlertDialog), findsOneWidget);
  });

  testWidgets('confirm invokes the callback', (tester) async {
    var calls = 0;
    await tester.pumpWidget(
      _wrap(
        Scaffold(
          body: Builder(
            builder: (ctx) => TextButton(
              onPressed: () => showDeleteMessageDialog(
                context: ctx,
                onConfirm: () async {
                  calls++;
                },
              ),
              child: const Text('Open'),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();

    // The destructive button is the ElevatedButton.
    await tester.tap(find.byType(ElevatedButton));
    await tester.pumpAndSettle();

    expect(calls, 1);
  });

  testWidgets('cancel does not invoke the callback', (tester) async {
    var calls = 0;
    await tester.pumpWidget(
      _wrap(
        Scaffold(
          body: Builder(
            builder: (ctx) => TextButton(
              onPressed: () => showDeleteMessageDialog(
                context: ctx,
                onConfirm: () async {
                  calls++;
                },
              ),
              child: const Text('Open'),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();

    expect(calls, 0);
  });
}
