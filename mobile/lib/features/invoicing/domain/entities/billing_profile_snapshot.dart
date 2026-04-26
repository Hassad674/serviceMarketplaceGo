import 'package:freezed_annotation/freezed_annotation.dart';

import 'billing_profile.dart';
import 'missing_field.dart';

part 'billing_profile_snapshot.freezed.dart';

/// Composite payload the backend returns on every billing-profile read /
/// write: the profile itself plus the gating signal used by wallet payout
/// and subscription endpoints.
///
/// `isComplete == false` <=> `missingFields` is non-empty. The boolean is
/// kept on the wire (and here) to avoid every consumer re-deriving the
/// same expression.
@freezed
class BillingProfileSnapshot with _$BillingProfileSnapshot {
  const factory BillingProfileSnapshot({
    required BillingProfile profile,
    required List<MissingField> missingFields,
    required bool isComplete,
  }) = _BillingProfileSnapshot;
}
