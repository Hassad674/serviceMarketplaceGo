import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/message_entity.dart';
import 'package:marketplace_mobile/features/messaging/presentation/widgets/chat/dialogs/edit_message_dialog.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

MessageEntity _msg({String content = 'old text'}) {
  return MessageEntity(
    id: 'm1',
    conversationId: 'conv-1',
    senderId: 'user-1',
    type: 'text',
    content: content,
    seq: 1,
    createdAt: DateTime.now().toIso8601String(),
  );
}

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
  testWidgets('renders message content prefilled', (tester) async {
    await tester.pumpWidget(
      _wrap(
        Scaffold(
          body: Builder(
            builder: (ctx) => TextButton(
              onPressed: () => showEditMessageDialog(
                context: ctx,
                message: _msg(content: 'Hello world'),
                onConfirm: (_) async {},
              ),
              child: const Text('Open'),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open'));
    await tester.pumpAndSettle();

    expect(find.byType(TextField), findsOneWidget);
    final textField = tester.widget<TextField>(find.byType(TextField));
    expect(textField.controller?.text, 'Hello world');
  });

  testWidgets('Save invokes onConfirm with the edited content',
      (tester) async {
    String? captured;
    await tester.pumpWidget(
      _wrap(
        Scaffold(
          body: Builder(
            builder: (ctx) => TextButton(
              onPressed: () => showEditMessageDialog(
                context: ctx,
                message: _msg(),
                onConfirm: (content) async {
                  captured = content;
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

    await tester.enterText(find.byType(TextField), '  new value  ');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(captured, 'new value'); // trimmed
  });

  testWidgets('Cancel dismisses without invoking onConfirm', (tester) async {
    var calls = 0;
    await tester.pumpWidget(
      _wrap(
        Scaffold(
          body: Builder(
            builder: (ctx) => TextButton(
              onPressed: () => showEditMessageDialog(
                context: ctx,
                message: _msg(),
                onConfirm: (_) async {
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
