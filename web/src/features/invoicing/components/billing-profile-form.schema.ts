import { z } from "zod"
import { isEUCountry } from "./eu-countries"

// Validation schema for the editable billing profile form.
//
// Goals:
//   - Mirror the backend's `is_complete` rules so the user sees the
//     same gate the API will enforce when they hit "Save". The
//     server is still the source of truth — this schema only catches
//     trivial mistakes early (empty required fields, malformed
//     SIRET, malformed VAT).
//   - Conditional rules driven by `country`:
//       * FR: SIRET (14 digits) is required.
//       * EU + non-FR: VAT number format check is required when the
//         field is non-empty (the user can save without VAT and
//         validate it later via the VIES round-trip).
//       * Non-EU: no SIRET / VAT requirement.
//   - All optional-feeling fields (trading_name, address_line2,
//     legal_form when individual) stay strings so RHF watches them
//     without turning controlled inputs into uncontrolled ones.
//
// The schema accepts partial input shapes during typing — the
// `superRefine` block emits the conditional errors only after every
// base-shape constraint passes.

const SIRET_REGEX = /^\d{14}$/
// Generic VIES VAT shape: 2-letter country prefix + 2-12 alphanumerics.
const VAT_REGEX = /^[A-Z]{2}[A-Z0-9]{2,12}$/

export const profileTypeSchema = z.enum(["individual", "business"])

export const billingProfileFormSchema = z
  .object({
    profile_type: profileTypeSchema,
    legal_name: z
      .string()
      .trim()
      .min(1, "Le nom légal est requis."),
    trading_name: z.string(),
    legal_form: z.string(),
    tax_id: z.string(),
    vat_number: z.string(),
    address_line1: z.string().trim().min(1, "L'adresse est requise."),
    address_line2: z.string(),
    postal_code: z.string().trim().min(1, "Le code postal est requis."),
    city: z.string().trim().min(1, "La ville est requise."),
    country: z.string().trim().length(2, "Sélectionne ton pays de facturation."),
    invoicing_email: z.string(),
  })
  .superRefine((value, ctx) => {
    const country = value.country.toUpperCase()
    // FR-only: SIRET is mandatory and must be 14 digits.
    if (country === "FR") {
      const trimmed = value.tax_id.replace(/\s+/g, "")
      if (!SIRET_REGEX.test(trimmed)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["tax_id"],
          message: "Le SIRET doit contenir 14 chiffres (sans espace).",
        })
      }
    }
    // Business profile_type with no legal_form filled in: explicit
    // hint instead of a backend round-trip rejection.
    if (
      value.profile_type === "business" &&
      value.legal_form.trim().length === 0
    ) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["legal_form"],
        message: "La forme juridique est requise pour une entreprise.",
      })
    }
    // EU (non-FR): VAT format check when supplied. Empty stays valid
    // — the user can save and run VIES validation later.
    if (
      isEUCountry(country) &&
      country !== "FR" &&
      value.vat_number.trim().length > 0
    ) {
      const upper = value.vat_number.toUpperCase().replace(/\s+/g, "")
      if (!VAT_REGEX.test(upper)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["vat_number"],
          message:
            "Le numéro de TVA intracom doit commencer par 2 lettres + 2 à 12 caractères (ex. FR12345678901).",
        })
      }
    }
  })

export type BillingProfileFormValues = z.infer<typeof billingProfileFormSchema>
