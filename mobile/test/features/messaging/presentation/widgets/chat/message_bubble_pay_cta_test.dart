import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/message_entity.dart';
import 'package:marketplace_mobile/features/messaging/presentation/widgets/chat/message_bubble.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

// Widget-level wiring for the `proposal_accepted` system message:
// MessageBubble must route it to SystemMessageBubble with the
// "Payer maintenant" CTA visible iff the viewer is the client AND
// status is still 'accepted'. Mirrors the web regression fix.

MessageEntity _accepted({
  required String clientId,
  String status = 'accepted',
  String proposalId = 'prop-1',
}) {
  return MessageEntity(
    id: 'm',
    conversationId: 'conv',
    senderId: 'someone',
    content: '',
    type: 'proposal_accepted',
    metadata: {
      'proposal_id': proposalId,
      'proposal_status': status,
      'proposal_client_id': clientId,
      'proposal_provider_id': 'provider-1',
    },
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
  group('MessageBubble — proposal_accepted Pay CTA wiring', () {
    testWidgets('shows CTA for the client when status=accepted',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          MessageBubble(
            message: _accepted(clientId: 'client-1'),
            isOwn: false,
            currentUserId: 'client-1',
            onPayProposal: (_) {},
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        find.byKey(const ValueKey('proposal-accepted-pay-cta')),
        findsOneWidget,
      );
    });

    testWidgets('CTA tap forwards the proposal id to onPayProposal',
        (tester) async {
      String? captured;
      await tester.pumpWidget(
        _wrap(
          MessageBubble(
            message: _accepted(
              clientId: 'client-1',
              proposalId: 'prop-42',
            ),
            isOwn: false,
            currentUserId: 'client-1',
            onPayProposal: (id) => captured = id,
          ),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(
        find.byKey(const ValueKey('proposal-accepted-pay-cta')),
      );
      await tester.pumpAndSettle();
      expect(captured, 'prop-42');
    });

    testWidgets('hides CTA when viewer is the provider', (tester) async {
      await tester.pumpWidget(
        _wrap(
          MessageBubble(
            message: _accepted(clientId: 'client-1'),
            isOwn: true,
            currentUserId: 'provider-1',
            onPayProposal: (_) {},
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        find.byKey(const ValueKey('proposal-accepted-pay-cta')),
        findsNothing,
      );
    });

    testWidgets('hides CTA when snapshot status has moved past accepted',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          MessageBubble(
            message: _accepted(clientId: 'client-1', status: 'paid'),
            isOwn: false,
            currentUserId: 'client-1',
            onPayProposal: (_) {},
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        find.byKey(const ValueKey('proposal-accepted-pay-cta')),
        findsNothing,
      );
    });

    testWidgets('hides CTA when onPayProposal is null', (tester) async {
      await tester.pumpWidget(
        _wrap(
          MessageBubble(
            message: _accepted(clientId: 'client-1'),
            isOwn: false,
            currentUserId: 'client-1',
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        find.byKey(const ValueKey('proposal-accepted-pay-cta')),
        findsNothing,
      );
    });
  });
}
