import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/message_entity.dart';
import 'package:marketplace_mobile/features/messaging/presentation/widgets/chat/bubbles/text_message_bubble.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

MessageEntity _msg({
  String content = 'Hello',
  String status = 'sent',
  bool isEdited = false,
  ReplyToInfo? replyTo,
}) {
  return MessageEntity(
    id: 'm1',
    conversationId: 'c',
    senderId: 'user',
    type: 'text',
    content: content,
    status: status,
    editedAt: isEdited ? '2026-05-01T11:00:00Z' : null,
    replyTo: replyTo,
    seq: 1,
    createdAt: '2026-05-01T10:00:00Z',
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
      home: Scaffold(body: child),
    );

void main() {
  testWidgets('renders the message content', (tester) async {
    await tester.pumpWidget(
      _wrap(TextMessageBubble(message: _msg(content: 'Hi'), isOwn: true)),
    );
    expect(find.text('Hi'), findsOneWidget);
  });

  testWidgets('renders the (edited) tag when message is edited',
      (tester) async {
    await tester.pumpWidget(
      _wrap(TextMessageBubble(message: _msg(isEdited: true), isOwn: true)),
    );
    await tester.pumpAndSettle();
    final l10n = await AppLocalizations.delegate.load(const Locale('en'));
    expect(find.text('(${l10n.messagingEdited})'), findsOneWidget);
  });

  testWidgets('renders blue read marks for own read messages',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        TextMessageBubble(
          message: _msg(status: 'read'),
          isOwn: true,
        ),
      ),
    );
    expect(find.byIcon(Icons.done_all), findsOneWidget);
  });

  testWidgets('renders single check for own sent messages', (tester) async {
    await tester.pumpWidget(
      _wrap(
        TextMessageBubble(
          message: _msg(status: 'sent'),
          isOwn: true,
        ),
      ),
    );
    expect(find.byIcon(Icons.check), findsOneWidget);
  });

  testWidgets('does not render status icons for received messages',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        TextMessageBubble(
          message: _msg(status: 'sent'),
          isOwn: false,
        ),
      ),
    );
    expect(find.byIcon(Icons.check), findsNothing);
  });

  testWidgets('renders reply preview when replyTo is set', (tester) async {
    await tester.pumpWidget(
      _wrap(
        TextMessageBubble(
          message: _msg(
            replyTo: const ReplyToInfo(
              id: 'r1',
              senderId: 'u2',
              content: 'previous message',
              type: 'text',
            ),
          ),
          isOwn: true,
        ),
      ),
    );
    expect(find.text('previous message'), findsOneWidget);
  });

  testWidgets('does not show context menu when no callbacks are set',
      (tester) async {
    await tester.pumpWidget(
      _wrap(TextMessageBubble(message: _msg(), isOwn: true)),
    );
    final gesture = tester.widget<GestureDetector>(find.byType(GestureDetector));
    expect(gesture.onLongPress, isNull);
  });

  testWidgets('attaches long-press handler when callbacks are provided',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        TextMessageBubble(
          message: _msg(),
          isOwn: true,
          onReply: () {},
        ),
      ),
    );
    final gesture = tester.widget<GestureDetector>(find.byType(GestureDetector));
    expect(gesture.onLongPress, isNotNull);
  });
}
