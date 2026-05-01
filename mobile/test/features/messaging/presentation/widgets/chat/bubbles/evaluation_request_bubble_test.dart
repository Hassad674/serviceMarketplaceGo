import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/message_entity.dart';
import 'package:marketplace_mobile/features/messaging/presentation/widgets/chat/bubbles/evaluation_request_bubble.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

MessageEntity _msg({Map<String, dynamic>? metadata}) {
  return MessageEntity(
    id: 'm',
    conversationId: 'conv',
    senderId: 'user',
    type: 'evaluation_request',
    content: '',
    metadata: metadata,
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
      home: Scaffold(body: child),
    );

void main() {
  testWidgets('renders the star icon and review CTA', (tester) async {
    await tester.pumpWidget(
      _wrap(
        EvaluationRequestBubble(
          message: _msg(
            metadata: {
              'proposal_id': 'P1',
              'proposal_title': 'Logo design',
              'proposal_client_organization_id': 'org-client',
              'proposal_provider_organization_id': 'org-provider',
            },
          ),
          onReview: (_, __, ___, ____) {},
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.star_outline), findsOneWidget);
    expect(find.byType(FilledButton), findsOneWidget);
  });

  testWidgets('disables CTA when org ids are missing (legacy message)',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        EvaluationRequestBubble(
          message: _msg(metadata: {'proposal_id': 'P1'}),
          onReview: (_, __, ___, ____) {},
        ),
      ),
    );
    await tester.pumpAndSettle();
    final button = tester.widget<FilledButton>(find.byType(FilledButton));
    expect(button.onPressed, isNull);
  });

  testWidgets('disables CTA when no onReview callback is provided',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        EvaluationRequestBubble(
          message: _msg(
            metadata: {
              'proposal_id': 'P1',
              'proposal_client_organization_id': 'org-client',
              'proposal_provider_organization_id': 'org-provider',
            },
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final button = tester.widget<FilledButton>(find.byType(FilledButton));
    expect(button.onPressed, isNull);
  });

  testWidgets('invokes onReview with the metadata when tapped',
      (tester) async {
    String? capturedId;
    String? capturedClientOrg;
    await tester.pumpWidget(
      _wrap(
        EvaluationRequestBubble(
          message: _msg(
            metadata: {
              'proposal_id': 'P1',
              'proposal_title': 'Logo',
              'proposal_client_organization_id': 'org-client',
              'proposal_provider_organization_id': 'org-provider',
            },
          ),
          onReview: (id, _, clientOrg, __) {
            capturedId = id;
            capturedClientOrg = clientOrg;
          },
        ),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byType(FilledButton));
    await tester.pumpAndSettle();
    expect(capturedId, 'P1');
    expect(capturedClientOrg, 'org-client');
  });
}
