import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/message_entity.dart';
import 'package:marketplace_mobile/features/messaging/presentation/widgets/chat/bubbles/system_message_bubble.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

MessageEntity _msg({required String type, String content = ''}) {
  return MessageEntity(
    id: 'm',
    conversationId: 'conv',
    senderId: 'user',
    type: type,
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
      home: Scaffold(body: Center(child: child)),
    );

void main() {
  testWidgets('renders the proposal_accepted icon', (tester) async {
    await tester.pumpWidget(
      _wrap(SystemMessageBubble(message: _msg(type: 'proposal_accepted'))),
    );
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.check_circle_outline), findsOneWidget);
  });

  testWidgets('renders the call_missed icon', (tester) async {
    await tester.pumpWidget(
      _wrap(SystemMessageBubble(message: _msg(type: 'call_missed'))),
    );
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.phone_missed_outlined), findsOneWidget);
  });

  testWidgets('falls back to info_outline for unknown types',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        SystemMessageBubble(
          message: _msg(type: 'mystery', content: 'noisy'),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.info_outline), findsOneWidget);
    expect(find.text('noisy'), findsOneWidget);
  });

  testWidgets('uses the centered pill layout', (tester) async {
    await tester.pumpWidget(
      _wrap(SystemMessageBubble(message: _msg(type: 'proposal_paid'))),
    );
    await tester.pumpAndSettle();
    expect(find.byType(Center), findsAtLeastNWidgets(1));
  });
}
