import 'package:freezed_annotation/freezed_annotation.dart';

part 'missing_field.freezed.dart';

/// One field the backend considers missing for the billing profile to
/// be considered "complete".
///
/// [reason] is a machine-readable token (e.g. `"required"`,
/// `"invalid_format"`) the UI maps to localized copy — never displayed
/// verbatim. Treat it as an opaque enum-string.
@freezed
class MissingField with _$MissingField {
  const factory MissingField({
    required String field,
    required String reason,
  }) = _MissingField;
}
