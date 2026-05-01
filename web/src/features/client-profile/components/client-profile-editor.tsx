"use client"

import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Loader2, Save } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { Button } from "@/shared/components/ui/button"

import { Input } from "@/shared/components/ui/input"
const COMPANY_NAME_MIN = 1
const COMPANY_NAME_MAX = 120
const DESCRIPTION_MAX = 2000

// Zod guarantees the two backend invariants upfront so the user sees
// inline error feedback before any network round-trip. Length limits
// mirror the backend contract — if either one drifts we want the
// frontend to fail loudly, not silently truncate.
const editorSchema = z.object({
  company_name: z
    .string()
    .trim()
    .min(COMPANY_NAME_MIN)
    .max(COMPANY_NAME_MAX),
  client_description: z.string().max(DESCRIPTION_MAX),
})

export type ClientProfileEditorValues = z.infer<typeof editorSchema>

export interface ClientProfileEditorProps {
  initialValues: ClientProfileEditorValues
  onSubmit: (values: ClientProfileEditorValues) => Promise<void> | void
  saving?: boolean
  submitError?: string | null
}

// ClientProfileEditor is the private form used on `/client-profile`.
// Kept under the 4-prop cap by bundling initial values into one
// object. Submit errors are surfaced via `submitError` so the page
// can map ApiError codes to localized copy before handing the string
// to the editor — the component itself stays display-only.
export function ClientProfileEditor(props: ClientProfileEditorProps) {
  const { initialValues, onSubmit, saving = false, submitError = null } = props
  const t = useTranslations("clientProfile")

  const form = useForm<ClientProfileEditorValues>({
    resolver: zodResolver(editorSchema),
    defaultValues: initialValues,
  })

  // Reset the form whenever the parent swaps `initialValues` (for
  // instance after a successful save that refetches the profile).
  // Without this the form keeps the previously-persisted draft and
  // users see stale values on subsequent edits.
  useEffect(() => {
    form.reset(initialValues)
  }, [initialValues, form])

  const descriptionValue = form.watch("client_description") ?? ""
  const descriptionLength = descriptionValue.length

  async function handleSubmit(values: ClientProfileEditorValues) {
    await onSubmit(values)
  }

  return (
    <form
      onSubmit={form.handleSubmit(handleSubmit)}
      className="bg-card border border-border rounded-2xl p-6 shadow-sm space-y-5"
      aria-labelledby="client-profile-editor-heading"
    >
      <h2
        id="client-profile-editor-heading"
        className="text-lg font-semibold text-foreground"
      >
        {t("editorTitle")}
      </h2>

      <Field
        id="client-profile-company-name"
        label={t("companyName")}
        error={form.formState.errors.company_name?.message}
      >
        <Input
          id="client-profile-company-name"
          type="text"
          autoComplete="organization"
          placeholder={t("companyNamePlaceholder")}
          aria-required="true"
          aria-invalid={Boolean(form.formState.errors.company_name)}
          className={cn(
            "w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground",
            "shadow-xs transition-colors duration-150",
            "focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
            form.formState.errors.company_name &&
              "border-red-500 focus:border-red-500 focus:ring-red-500/10",
          )}
          {...form.register("company_name")}
        />
      </Field>

      <Field
        id="client-profile-description"
        label={t("description")}
        help={t("descriptionHelp")}
        error={form.formState.errors.client_description?.message}
      >
        <textarea
          id="client-profile-description"
          rows={6}
          placeholder={t("descriptionPlaceholder")}
          maxLength={DESCRIPTION_MAX}
          aria-invalid={Boolean(form.formState.errors.client_description)}
          className={cn(
            "w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground",
            "shadow-xs transition-colors duration-150",
            "focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
            form.formState.errors.client_description &&
              "border-red-500 focus:border-red-500 focus:ring-red-500/10",
          )}
          {...form.register("client_description")}
        />
        <p
          className="mt-1 text-xs text-muted-foreground"
          aria-live="polite"
        >
          {t("descriptionCounter", {
            current: descriptionLength,
            max: DESCRIPTION_MAX,
          })}
        </p>
      </Field>

      {submitError ? (
        <p
          role="alert"
          className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive"
        >
          {submitError}
        </p>
      ) : null}

      <div className="flex justify-end">
        <Button variant="ghost" size="auto"
          type="submit"
          disabled={saving || !form.formState.isDirty}
          className={cn(
            "inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium text-white",
            "gradient-primary bg-rose-500 hover:bg-rose-600 transition-all duration-200",
            "focus:outline-none focus:ring-4 focus:ring-rose-500/20",
            "disabled:cursor-not-allowed disabled:opacity-60",
          )}
        >
          {saving ? (
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          ) : (
            <Save className="h-4 w-4" aria-hidden="true" />
          )}
          {saving ? t("saving") : t("saveChanges")}
        </Button>
      </div>
    </form>
  )
}

interface FieldProps {
  id: string
  label: string
  help?: string
  error?: string
  children: React.ReactNode
}

function Field({ id, label, help, error, children }: FieldProps) {
  return (
    <div className="space-y-1.5">
      <label
        htmlFor={id}
        className="block text-sm font-medium text-foreground"
      >
        {label}
      </label>
      {children}
      {error ? (
        <p className="text-xs text-destructive" role="alert">
          {error}
        </p>
      ) : help ? (
        <p className="text-xs text-muted-foreground">{help}</p>
      ) : null}
    </div>
  )
}
