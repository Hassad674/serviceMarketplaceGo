import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/message_entity.dart';
import 'package:marketplace_mobile/features/messaging/presentation/widgets/chat/bubbles/reply_preview_widget.dart';
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
      home: Scaffold(body: child),
    );

void main() {
  testWidgets('renders the original message text', (tester) async {
    await tester.pumpWidget(
      _wrap(
        const ReplyPreviewWidget(
          replyTo: ReplyToInfo(
            id: 'r',
            senderId: 'u',
            content: 'short text',
            type: 'text',
          ),
          isOwn: false,
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('short text'), findsOneWidget);
  });

  testWidgets('truncates long content', (tester) async {
    final longContent = 'a' * 60; // > 50 chars
    await tester.pumpWidget(
      _wrap(
        ReplyPreviewWidget(
          replyTo: ReplyToInfo(
            id: 'r',
            senderId: 'u',
            content: longContent,
            type: 'text',
          ),
          isOwn: false,
        ),
      ),
    );
    await tester.pumpAndSettle();
    final expected = '${'a' * 50}...';
    expect(find.text(expected), findsOneWidget);
  });

  testWidgets('renders "deleted" placeholder when content is empty',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const ReplyPreviewWidget(
          replyTo: ReplyToInfo(
            id: 'r',
            senderId: 'u',
            content: '',
            type: 'text',
          ),
          isOwn: false,
        ),
      ),
    );
    await tester.pumpAndSettle();
    final l10n = await AppLocalizations.delegate.load(const Locale('en'));
    expect(find.text(l10n.messagingDeleted), findsOneWidget);
  });

  testWidgets('uses white-tinted background when isOwn=true',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const ReplyPreviewWidget(
          replyTo: ReplyToInfo(
            id: 'r',
            senderId: 'u',
            content: 'hi',
            type: 'text',
          ),
          isOwn: true,
        ),
      ),
    );
    await tester.pumpAndSettle();
    final container = tester.widget<Container>(find.byType(Container));
    final decoration = container.decoration as BoxDecoration?;
    expect(decoration?.color, Colors.white.withValues(alpha: 0.15));
  });
}
