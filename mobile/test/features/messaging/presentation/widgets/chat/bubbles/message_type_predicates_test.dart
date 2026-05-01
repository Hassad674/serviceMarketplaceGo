import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/messaging/presentation/widgets/chat/bubbles/message_type_predicates.dart';

void main() {
  group('isProposalCardType', () {
    test('returns true for proposal-card variants', () {
      expect(isProposalCardType('proposal_sent'), isTrue);
      expect(isProposalCardType('proposal_modified'), isTrue);
      expect(isProposalCardType('proposal_payment_requested'), isTrue);
    });

    test('returns false for other types', () {
      expect(isProposalCardType('proposal_accepted'), isFalse);
      expect(isProposalCardType('text'), isFalse);
      expect(isProposalCardType('evaluation_request'), isFalse);
    });
  });

  group('isSystemMessageType', () {
    test('returns true for proposal lifecycle pills', () {
      expect(isSystemMessageType('proposal_accepted'), isTrue);
      expect(isSystemMessageType('proposal_declined'), isTrue);
      expect(isSystemMessageType('proposal_paid'), isTrue);
      expect(isSystemMessageType('proposal_completion_requested'), isTrue);
      expect(isSystemMessageType('proposal_completed'), isTrue);
      expect(isSystemMessageType('proposal_completion_rejected'), isTrue);
    });

    test('returns true for call lifecycle events', () {
      expect(isSystemMessageType('call_ended'), isTrue);
      expect(isSystemMessageType('call_missed'), isTrue);
    });

    test('returns true for closed dispute lifecycle events', () {
      expect(isSystemMessageType('dispute_counter_accepted'), isTrue);
      expect(isSystemMessageType('dispute_escalated'), isTrue);
      expect(isSystemMessageType('dispute_cancelled'), isTrue);
      expect(isSystemMessageType('dispute_cancellation_refused'), isTrue);
    });

    test('returns false for proposal card types', () {
      expect(isSystemMessageType('proposal_sent'), isFalse);
      expect(isSystemMessageType('proposal_payment_requested'), isFalse);
    });

    test('returns false for evaluation_request', () {
      expect(isSystemMessageType('evaluation_request'), isFalse);
    });
  });

  group('isReferralSystemMessageType', () {
    test('returns true for the four referral message types', () {
      expect(isReferralSystemMessageType('referral_intro_sent'), isTrue);
      expect(
        isReferralSystemMessageType('referral_intro_negotiated'),
        isTrue,
      );
      expect(
        isReferralSystemMessageType('referral_intro_activated'),
        isTrue,
      );
      expect(isReferralSystemMessageType('referral_intro_closed'), isTrue);
    });

    test('returns false for unrelated types', () {
      expect(isReferralSystemMessageType('proposal_sent'), isFalse);
      expect(isReferralSystemMessageType('text'), isFalse);
    });
  });

  group('isDisputeCardType', () {
    test('returns true for dispute card variants', () {
      expect(isDisputeCardType('dispute_opened'), isTrue);
      expect(isDisputeCardType('dispute_counter_proposal'), isTrue);
      expect(isDisputeCardType('dispute_counter_rejected'), isTrue);
      expect(isDisputeCardType('dispute_resolved'), isTrue);
      expect(isDisputeCardType('dispute_auto_resolved'), isTrue);
      expect(isDisputeCardType('dispute_cancellation_requested'), isTrue);
    });

    test('returns false for closed dispute lifecycle events', () {
      // These are simple system pills, not full cards.
      expect(isDisputeCardType('dispute_counter_accepted'), isFalse);
      expect(isDisputeCardType('dispute_escalated'), isFalse);
    });
  });
}
