import '../../../../core/models/review.dart';

/// Returns the review direction for the current user relative to a proposal.
///
/// Since the team/org refactor, `proposal.client_id` and `proposal.provider_id`
/// are organization IDs (not user IDs). Reviews are attributed to the operator's
/// current organization, so the side is derived by comparing the user's org id
/// with the proposal's client and provider org ids.
///
/// Returns `null` when the organization is neither the client nor the provider
/// on the proposal — the caller should treat that as "not a participant" and
/// hide the review CTA.
String? deriveReviewSide({
  required String userOrganizationId,
  required String proposalClientOrgId,
  required String proposalProviderOrgId,
}) {
  if (userOrganizationId.isEmpty) return null;
  if (userOrganizationId == proposalClientOrgId) {
    return ReviewSide.clientToProvider;
  }
  if (userOrganizationId == proposalProviderOrgId) {
    return ReviewSide.providerToClient;
  }
  return null;
}
