"use client"

import { useState } from "react"
import { Loader2, X, Mail } from "lucide-react"
import { useTranslations } from "next-intl"
import { useSendInvitation } from "../hooks/use-team"

// Soleil v2 invite modal. Ivoire surface, Fraunces title, corail-soft
// header chip + corail CTA. Field validation is inline (required
// + email format) — backend re-validates on submit.

type InviteMemberModalProps = {
  open: boolean
  onClose: () => void
  orgID: string
}

type FormState = {
  email: string
  firstName: string
  lastName: string
  title: string
  role: "admin" | "member" | "viewer"
}

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function InviteMemberModal({ open, onClose, orgID }: InviteMemberModalProps) {
  const t = useTranslations("team")
  const mutation = useSendInvitation(orgID)

  const [form, setForm] = useState<FormState>({
    email: "",
    firstName: "",
    lastName: "",
    title: "",
    role: "member",
  })
  const [touched, setTouched] = useState(false)

  if (!open) return null

  const emailError =
    touched && (!form.email.trim() || !EMAIL_RE.test(form.email.trim()))
  const firstNameError = touched && !form.firstName.trim()
  const lastNameError = touched && !form.lastName.trim()
  const hasValidationError = emailError || firstNameError || lastNameError

  function reset() {
    setForm({ email: "", firstName: "", lastName: "", title: "", role: "member" })
    setTouched(false)
  }

  function handleSubmit() {
    setTouched(true)
    if (
      !form.email.trim() ||
      !EMAIL_RE.test(form.email.trim()) ||
      !form.firstName.trim() ||
      !form.lastName.trim()
    ) {
      return
    }
    mutation.mutate(
      {
        email: form.email.trim(),
        first_name: form.firstName.trim(),
        last_name: form.lastName.trim(),
        title: form.title.trim(),
        role: form.role,
      },
      {
        onSuccess: () => {
          reset()
          onClose()
        },
      },
    )
  }

  const inputBase =
    "w-full rounded-xl border bg-[var(--surface)] px-3 py-2.5 text-[14px] text-[var(--foreground)] placeholder:text-[var(--subtle-foreground)] focus:outline-none focus:ring-2"
  const inputOk =
    "border-[var(--border)] focus:border-[var(--primary)] focus:ring-[var(--primary-soft)]"
  const inputErr = "border-[var(--primary-deep)] focus:ring-[var(--primary-soft)]"

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(42,31,21,0.45)] p-4 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="animate-scale-in w-full max-w-lg rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-6 shadow-[var(--shadow-card-strong)]"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-start justify-between gap-3">
          <div className="flex items-start gap-3">
            <span
              aria-hidden="true"
              className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[var(--primary-soft)] text-[var(--primary)]"
            >
              <Mail className="h-5 w-5" strokeWidth={1.8} />
            </span>
            <div>
              <h3 className="font-serif text-[20px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
                {t("inviteTitle")}
              </h3>
              <p className="mt-1 font-serif text-[13px] italic text-[var(--muted-foreground)]">
                {t("inviteDescription")}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label={t("cancel")}
            className="rounded-full p-1 text-[var(--muted-foreground)] transition-colors hover:bg-[var(--background)] hover:text-[var(--foreground)]"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="space-y-4">
          <div>
            <label
              htmlFor="invite-email"
              className="mb-1.5 block text-[12px] font-semibold uppercase tracking-[0.04em] text-[var(--muted-foreground)]"
            >
              {t("emailLabel")}
            </label>
            <input
              id="invite-email"
              type="email"
              value={form.email}
              onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
              placeholder={t("emailPlaceholder")}
              className={`${inputBase} ${emailError ? inputErr : inputOk}`}
            />
            {emailError && (
              <p className="mt-1 text-[12px] text-[var(--primary-deep)]">
                {t("errors.emailInvalid")}
              </p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label
                htmlFor="invite-firstname"
                className="mb-1.5 block text-[12px] font-semibold uppercase tracking-[0.04em] text-[var(--muted-foreground)]"
              >
                {t("firstNameLabel")}
              </label>
              <input
                id="invite-firstname"
                type="text"
                value={form.firstName}
                onChange={(e) => setForm((f) => ({ ...f, firstName: e.target.value }))}
                className={`${inputBase} ${firstNameError ? inputErr : inputOk}`}
              />
            </div>
            <div>
              <label
                htmlFor="invite-lastname"
                className="mb-1.5 block text-[12px] font-semibold uppercase tracking-[0.04em] text-[var(--muted-foreground)]"
              >
                {t("lastNameLabel")}
              </label>
              <input
                id="invite-lastname"
                type="text"
                value={form.lastName}
                onChange={(e) => setForm((f) => ({ ...f, lastName: e.target.value }))}
                className={`${inputBase} ${lastNameError ? inputErr : inputOk}`}
              />
            </div>
          </div>

          <div>
            <label
              htmlFor="invite-title"
              className="mb-1.5 block text-[12px] font-semibold uppercase tracking-[0.04em] text-[var(--muted-foreground)]"
            >
              {t("titleLabel")}
            </label>
            <input
              id="invite-title"
              type="text"
              value={form.title}
              onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
              maxLength={100}
              placeholder={t("titlePlaceholder")}
              className={`${inputBase} ${inputOk}`}
            />
          </div>

          <div>
            <label
              htmlFor="invite-role"
              className="mb-1.5 block text-[12px] font-semibold uppercase tracking-[0.04em] text-[var(--muted-foreground)]"
            >
              {t("roleLabel")}
            </label>
            <select
              id="invite-role"
              value={form.role}
              onChange={(e) =>
                setForm((f) => ({ ...f, role: e.target.value as FormState["role"] }))
              }
              className={`${inputBase} ${inputOk} cursor-pointer`}
            >
              <option value="admin">{t("roles.admin")}</option>
              <option value="member">{t("roles.member")}</option>
              <option value="viewer">{t("roles.viewer")}</option>
            </select>
            <p className="mt-1 font-serif text-[12px] italic text-[var(--muted-foreground)]">
              {t("roleHelp")}
            </p>
          </div>

          {mutation.isError && !hasValidationError && (
            <p className="rounded-xl border border-[var(--primary-soft)] bg-[var(--primary-soft)] px-3 py-2 text-[13px] text-[var(--primary-deep)]">
              {t("errors.inviteFailed")}
            </p>
          )}
        </div>

        <div className="mt-6 flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            disabled={mutation.isPending}
            className="rounded-full border border-[var(--border)] bg-[var(--surface)] px-4 py-2 text-[13px] font-semibold text-[var(--foreground)] transition-colors hover:border-[var(--border-strong)] hover:bg-[var(--background)] disabled:opacity-50"
          >
            {t("cancel")}
          </button>
          <button
            type="button"
            onClick={handleSubmit}
            disabled={mutation.isPending}
            className="inline-flex items-center gap-2 rounded-full bg-[var(--primary)] px-4 py-2 text-[13px] font-semibold text-[var(--primary-foreground)] shadow-[var(--shadow-message)] transition-colors hover:bg-[var(--primary-deep)] disabled:opacity-50"
          >
            {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("sendInvite")}
          </button>
        </div>
      </div>
    </div>
  )
}
