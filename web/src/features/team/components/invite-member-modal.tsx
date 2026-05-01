"use client"

import { useState } from "react"
import { Loader2, X, Mail } from "lucide-react"
import { useTranslations } from "next-intl"
import { useSendInvitation } from "../hooks/use-team"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"
import { Select } from "@/shared/components/ui/select"
// Form modal to send a new invitation. Permission gating is done
// upstream (the "Inviter" button only renders when the caller has
// team.invite). Field validation is inline — nothing fancier than
// required checks and a minimal email regex, because the backend
// re-validates and returns a clean error on failure.

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

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-6 w-full max-w-lg mx-4 animate-scale-in"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Mail className="h-5 w-5 text-rose-500" />
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
              {t("inviteTitle")}
            </h3>
          </div>
          <Button variant="ghost" size="auto"
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 hover:bg-slate-100 dark:hover:bg-slate-700"
          >
            <X className="h-5 w-5 text-slate-400" />
          </Button>
        </div>

        <p className="text-sm text-slate-500 dark:text-slate-400 mb-4">
          {t("inviteDescription")}
        </p>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
              {t("emailLabel")}
            </label>
            <Input
              type="email"
              value={form.email}
              onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
              placeholder={t("emailPlaceholder")}
              className={`w-full rounded-lg border bg-white dark:bg-slate-900 px-3 py-2 text-sm text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-rose-500/20 ${
                emailError
                  ? "border-rose-500"
                  : "border-slate-200 dark:border-slate-600 focus:border-rose-500"
              }`}
            />
            {emailError && (
              <p className="mt-1 text-xs text-rose-600 dark:text-rose-400">
                {t("errors.emailInvalid")}
              </p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                {t("firstNameLabel")}
              </label>
              <Input
                type="text"
                value={form.firstName}
                onChange={(e) => setForm((f) => ({ ...f, firstName: e.target.value }))}
                className={`w-full rounded-lg border bg-white dark:bg-slate-900 px-3 py-2 text-sm text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-rose-500/20 ${
                  firstNameError
                    ? "border-rose-500"
                    : "border-slate-200 dark:border-slate-600 focus:border-rose-500"
                }`}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                {t("lastNameLabel")}
              </label>
              <Input
                type="text"
                value={form.lastName}
                onChange={(e) => setForm((f) => ({ ...f, lastName: e.target.value }))}
                className={`w-full rounded-lg border bg-white dark:bg-slate-900 px-3 py-2 text-sm text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-rose-500/20 ${
                  lastNameError
                    ? "border-rose-500"
                    : "border-slate-200 dark:border-slate-600 focus:border-rose-500"
                }`}
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
              {t("titleLabel")}
            </label>
            <Input
              type="text"
              value={form.title}
              onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
              maxLength={100}
              placeholder={t("titlePlaceholder")}
              className="w-full rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-2 text-sm text-slate-900 dark:text-white focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-500/20"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
              {t("roleLabel")}
            </label>
            <Select
              value={form.role}
              onChange={(e) =>
                setForm((f) => ({ ...f, role: e.target.value as FormState["role"] }))
              }
              className="w-full rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-2 text-sm text-slate-900 dark:text-white focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-500/20"
            >
              <option value="admin">{t("roles.admin")}</option>
              <option value="member">{t("roles.member")}</option>
              <option value="viewer">{t("roles.viewer")}</option>
            </Select>
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
              {t("roleHelp")}
            </p>
          </div>

          {mutation.isError && !hasValidationError && (
            <p className="text-sm text-rose-600 dark:text-rose-400">
              {t("errors.inviteFailed")}
            </p>
          )}
        </div>

        <div className="mt-6 flex justify-end gap-3">
          <Button variant="ghost" size="auto"
            type="button"
            onClick={onClose}
            disabled={mutation.isPending}
            className="rounded-lg border border-slate-200 dark:border-slate-600 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50"
          >
            {t("cancel")}
          </Button>
          <Button variant="ghost" size="auto"
            type="button"
            onClick={handleSubmit}
            disabled={mutation.isPending}
            className="inline-flex items-center gap-2 rounded-lg bg-rose-500 px-4 py-2 text-sm font-semibold text-white hover:bg-rose-600 disabled:opacity-50"
          >
            {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("sendInvite")}
          </Button>
        </div>
      </div>
    </div>
  )
}
