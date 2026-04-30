import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:marketplace_mobile/core/utils/permissions.dart';
import 'package:marketplace_mobile/features/proposal/domain/entities/proposal_entity.dart';
import 'package:marketplace_mobile/features/proposal/presentation/widgets/proposal_action_buttons.dart';
import 'package:marketplace_mobile/features/proposal/types/proposal.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

ProposalEntity _proposal({
  String status = 'pending',
  String senderId = 'sender',
  String clientId = 'client',
  String providerId = 'provider',
  List<MilestoneEntity> milestones = const [],
  int? currentSeq,
  String? activeDispute,
}) {
  return ProposalEntity(
    id: 'p1',
    senderId: senderId,
    recipientId: 'recipient',
    conversationId: 'c1',
    title: 'X',
    description: '',
    amount: 1000,
    deadline: null,
    status: status,
    version: 1,
    documents: const [],
    createdAt: '2026-04-30T12:00:00Z',
    clientId: clientId,
    providerId: providerId,
    paymentMode: 'one_time',
    milestones: milestones,
    currentMilestoneSequence: currentSeq,
    activeDisputeId: activeDispute,
    lastDisputeId: null,
  );
}

MilestoneEntity _milestone({
  String status = 'pending_funding',
  int sequence = 1,
}) =>
    MilestoneEntity(
      id: 'm1',
      sequence: sequence,
      title: 'M1',
      description: '',
      amount: 100,
      status: status,
      version: 1,
    );


Widget _wrap(
  Widget child, {
  bool canRespond = true,
}) {
  final router = GoRouter(
    routes: [
      GoRoute(path: '/', builder: (_, __) => Scaffold(body: child)),
    ],
  );
  return ProviderScope(
    overrides: [
      hasPermissionProvider(OrgPermission.proposalsRespond)
          .overrideWith((ref) => canRespond),
    ],
    child: MaterialApp.router(
      routerConfig: router,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      locale: const Locale('en'),
    ),
  );
}

void main() {
  group('ProposalActionButtons - pending', () {
    testWidgets('recipient (not own) shows Accept + Decline + Modify',
        (tester) async {
      final p = _proposal(senderId: 'someone-else');
      await tester.pumpWidget(
        _wrap(
          ProposalActionButtons(
            proposal: p,
            isOwn: false,
            status: ProposalStatus.pending,
            currentUserId: 'me',
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(ElevatedButton), findsOneWidget); // Accept
      expect(find.byType(OutlinedButton), findsAtLeastNWidgets(2)); // Decline + Modify
    });

    testWidgets('canRespond=false hides everything', (tester) async {
      final p = _proposal();
      await tester.pumpWidget(
        _wrap(
          ProposalActionButtons(
            proposal: p,
            isOwn: false,
            status: ProposalStatus.pending,
            currentUserId: 'me',
          ),
          canRespond: false,
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(ElevatedButton), findsNothing);
      expect(find.byType(OutlinedButton), findsNothing);
    });

    testWidgets('isOwn=true on pending hides actions', (tester) async {
      final p = _proposal();
      await tester.pumpWidget(
        _wrap(
          ProposalActionButtons(
            proposal: p,
            isOwn: true,
            status: ProposalStatus.pending,
            currentUserId: 'me',
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(ElevatedButton), findsNothing);
    });
  });

  group('ProposalActionButtons - accepted (client funding)', () {
    testWidgets('client viewer sees Pay Now button', (tester) async {
      final p = _proposal(status: 'accepted', clientId: 'me-client');
      await tester.pumpWidget(
        _wrap(
          ProposalActionButtons(
            proposal: p,
            isOwn: false,
            status: ProposalStatus.accepted,
            currentUserId: 'me-client',
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byIcon(Icons.payment_outlined), findsOneWidget);
    });

    testWidgets('non-client viewer sees nothing', (tester) async {
      final p = _proposal(status: 'accepted');
      await tester.pumpWidget(
        _wrap(
          ProposalActionButtons(
            proposal: p,
            isOwn: false,
            status: ProposalStatus.accepted,
            currentUserId: 'someone-else',
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byIcon(Icons.payment_outlined), findsNothing);
    });
  });

  group('ProposalActionButtons - active', () {
    testWidgets('client + pending_funding milestone → Pay Now',
        (tester) async {
      final p = _proposal(
        status: 'active',
        clientId: 'me',
        milestones: [_milestone(status: 'pending_funding')],
        currentSeq: 1,
      );
      await tester.pumpWidget(
        _wrap(
          ProposalActionButtons(
            proposal: p,
            isOwn: false,
            status: ProposalStatus.active,
            currentUserId: 'me',
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byIcon(Icons.payment_outlined), findsOneWidget);
    });

    testWidgets('provider + funded milestone → Submit work',
        (tester) async {
      final p = _proposal(
        status: 'active',
        providerId: 'me',
        milestones: [_milestone(status: 'funded')],
        currentSeq: 1,
      );
      await tester.pumpWidget(
        _wrap(
          ProposalActionButtons(
            proposal: p,
            isOwn: false,
            status: ProposalStatus.active,
            currentUserId: 'me',
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byIcon(Icons.check_circle_outline), findsOneWidget);
    });

    testWidgets('active state with no current milestone hides actions',
        (tester) async {
      final p = _proposal(status: 'active');
      await tester.pumpWidget(
        _wrap(
          ProposalActionButtons(
            proposal: p,
            isOwn: false,
            status: ProposalStatus.active,
            currentUserId: 'me',
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(ElevatedButton), findsNothing);
    });
  });

  group('ProposalActionButtons - completionRequested', () {
    testWidgets('client viewer sees Approve + Request revisions',
        (tester) async {
      final p = _proposal(
        status: 'completion_requested',
        clientId: 'me',
        milestones: [_milestone(status: 'submitted')],
        currentSeq: 1,
      );
      await tester.pumpWidget(
        _wrap(
          ProposalActionButtons(
            proposal: p,
            isOwn: false,
            status: ProposalStatus.completionRequested,
            currentUserId: 'me',
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byIcon(Icons.check_circle_outline), findsOneWidget);
      expect(find.byIcon(Icons.undo_outlined), findsOneWidget);
    });
  });
}
