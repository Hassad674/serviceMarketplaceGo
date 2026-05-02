import type { MissingField } from "@/shared/types/billing-profile"

/**
 * Maps a `MissingField` token returned by the backend to a short
 * French label suitable for a list inside the completion modal.
 *
 * The backend ships `field` as a snake_case identifier; the modal
 * renders the label, NEVER the raw token. Unknown tokens fall back
 * to the field name with underscores replaced — so a future
 * field added on the backend is still readable until this map
 * is updated.
 */
const FIELD_LABELS: Record<string, string> = {
  legal_name: "Raison sociale ou nom légal",
  trading_name: "Nom commercial",
  legal_form: "Forme juridique",
  tax_id: "Numéro SIRET ou identifiant fiscal",
  vat_number: "Numéro de TVA intracommunautaire",
  address_line1: "Adresse",
  postal_code: "Code postal",
  city: "Ville",
  country: "Pays",
  invoicing_email: "Email de facturation",
  profile_type: "Type de profil (particulier ou entreprise)",
}

/**
 * Maps the `reason` token to a short qualifier — kept compact so
 * the modal stays scannable. The label is shown before the dash.
 */
const REASON_LABELS: Record<string, string> = {
  required: "obligatoire",
  invalid_format: "format invalide",
  not_validated: "non validé",
}

export function fieldLabel(field: string): string {
  return FIELD_LABELS[field] ?? field.replace(/_/g, " ")
}

export function reasonLabel(reason: string): string {
  return REASON_LABELS[reason] ?? reason.replace(/_/g, " ")
}

export function describeMissing(field: MissingField): string {
  const reason = REASON_LABELS[field.reason] ?? null
  const label = fieldLabel(field.field)
  return reason ? `${label} — ${reason}` : label
}
