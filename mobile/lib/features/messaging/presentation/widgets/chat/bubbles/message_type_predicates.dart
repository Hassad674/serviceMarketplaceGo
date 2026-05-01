// Predicates classifying message types into bubble variants.
//
// Centralized here so a new system message type only requires a
// single edit.

bool isProposalCardType(String type) {
  return type == 'proposal_sent' ||
      type == 'proposal_modified' ||
      type == 'proposal_payment_requested';
}

/// Returns true for system-level lifecycle events (proposals, calls,
/// disputes). Note: `evaluation_request` is handled separately with
/// a review button.
bool isSystemMessageType(String type) {
  return type == 'proposal_accepted' ||
      type == 'proposal_declined' ||
      type == 'proposal_paid' ||
      type == 'proposal_completion_requested' ||
      type == 'proposal_completed' ||
      type == 'proposal_completion_rejected' ||
      type == 'call_ended' ||
      type == 'call_missed' ||
      type == 'dispute_counter_accepted' ||
      type == 'dispute_escalated' ||
      type == 'dispute_cancelled' ||
      type == 'dispute_cancellation_refused';
}

/// Returns true for the four referral (apport d'affaires) system
/// message types posted by the Go backend. Each renders as an
/// interactive card via ReferralSystemMessageWidget.
bool isReferralSystemMessageType(String type) {
  return type == 'referral_intro_sent' ||
      type == 'referral_intro_negotiated' ||
      type == 'referral_intro_activated' ||
      type == 'referral_intro_closed';
}

/// Returns true for dispute messages that should render as a rich card.
/// dispute_resolved and dispute_auto_resolved both render a full
/// decision card with split + user share highlight + admin note;
/// the others use the simpler subtitle layout with a "View details"
/// button.
bool isDisputeCardType(String type) {
  return type == 'dispute_opened' ||
      type == 'dispute_counter_proposal' ||
      type == 'dispute_counter_rejected' ||
      type == 'dispute_resolved' ||
      type == 'dispute_auto_resolved' ||
      type == 'dispute_cancellation_requested';
}
