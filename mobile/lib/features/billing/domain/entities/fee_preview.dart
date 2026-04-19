import 'package:freezed_annotation/freezed_annotation.dart';

part 'fee_preview.freezed.dart';

/// A single bracket of the platform fee grid.
///
/// [maxCents] is null for the open-ended last tier (e.g. "Plus de 1 000 €"),
/// which lets the UI render that tier without hardcoding sentinel values.
@freezed
class FeeTier with _$FeeTier {
  const factory FeeTier({
    required String label,
    required int? maxCents,
    required int feeCents,
  }) = _FeeTier;
}

/// The result of a fee preview computation for a given milestone amount.
///
/// Mirrors the backend `billing.Result` payload. The prestataire's role
/// (freelance vs. agency) is resolved server-side from the JWT and drives
/// which grid applies — clients never pick a grid themselves.
@freezed
class FeePreview with _$FeePreview {
  const factory FeePreview({
    required int amountCents,
    required int feeCents,
    required int netCents,
    required String role,
    required int activeTierIndex,
    required List<FeeTier> tiers,
  }) = _FeePreview;
}
