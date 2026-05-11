// Unit tests for dashboardActionsProvider — verifies the aggregation
// rules surface the right rows AND, critically, that billing completion
// is NOT in the list (web parity + FIX-DASH regression — billing info
// is collected at withdrawal time, only KYC gates the payout flow).
//
// Each test stubs the upstream providers (auth, profile completion,
// conversations, projects, subscription) and asserts the action list.

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/dashboard/domain/dashboard_action.dart';
import 'package:marketplace_mobile/features/dashboard/presentation/providers/dashboard_actions_provider.dart';
import 'package:marketplace_mobile/features/messaging/presentation/providers/conversations_provider.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/conversation_entity.dart';
import 'package:marketplace_mobile/features/profile_completion/domain/entities/profile_completion_report.dart';
import 'package:marketplace_mobile/features/profile_completion/presentation/providers/profile_completion_providers.dart';
import 'package:marketplace_mobile/features/proposal/presentation/providers/proposal_provider.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/subscription.dart';
import 'package:marketplace_mobile/features/subscription/presentation/providers/subscription_providers.dart';

class _AuthStub extends StateNotifier<AuthState> implements AuthNotifier {
  _AuthStub(super.state);

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class _ConversationsStub extends StateNotifier<ConversationsState>
    implements ConversationsNotifier {
  _ConversationsStub(super.state);

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

/// A "complete enough" profile completion default so isolated KYC /
/// messaging tests aren't perturbed by a phantom `profile_incomplete`
/// row (the empty report defaults to percent=0).
const _completeProfile = ProfileCompletionReport(
  role: 'provider',
  persona: 'freelance',
  percent: 100,
  totalSections: 8,
  filledSections: 8,
  sections: [],
);

ProviderContainer _container({
  Map<String, dynamic>? user,
  Map<String, dynamic>? organization,
  ProfileCompletionReport? completion,
  List<ConversationEntity>? conversations,
  Subscription? subscription,
}) {
  return ProviderContainer(
    overrides: [
      authProvider.overrideWith(
        (ref) => _AuthStub(
          AuthState(
            status: AuthStatus.authenticated,
            user: user ?? const {'id': 'u1', 'role': 'provider'},
            organization: organization,
          ),
        ),
      ),
      profileCompletionProvider.overrideWith(
        (ref) async => completion ?? _completeProfile,
      ),
      conversationsProvider.overrideWith(
        (ref) => _ConversationsStub(
          ConversationsState(conversations: conversations ?? const []),
        ),
      ),
      projectsProvider.overrideWith((ref) async => const []),
      subscriptionProvider.overrideWith((ref) async => subscription),
    ],
  );
}

void main() {
  group('dashboardActionsProvider', () {
    test('returns an empty list when nothing needs attention', () async {
      final container = _container();
      addTearDown(container.dispose);

      // Let async providers resolve.
      await container.read(profileCompletionProvider.future);

      final actions = container.read(dashboardActionsProvider);
      expect(actions, isEmpty);
    });

    test('surfaces KYC restricted as the highest-severity row', () async {
      final container = _container(
        user: const {
          'id': 'u1',
          'role': 'provider',
          'kyc_status': 'restricted',
        },
      );
      addTearDown(container.dispose);
      await container.read(profileCompletionProvider.future);

      final actions = container.read(dashboardActionsProvider);
      expect(actions, hasLength(1));
      expect(actions.first.id, 'kyc_restricted');
      expect(actions.first.severity, DashboardActionSeverity.critical);
    });

    test('surfaces KYC pending as warning', () async {
      final container = _container(
        user: const {
          'id': 'u1',
          'role': 'provider',
          'kyc_status': 'pending',
        },
      );
      addTearDown(container.dispose);
      await container.read(profileCompletionProvider.future);

      final actions = container.read(dashboardActionsProvider);
      expect(actions, hasLength(1));
      expect(actions.first.id, 'kyc_pending');
      expect(actions.first.severity, DashboardActionSeverity.warning);
    });

    test('skips KYC rows when the user is enterprise', () async {
      final container = _container(
        user: const {
          'id': 'u1',
          'role': 'enterprise',
          'kyc_status': 'restricted',
        },
      );
      addTearDown(container.dispose);
      await container.read(profileCompletionProvider.future);

      final actions = container.read(dashboardActionsProvider);
      expect(actions.where((a) => a.id.startsWith('kyc_')), isEmpty);
    });

    test('surfaces profile completion when < 80%', () async {
      const report = ProfileCompletionReport(
        role: 'provider',
        persona: 'freelance',
        percent: 50,
        totalSections: 8,
        filledSections: 4,
        sections: [],
      );
      final container = _container(completion: report);
      addTearDown(container.dispose);
      await container.read(profileCompletionProvider.future);

      final actions = container.read(dashboardActionsProvider);
      final ids = actions.map((a) => a.id).toList();
      expect(ids, contains('profile_incomplete'));
    });

    test('does NOT surface profile completion when ≥ 80%', () async {
      const report = ProfileCompletionReport(
        role: 'provider',
        persona: 'freelance',
        percent: 85,
        totalSections: 8,
        filledSections: 7,
        sections: [],
      );
      final container = _container(completion: report);
      addTearDown(container.dispose);
      await container.read(profileCompletionProvider.future);

      final actions = container.read(dashboardActionsProvider);
      expect(actions.where((a) => a.id == 'profile_incomplete'), isEmpty);
    });

    test(
        'surfaces unread messages with critical severity once 5+ are waiting',
        () async {
      final conversations = <ConversationEntity>[
        for (var i = 0; i < 2; i++)
          ConversationEntity(
            id: 'c$i',
            otherUserId: 'u-other-$i',
            otherOrgId: 'o-$i',
            otherOrgName: 'Org $i',
            otherOrgType: 'agency',
            otherPhotoUrl: '',
            unreadCount: 3,
          ),
      ];
      final container = _container(conversations: conversations);
      addTearDown(container.dispose);
      await container.read(profileCompletionProvider.future);

      final actions = container.read(dashboardActionsProvider);
      final unread =
          actions.firstWhere((a) => a.id == 'messages_unread');
      expect(unread.severity, DashboardActionSeverity.critical);
      // The "X waiting" detail mirrors the total unread count.
      expect(unread.detail, '6 waiting');
    });

    test('REGRESSION: billing completeness is NEVER surfaced', () async {
      // The provider docstring says billing is collected at withdrawal
      // time (web parity) — only KYC gates the payout flow. This pins
      // the regression: even with a fully-loaded auth state, no action
      // id should contain 'billing' or 'billing_info'.
      final container = _container(
        user: const {
          'id': 'u1',
          'role': 'provider',
          'kyc_status': 'pending',
          'billing_info_complete': false,
        },
      );
      addTearDown(container.dispose);
      await container.read(profileCompletionProvider.future);

      final actions = container.read(dashboardActionsProvider);
      final billingMatches = actions.where(
        (a) => a.id.toLowerCase().contains('billing'),
      );
      expect(
        billingMatches,
        isEmpty,
        reason: 'Billing should never surface in dashboard actions — '
            'only KYC gates the payout flow.',
      );
    });

    test('actions are sorted critical → warning → info', () async {
      final conversations = <ConversationEntity>[
        const ConversationEntity(
          id: 'c1',
          otherUserId: 'u-x',
          otherOrgId: 'o-x',
          otherOrgName: 'Org X',
          otherOrgType: 'agency',
          otherPhotoUrl: '',
          unreadCount: 1, // warning severity (< 5)
        ),
      ];
      final container = _container(
        user: const {
          'id': 'u1',
          'role': 'provider',
          'kyc_status': 'restricted',
        },
        conversations: conversations,
      );
      addTearDown(container.dispose);
      await container.read(profileCompletionProvider.future);

      final actions = container.read(dashboardActionsProvider);
      // First row is the critical KYC restricted, second the warning.
      expect(actions.first.severity, DashboardActionSeverity.critical);
      expect(actions.last.severity, DashboardActionSeverity.warning);
    });
  });
}
