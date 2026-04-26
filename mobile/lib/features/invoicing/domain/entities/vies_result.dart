import 'package:freezed_annotation/freezed_annotation.dart';

part 'vies_result.freezed.dart';

/// VIES (EU VAT validation) round-trip result returned by
/// `POST /api/v1/me/billing-profile/validate-vat`.
///
/// On a successful lookup [valid] is true and [registeredName] holds the
/// legal name VIES has on file for the queried VAT number. On a failed
/// lookup [valid] is false and [registeredName] is empty — callers
/// should surface a UI hint without overwriting the user-entered legal
/// name.
@freezed
class VIESResult with _$VIESResult {
  const factory VIESResult({
    required bool valid,
    required String registeredName,
    required DateTime checkedAt,
  }) = _VIESResult;
}
