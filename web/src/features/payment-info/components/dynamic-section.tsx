"use client"

import { useState } from "react"
import { Upload, CheckCircle2, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { CountrySelect } from "./country-select"
import { UploadModal } from "@/shared/components/upload-modal"
import { useUploadIdentityDocument } from "../hooks/use-identity-documents"
import type { FieldSection, FieldSpec } from "../api/payment-info-api"

interface DynamicSectionProps {
  section: FieldSection
  values: Record<string, string>
  onChange: (key: string, value: string) => void
}

export function DynamicSection({ section, values, onChange }: DynamicSectionProps) {
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
}

function DynamicField({ field, value, onChange }: DynamicFieldProps) {
  const t = useTranslations("paymentInfo")

  if (field.type === "document_upload") {
    return <DocumentUploadField field={field} value={value} onChange={onChange} />
  }

  const label = safeTranslate(t, field.label_key)

  if (field.type === "select") {
    return <SelectField field={field} value={value} onChange={onChange} label={label} />
  }

  const isIban = field.key.includes("iban") || field.path.includes("iban")

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
        className={cn(
          "h-10 w-full rounded-lg border border-gray-200 bg-white px-3 text-sm shadow-xs transition-all duration-200",
          "placeholder:text-gray-400",
          "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
          "dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100 dark:placeholder:text-gray-500",
        )}
      />
      {isIban && (
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

function DocumentUploadField({ field, value, onChange }: DynamicFieldProps) {
  const t = useTranslations("paymentInfo")
  const uploadMutation = useUploadIdentityDocument()
  const [modalOpen, setModalOpen] = useState(false)
  const label = safeTranslate(t, field.label_key)
  const descKey = field.label_key + "Desc"
  const description = safeTranslateOptional(t, descKey)
  const isUploaded = value === "uploaded"

  const category = field.path.startsWith("company") || field.path.startsWith("documents") ? "company" : "identity"
  const documentType = deriveDocumentType(field.path)

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

      {isUploaded ? (
        <div className="flex items-center gap-3 rounded-xl border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-500/30 dark:bg-emerald-500/10">
          <CheckCircle2 className="h-5 w-5 shrink-0 text-emerald-600 dark:text-emerald-400" />
          <span className="flex-1 text-sm font-medium text-emerald-700 dark:text-emerald-300">
            {t("documentUploaded")}
          </span>
          <button
            type="button"
            onClick={() => setModalOpen(true)}
            className="text-xs font-medium text-emerald-600 hover:text-emerald-800 dark:text-emerald-400"
          >
            {t("replaceDocument")}
          </button>
        </div>
      ) : (
        <button
          type="button"
          onClick={() => setModalOpen(true)}
          disabled={uploadMutation.isPending}
          className={cn(
            "w-full rounded-xl border-2 border-dashed p-6",
            "flex flex-col items-center gap-2 transition-colors",
            "border-slate-200 dark:border-slate-600",
            "hover:border-rose-300 hover:bg-rose-50/50 dark:hover:border-rose-500/30 dark:hover:bg-rose-500/5",
          )}
        >
          {uploadMutation.isPending ? (
            <Loader2 className="h-8 w-8 animate-spin text-rose-500" />
          ) : (
            <Upload className="h-8 w-8 text-slate-400" />
          )}
          <span className="text-sm font-medium text-slate-600 dark:text-slate-400">
            {t("documentUploadClick")}
          </span>
          <span className="text-xs text-slate-400">{t("documentUploadFormats")}</span>
        </button>
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

function SelectField({ field, value, onChange, label }: DynamicFieldProps & { label: string }) {
  if (field.label_key === "nationality" || field.label_key === "country" || field.label_key === "bankCountry") {
    return (
      <div>
        <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {label}
          {field.required && <span className="ml-0.5 text-red-500">*</span>}
        </label>
        <CountrySelect value={value} onChange={onChange} />
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
        className={cn(
          "h-10 w-full rounded-lg border border-gray-200 bg-white px-3 text-sm shadow-xs transition-all duration-200",
          "placeholder:text-gray-400",
          "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
          "dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100 dark:placeholder:text-gray-500",
        )}
      />
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
