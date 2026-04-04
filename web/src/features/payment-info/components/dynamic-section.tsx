"use client"

import { useState, useEffect } from "react"
import { Upload, CheckCircle2, Loader2, AlertTriangle, XCircle } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { CountrySelect } from "./country-select"
import { UploadModal } from "@/shared/components/upload-modal"
import { useUploadIdentityDocument } from "../hooks/use-identity-documents"
import { isStateField, getStatesForCountry, hasStates } from "../lib/country-states"
import type { StateOption } from "../lib/country-states"
import type { FieldSection, FieldSpec } from "../api/payment-info-api"
import type { IdentityDocumentResponse } from "../api/identity-document-api"

interface DynamicSectionProps {
  section: FieldSection
  values: Record<string, string>
  onChange: (key: string, value: string) => void
  fieldErrors?: Record<string, string>
  fieldWarnings?: Record<string, string>
  documents?: IdentityDocumentResponse[]
  countryCode?: string
}

export function DynamicSection({
  section, values, onChange, fieldErrors, fieldWarnings, documents, countryCode,
}: DynamicSectionProps) {
  const t = useTranslations("paymentInfo")

  return (
    <section className="rounded-2xl border border-gray-100 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900">
      <h2 className="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
        {safeTranslate(t, section.title_key)}
      </h2>
      <div className="grid gap-4 sm:grid-cols-2">
        {section.fields.map((field) => (
          <DynamicField
            key={field.key}
            field={field}
            value={values[field.key] ?? ""}
            onChange={(v) => onChange(field.key, v)}
            error={fieldErrors?.[field.key]}
            warning={fieldWarnings?.[field.key]}
            documents={documents}
            countryCode={countryCode}
          />
        ))}
      </div>
    </section>
  )
}

interface DynamicFieldProps {
  field: FieldSpec
  value: string
  onChange: (value: string) => void
  error?: string
  warning?: string
  documents?: IdentityDocumentResponse[]
  countryCode?: string
}

function DynamicField({ field, value, onChange, error, warning, documents, countryCode }: DynamicFieldProps) {
  const t = useTranslations("paymentInfo")

  if (field.type === "document_upload") {
    return (
      <DocumentUploadField
        field={field}
        value={value}
        onChange={onChange}
        error={error}
        warning={warning}
        documents={documents}
      />
    )
  }

  const label = safeTranslate(t, field.label_key)

  // Render state/province fields as a dropdown when the country has known states
  if (isStateField(field.label_key, field.path) && countryCode && hasStates(countryCode)) {
    return (
      <StateSelectField
        field={field}
        value={value}
        onChange={onChange}
        label={label}
        error={error}
        countryCode={countryCode}
      />
    )
  }

  if (field.type === "select") {
    return <SelectField field={field} value={value} onChange={onChange} label={label} error={error} />
  }

  const isIban = field.key.includes("iban") || field.path.includes("iban")
  const hasError = !!error

  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
        {field.required && <span className="ml-0.5 text-red-500">*</span>}
      </label>
      <input
        type={inputType(field.type)}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={field.placeholder ?? ""}
        aria-invalid={hasError}
        className={cn(
          "h-10 w-full rounded-lg border bg-white px-3 text-sm shadow-xs transition-all duration-200",
          "placeholder:text-gray-400",
          "focus:outline-none",
          "dark:bg-gray-900 dark:text-gray-100 dark:placeholder:text-gray-500",
          hasError
            ? "border-red-500 ring-4 ring-red-500/10 dark:border-red-500"
            : "border-gray-200 focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 dark:border-gray-700",
        )}
      />
      {hasError && (
        <p className="mt-1 text-xs text-red-600 dark:text-red-400" role="alert">{error}</p>
      )}
      {isIban && !hasError && (
        <p className="mt-1.5 text-xs text-slate-500 dark:text-slate-400">
          {t("ibanHelp")}{" "}
          <a href="https://www.iban.com" target="_blank" rel="noopener noreferrer" className="text-rose-500 hover:underline">
            iban.com
          </a>
        </p>
      )}
    </div>
  )
}

function DocumentUploadField({ field, value, onChange, error, warning, documents }: DynamicFieldProps) {
  const t = useTranslations("paymentInfo")
  const uploadMutation = useUploadIdentityDocument()
  const [modalOpen, setModalOpen] = useState(false)
  const label = safeTranslate(t, field.label_key)
  const descKey = field.label_key + "Desc"
  const description = safeTranslateOptional(t, descKey)

  const category = field.path.startsWith("company") || field.path.startsWith("documents") ? "company" : "identity"
  const documentType = deriveDocumentType(field.path)

  // Find matching document from the existing docs list for status display
  const matchingDoc = documents?.find(
    (d) => d.category === category && d.document_type === documentType,
  )
  const docStatus = matchingDoc?.status

  // If value is "uploaded" (local state or from existing docs), suppress stale error
  const isUploaded = value === "uploaded"
  const effectiveError = isUploaded ? undefined : error

  async function handleUpload(file: File) {
    setModalOpen(false)
    uploadMutation.mutate(
      { file, category, documentType, side: "single" },
      { onSuccess: () => onChange("uploaded") },
    )
  }

  return (
    <div className="sm:col-span-2">
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
        {field.required && <span className="ml-0.5 text-red-500">*</span>}
      </label>
      {description && (
        <p className="mb-2 text-xs text-slate-500 dark:text-slate-400">{description}</p>
      )}

      {effectiveError && (
        <p className="mb-2 text-xs font-medium text-red-600 dark:text-red-400" role="alert">{effectiveError}</p>
      )}
      {!effectiveError && warning && !isUploaded && (
        <p className="mb-2 text-xs font-medium text-amber-600 dark:text-amber-400" role="status">{warning}</p>
      )}

      {isUploaded ? (
        <DocumentStatusDisplay
          status={docStatus}
          rejectionReason={matchingDoc?.rejection_reason}
          onReplace={() => setModalOpen(true)}
          t={t}
        />
      ) : (
        <DocumentUploadZone
          isPending={uploadMutation.isPending}
          hasError={!!effectiveError}
          hasWarning={!!warning}
          onClick={() => setModalOpen(true)}
          t={t}
        />
      )}

      {uploadMutation.isError && (
        <p className="mt-1 text-sm text-red-500">
          {uploadMutation.error?.message || "Upload failed"}
        </p>
      )}

      <UploadModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onUpload={handleUpload}
        accept="image/*,application/pdf"
        maxSize={10 * 1024 * 1024}
        title={label}
        description={t("documentUploadFormats")}
      />
    </div>
  )
}

/** Shows the appropriate status display for an uploaded document. */
function DocumentStatusDisplay({ status, rejectionReason, onReplace, t }: {
  status?: "pending" | "verified" | "rejected"
  rejectionReason?: string
  onReplace: () => void
  t: (key: string) => string
}) {
  if (status === "rejected") {
    return (
      <div className="flex items-center gap-3 rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-500/30 dark:bg-red-500/10">
        <XCircle className="h-5 w-5 shrink-0 text-red-600 dark:text-red-400" />
        <div className="flex-1">
          <span className="text-sm font-medium text-red-700 dark:text-red-300">
            {rejectionReason ? `Rejected: ${rejectionReason}` : t("documentRejected")}
          </span>
        </div>
        <button
          type="button"
          onClick={onReplace}
          className="text-xs font-semibold text-red-600 hover:text-red-800 dark:text-red-400"
        >
          {t("reuploadDocument")}
        </button>
      </div>
    )
  }

  if (status === "verified") {
    return (
      <div className="flex items-center gap-3 rounded-xl border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-500/30 dark:bg-emerald-500/10">
        <CheckCircle2 className="h-5 w-5 shrink-0 text-emerald-600 dark:text-emerald-400" />
        <span className="flex-1 text-sm font-medium text-emerald-700 dark:text-emerald-300">
          {t("documentVerified")}
        </span>
        <button
          type="button"
          onClick={onReplace}
          className="text-xs font-medium text-emerald-600 hover:text-emerald-800 dark:text-emerald-400"
        >
          {t("replaceDocument")}
        </button>
      </div>
    )
  }

  // Default: pending or unknown status
  return (
    <div className="flex items-center gap-3 rounded-xl border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-500/30 dark:bg-emerald-500/10">
      <CheckCircle2 className="h-5 w-5 shrink-0 text-emerald-600 dark:text-emerald-400" />
      <span className="flex-1 text-sm font-medium text-emerald-700 dark:text-emerald-300">
        {t("documentPending")}
      </span>
      <button
        type="button"
        onClick={onReplace}
        className="text-xs font-medium text-emerald-600 hover:text-emerald-800 dark:text-emerald-400"
      >
        {t("replaceDocument")}
      </button>
    </div>
  )
}

/** Upload dropzone with error (red) and warning (orange) border variants. */
function DocumentUploadZone({ isPending, hasError, hasWarning, onClick, t }: {
  isPending: boolean
  hasError: boolean
  hasWarning: boolean
  onClick: () => void
  t: (key: string) => string
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={isPending}
      className={cn(
        "w-full rounded-xl border-2 border-dashed p-6",
        "flex flex-col items-center gap-2 transition-colors",
        hasError
          ? "border-red-300 bg-red-50/50 dark:border-red-500/30 dark:bg-red-500/5"
          : hasWarning
            ? "border-amber-300 bg-amber-50/50 dark:border-amber-500/30 dark:bg-amber-500/5"
            : "border-slate-200 dark:border-slate-600",
        "hover:border-rose-300 hover:bg-rose-50/50 dark:hover:border-rose-500/30 dark:hover:bg-rose-500/5",
      )}
    >
      {isPending ? (
        <Loader2 className="h-8 w-8 animate-spin text-rose-500" />
      ) : (
        <Upload className="h-8 w-8 text-slate-400" />
      )}
      <span className="text-sm font-medium text-slate-600 dark:text-slate-400">
        {t("documentUploadClick")}
      </span>
      <span className="text-xs text-slate-400">{t("documentUploadFormats")}</span>
    </button>
  )
}

function StateSelectField({ field, value, onChange, label, error, countryCode }: DynamicFieldProps & { label: string; countryCode: string }) {
  const [states, setStates] = useState<StateOption[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    getStatesForCountry(countryCode).then((result) => {
      if (!cancelled) {
        setStates(result)
        setLoading(false)
      }
    })
    return () => { cancelled = true }
  }, [countryCode])

  const hasError = !!error

  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
        {field.required && <span className="ml-0.5 text-red-500">*</span>}
      </label>
      {loading ? (
        <div className="flex h-10 items-center rounded-lg border border-gray-200 bg-gray-50 px-3 dark:border-gray-700 dark:bg-gray-800">
          <Loader2 className="h-4 w-4 animate-spin text-gray-400" />
          <span className="ml-2 text-sm text-gray-400">Loading...</span>
        </div>
      ) : (
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          aria-invalid={hasError}
          className={cn(
            "h-10 w-full rounded-lg border bg-white px-3 text-sm shadow-xs transition-all duration-200",
            "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
            "dark:border-gray-600 dark:bg-gray-800 dark:text-white",
            hasError
              ? "border-red-500 ring-4 ring-red-500/10 dark:border-red-500"
              : "border-gray-200 dark:border-gray-700",
          )}
        >
          <option value="">--</option>
          {states.map((s) => (
            <option key={s.code} value={s.code}>{s.name}</option>
          ))}
        </select>
      )}
      {hasError && (
        <p className="mt-1 text-xs text-red-600 dark:text-red-400" role="alert">{error}</p>
      )}
    </div>
  )
}

function SelectField({ field, value, onChange, label, error }: DynamicFieldProps & { label: string }) {
  const hasError = !!error

  if (field.label_key === "nationality" || field.label_key === "country" || field.label_key === "bankCountry") {
    return (
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
          {field.required && <span className="ml-0.5 text-red-500">*</span>}
        </label>
        <CountrySelect value={value} onChange={onChange} />
        {hasError && <p className="mt-1 text-xs text-red-600 dark:text-red-400" role="alert">{error}</p>}
      </div>
    )
  }

  if (field.label_key === "politicalExposure") {
    return (
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
        </label>
        <SelectInput value={value} onChange={onChange} options={[
          { value: "none", label: "None" },
          { value: "existing", label: "Existing" },
        ]} />
        {hasError && <p className="mt-1 text-xs text-red-600 dark:text-red-400" role="alert">{error}</p>}
      </div>
    )
  }

  if (field.label_key === "gender") {
    return (
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
        </label>
        <SelectInput value={value} onChange={onChange} options={[
          { value: "male", label: "Male" },
          { value: "female", label: "Female" },
        ]} />
        {hasError && <p className="mt-1 text-xs text-red-600 dark:text-red-400" role="alert">{error}</p>}
      </div>
    )
  }

  if (field.label_key === "isExecutive") {
    return (
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
        </label>
        <SelectInput value={value} onChange={onChange} options={[
          { value: "true", label: "Yes" },
          { value: "false", label: "No" },
        ]} />
        {hasError && <p className="mt-1 text-xs text-red-600 dark:text-red-400" role="alert">{error}</p>}
      </div>
    )
  }

  // Generic select fallback — render as text input
  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
      </label>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={field.placeholder ?? ""}
        aria-invalid={hasError}
        className={cn(
          "h-10 w-full rounded-lg border bg-white px-3 text-sm shadow-xs transition-all duration-200",
          "placeholder:text-gray-400",
          "focus:outline-none",
          "dark:bg-gray-900 dark:text-gray-100 dark:placeholder:text-gray-500",
          hasError
            ? "border-red-500 ring-4 ring-red-500/10 dark:border-red-500"
            : "border-gray-200 focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 dark:border-gray-700",
        )}
      />
      {hasError && <p className="mt-1 text-xs text-red-600 dark:text-red-400" role="alert">{error}</p>}
    </div>
  )
}

function SelectInput({ value, onChange, options }: {
  value: string
  onChange: (value: string) => void
  options: { value: string; label: string }[]
}) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className={cn(
        "h-10 w-full rounded-lg border border-gray-200 bg-white px-3 text-sm shadow-xs transition-all duration-200",
        "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
        "dark:border-gray-600 dark:bg-gray-800 dark:text-white",
      )}
    >
      <option value="">--</option>
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>{opt.label}</option>
      ))}
    </select>
  )
}

/** Map our field types to HTML input types. */
function inputType(fieldType: string): string {
  switch (fieldType) {
    case "email": return "email"
    case "phone": return "tel"
    case "date": return "date"
    default: return "text"
  }
}

/** Safely translate a key, returning a humanized fallback if key is missing. */
function safeTranslate(t: (key: string) => string, key: string): string {
  try {
    const result = t(key)
    // If next-intl returns the full namespace key (e.g. "paymentInfo.xxx"), humanize it
    if (result.startsWith("paymentInfo.") || result === key) {
      return humanizeKey(key)
    }
    return result
  } catch {
    return humanizeKey(key)
  }
}

/** Translate a key, returning null if the key has no translation. */
function safeTranslateOptional(t: (key: string) => string, key: string): string | null {
  try {
    const result = t(key)
    if (result.startsWith("paymentInfo.") || result === key) {
      return null
    }
    return result
  } catch {
    return null
  }
}

/** Convert a camelCase or snake_case key to a human-readable label. */
function humanizeKey(key: string): string {
  return key
    .replace(/([A-Z])/g, " $1")
    .replace(/_/g, " ")
    .replace(/^\s+/, "")
    .replace(/\b\w/g, (c) => c.toUpperCase())
    .trim()
}

/** Derive a document_type string from a Stripe path for the upload API. */
function deriveDocumentType(path: string): string {
  if (path.includes("proof_of_liveness")) return "proof_of_liveness"
  if (path.includes("additional_document")) return "additional_document"
  if (path.includes("company_authorization")) return "company_authorization"
  if (path.includes("passport")) return "passport"
  if (path.includes("bank_account_ownership")) return "bank_account_ownership"
  return "document"
}
