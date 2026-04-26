import '../../domain/entities/missing_field.dart';

/// Maps a `MissingField` token returned by the backend to a short
/// French label suitable for the completion modal list.
///
/// Mirrors `web/src/features/invoicing/components/missing-fields-copy.ts`
/// verbatim so users get the same wording across web and mobile.
const Map<String, String> kMissingFieldLabels = {
  'legal_name': 'Raison sociale ou nom légal',
  'trading_name': 'Nom commercial',
  'legal_form': 'Forme juridique',
  'tax_id': 'Numéro SIRET ou identifiant fiscal',
  'vat_number': 'Numéro de TVA intracommunautaire',
  'address_line1': 'Adresse',
  'postal_code': 'Code postal',
  'city': 'Ville',
  'country': 'Pays',
  'invoicing_email': 'Email de facturation',
  'profile_type': 'Type de profil (particulier ou entreprise)',
};

/// Maps the `reason` token to a short qualifier — kept compact so the
/// modal stays scannable. The label is shown before the dash.
const Map<String, String> kMissingFieldReasonLabels = {
  'required': 'obligatoire',
  'invalid_format': 'format invalide',
  'not_validated': 'non validé',
};

/// Returns a French label for the given snake_case field token. Falls
/// back to the underscored token spaced out so a future backend field
/// is still readable until this map is updated.
String fieldLabel(String field) {
  return kMissingFieldLabels[field] ?? field.replaceAll('_', ' ');
}

/// Returns a French label for the given reason token, or null when the
/// token is unknown (so callers can omit the qualifier entirely rather
/// than printing the raw string).
String? reasonLabel(String reason) {
  return kMissingFieldReasonLabels[reason];
}

/// Composes the label rendered as one bullet inside the gate modal:
/// `"Code postal — obligatoire"` when both tokens are known,
/// `"Code postal"` when only the field is known.
String describeMissing(MissingField field) {
  final reason = reasonLabel(field.reason);
  final label = fieldLabel(field.field);
  return reason == null ? label : '$label — $reason';
}
