import 'package:freezed_annotation/freezed_annotation.dart';

part 'billing_profile.freezed.dart';

/// Billing-profile classification.
///
/// Mirrors the backend `invoicing.ProfileType` and the web type. The
/// backend rejects unknown values, so the client cannot invent new ones —
/// [fromJson] throws on anything outside the known set.
enum ProfileType {
  individual,
  business;

  static ProfileType fromJson(String raw) {
    switch (raw) {
      case 'individual':
        return ProfileType.individual;
      case 'business':
        return ProfileType.business;
      default:
        throw ArgumentError('Unknown ProfileType: $raw');
    }
  }

  String toJson() => name;
}

/// Snapshot of the organization's billing profile as returned by
/// `GET /api/v1/me/billing-profile`.
///
/// Field names mirror the web reference (`web/src/features/invoicing/types.ts`)
/// to keep the cross-platform contract aligned. Empty strings are valid
/// "not set yet" placeholders for free-text fields; nullable timestamps
/// represent "never happened" events (VIES validation, KYC sync).
@freezed
class BillingProfile with _$BillingProfile {
  const factory BillingProfile({
    required String organizationId,
    required ProfileType profileType,
    required String legalName,
    required String tradingName,
    required String legalForm,
    required String taxId,
    required String vatNumber,
    DateTime? vatValidatedAt,
    required String addressLine1,
    required String addressLine2,
    required String postalCode,
    required String city,
    required String country,
    required String invoicingEmail,
    DateTime? syncedFromKycAt,
  }) = _BillingProfile;
}

/// Mutation payload accepted by `PUT /api/v1/me/billing-profile`.
///
/// Wraps the subset of [BillingProfile] fields the user can edit
/// directly. `organization_id`, `vat_validated_at`, `synced_from_kyc_at`
/// are server-managed and intentionally absent.
@freezed
class UpdateBillingProfileInput with _$UpdateBillingProfileInput {
  const factory UpdateBillingProfileInput({
    required ProfileType profileType,
    required String legalName,
    required String tradingName,
    required String legalForm,
    required String taxId,
    required String vatNumber,
    required String addressLine1,
    required String addressLine2,
    required String postalCode,
    required String city,
    required String country,
    required String invoicingEmail,
  }) = _UpdateBillingProfileInput;
}
