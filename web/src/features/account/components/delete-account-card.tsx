"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { Trash2, Download, AlertTriangle } from "lucide-react"

import { Button } from "@/shared/components/ui/button"
import { Modal } from "@/shared/components/ui/modal"
import { ApiError } from "@/shared/lib/api-client"
import {
  cancelDeletion,
  downloadExport,
  requestDeletion,
  type BlockedOrg,
} from "../api/gdpr"

/**
 * DeleteAccountCard is the user-facing entry point for the GDPR
 * right-to-erasure flow. It renders three sections:
 *
 *   1. Export — single click triggers a ZIP download. No confirmation
 *      because the action is non-destructive.
 *   2. Delete — opens a password modal. On submit we POST
 *      /me/account/request-deletion and surface 401 / 409 errors
 *      inline (wrong password / org-owner-blocked).
 *   3. Cancel — visible only when `pendingDeletionAt` is set.
 *      One click rolls back the soft-delete server-side and reloads
 *      the page so the banner disappears.
 *
 * Inputs from the parent (account-settings-page) keep this component
 * stateless across navigations — the password value is local only.
 */
export type DeleteAccountCardProps = {
  /**
   * RFC3339 timestamp the deletion was requested, when the account
   * is currently in its 30-day cooldown. `null` for healthy accounts.
   */
  pendingDeletionAt: string | null
  /**
   * RFC3339 timestamp the cron will purge if the user does not
   * cancel. Computed by the backend at request time.
   */
  hardDeleteAt: string | null
  /**
   * Optional callback invoked after a successful cancel. The parent
   * uses this to refresh the user state (e.g. via TanStack Query
   * invalidation). When omitted, the component performs window.location.reload().
   */
  onCancelled?: () => void
}

export function DeleteAccountCard({
  pendingDeletionAt,
  hardDeleteAt,
  onCancelled,
}: DeleteAccountCardProps) {
  const t = useTranslations("account.gdpr")
  const [modalOpen, setModalOpen] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [exportError, setExportError] = useState<string | null>(null)
  const [cancelling, setCancelling] = useState(false)
  const [cancelError, setCancelError] = useState<string | null>(null)

  async function handleExport() {
    setExporting(true)
    setExportError(null)
    try {
      await downloadExport()
    } catch (err) {
      setExportError(err instanceof Error ? err.message : t("export.error"))
    } finally {
      setExporting(false)
    }
  }

  async function handleCancel() {
    setCancelling(true)
    setCancelError(null)
    try {
      await cancelDeletion()
      if (onCancelled) onCancelled()
      else window.location.reload()
    } catch (err) {
      setCancelError(err instanceof Error ? err.message : t("cancel.error"))
    } finally {
      setCancelling(false)
    }
  }

  return (
    <section className="space-y-6">
      <header>
        <h2 className="text-xl font-semibold text-slate-900 dark:text-white">
          {t("title")}
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
          {t("subtitle")}
        </p>
      </header>

      {pendingDeletionAt && (
        <PendingDeletionBanner
          deletedAt={pendingDeletionAt}
          hardDeleteAt={hardDeleteAt}
          onCancel={handleCancel}
          cancelling={cancelling}
          cancelError={cancelError}
        />
      )}

      <div className="rounded-xl border border-slate-200 bg-white p-5 dark:border-slate-700 dark:bg-slate-800">
        <div className="flex items-start gap-3">
          <Download className="mt-0.5 h-5 w-5 text-slate-500" aria-hidden="true" />
          <div className="flex-1">
            <h3 className="font-medium text-slate-900 dark:text-white">
              {t("export.title")}
            </h3>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
              {t("export.description")}
            </p>
            {exportError && (
              <p className="mt-2 text-sm text-red-600" role="alert">
                {exportError}
              </p>
            )}
          </div>
          <Button
            type="button"
            variant="outline"
            onClick={handleExport}
            disabled={exporting}
          >
            {exporting ? t("export.preparing") : t("export.button")}
          </Button>
        </div>
      </div>

      {!pendingDeletionAt && (
        <div className="rounded-xl border border-red-200 bg-red-50/50 p-5 dark:border-red-900/40 dark:bg-red-950/20">
          <div className="flex items-start gap-3">
            <Trash2 className="mt-0.5 h-5 w-5 text-red-600" aria-hidden="true" />
            <div className="flex-1">
              <h3 className="font-medium text-red-900 dark:text-red-300">
                {t("delete.title")}
              </h3>
              <p className="mt-1 text-sm text-red-800/80 dark:text-red-300/80">
                {t("delete.description")}
              </p>
            </div>
            <Button
              type="button"
              variant="destructive"
              onClick={() => setModalOpen(true)}
            >
              {t("delete.button")}
            </Button>
          </div>
        </div>
      )}

      <DeleteAccountModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onSent={() => {
          // After the email is sent the modal closes itself; the
          // parent will re-render with pendingDeletionAt once the
          // user clicks the link. No state to set here.
          setModalOpen(false)
        }}
      />
    </section>
  )
}

function PendingDeletionBanner({
  deletedAt,
  hardDeleteAt,
  onCancel,
  cancelling,
  cancelError,
}: {
  deletedAt: string
  hardDeleteAt: string | null
  onCancel: () => void
  cancelling: boolean
  cancelError: string | null
}) {
  const t = useTranslations("account.gdpr.pending")
  const hardDate = hardDeleteAt ? new Date(hardDeleteAt) : null

  return (
    <div
      role="alert"
      className="rounded-xl border-l-4 border-amber-500 bg-amber-50 p-4 dark:bg-amber-950/30"
    >
      <div className="flex items-start gap-3">
        <AlertTriangle className="mt-0.5 h-5 w-5 text-amber-600" aria-hidden="true" />
        <div className="flex-1">
          <h3 className="font-medium text-amber-900 dark:text-amber-200">
            {t("title")}
          </h3>
          <p className="mt-1 text-sm text-amber-800 dark:text-amber-200/80">
            {hardDate
              ? t("body", { date: hardDate.toLocaleDateString() })
              : t("bodyNoDate")}
          </p>
          {cancelError && (
            <p className="mt-2 text-sm text-red-700" role="alert">
              {cancelError}
            </p>
          )}
          <Button
            type="button"
            variant="primary"
            className="mt-3"
            onClick={onCancel}
            disabled={cancelling}
          >
            {cancelling ? t("cancelling") : t("cancelButton")}
          </Button>
        </div>
      </div>
    </div>
  )
}

function DeleteAccountModal({
  open,
  onClose,
  onSent,
}: {
  open: boolean
  onClose: () => void
  onSent: () => void
}) {
  const t = useTranslations("account.gdpr.delete.modal")
  const [password, setPassword] = useState("")
  const [confirmed, setConfirmed] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [blocked, setBlocked] = useState<BlockedOrg[] | null>(null)
  const [emailSentTo, setEmailSentTo] = useState<string | null>(null)

  function reset() {
    setPassword("")
    setConfirmed(false)
    setError(null)
    setBlocked(null)
    setEmailSentTo(null)
    setSubmitting(false)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setBlocked(null)
    setSubmitting(true)
    try {
      const res = await requestDeletion(password)
      setEmailSentTo(res.email_sent_to)
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 409) {
          const details = (err.body?.error as { details?: { blocked_orgs?: BlockedOrg[] } } | undefined)
            ?.details
          setBlocked(details?.blocked_orgs ?? [])
        } else if (err.code === "invalid_password") {
          setError(t("errors.wrongPassword"))
        } else if (err.code === "confirm_required") {
          setError(t("errors.confirmRequired"))
        } else if (err.code === "password_required") {
          setError(t("errors.passwordRequired"))
        } else {
          setError(err.message)
        }
      } else {
        setError(t("errors.generic"))
      }
    } finally {
      setSubmitting(false)
    }
  }

  function handleClose() {
    reset()
    onClose()
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={t("title")}
      maxWidthClassName="max-w-lg"
    >
      {emailSentTo ? (
        <SuccessPanel email={emailSentTo} onDone={() => { reset(); onSent() }} />
      ) : blocked ? (
        <BlockedPanel orgs={blocked} onClose={handleClose} />
      ) : (
        <form onSubmit={handleSubmit} className="space-y-4">
          <p className="text-sm text-slate-700 dark:text-slate-300">
            {t("intro")}
          </p>

          <ul className="list-disc space-y-1 pl-5 text-sm text-slate-700 dark:text-slate-300">
            <li>{t("bullet1")}</li>
            <li>{t("bullet2")}</li>
            <li>{t("bullet3")}</li>
          </ul>

          <div>
            <label
              htmlFor="gdpr-password"
              className="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300"
            >
              {t("passwordLabel")}
            </label>
            <input
              id="gdpr-password"
              type="password"
              required
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10 dark:border-slate-700 dark:bg-slate-900"
            />
          </div>

          <label className="flex items-start gap-2 text-sm text-slate-700 dark:text-slate-300">
            <input
              type="checkbox"
              checked={confirmed}
              onChange={(e) => setConfirmed(e.target.checked)}
              className="mt-0.5 h-4 w-4 rounded border-slate-300 text-rose-600 focus:ring-rose-500"
            />
            <span>{t("confirmCheckbox")}</span>
          </label>

          {error && (
            <p className="text-sm text-red-600" role="alert">
              {error}
            </p>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="ghost" onClick={handleClose}>
              {t("cancel")}
            </Button>
            <Button
              type="submit"
              variant="destructive"
              disabled={!confirmed || !password || submitting}
            >
              {submitting ? t("submitting") : t("submit")}
            </Button>
          </div>
        </form>
      )}
    </Modal>
  )
}

function SuccessPanel({ email, onDone }: { email: string; onDone: () => void }) {
  const t = useTranslations("account.gdpr.delete.modal.success")
  return (
    <div className="space-y-4">
      <p className="text-sm text-slate-700 dark:text-slate-300">{t("intro")}</p>
      <p className="rounded-lg bg-rose-50 px-3 py-2 text-sm font-medium text-rose-900 dark:bg-rose-950/30 dark:text-rose-200">
        {email}
      </p>
      <p className="text-sm text-slate-600 dark:text-slate-400">{t("ttl")}</p>
      <div className="flex justify-end pt-2">
        <Button variant="primary" onClick={onDone}>
          {t("close")}
        </Button>
      </div>
    </div>
  )
}

function BlockedPanel({ orgs, onClose }: { orgs: BlockedOrg[]; onClose: () => void }) {
  const t = useTranslations("account.gdpr.delete.modal.blocked")
  return (
    <div className="space-y-4">
      <p className="text-sm text-slate-700 dark:text-slate-300">{t("intro")}</p>
      <div className="space-y-3">
        {orgs.map((org) => (
          <div
            key={org.org_id}
            className="rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-900/40 dark:bg-amber-950/30"
          >
            <p className="font-medium text-amber-900 dark:text-amber-200">
              {org.org_name}
            </p>
            <p className="text-xs text-amber-800/80 dark:text-amber-200/80">
              {t("memberCount", { count: org.member_count })}
            </p>
            <ul className="mt-2 flex flex-wrap gap-2">
              {org.actions.map((a) => (
                <li key={a}>
                  <span className="inline-flex items-center rounded-full bg-white px-2 py-1 text-xs font-medium text-amber-900 ring-1 ring-amber-200 dark:bg-slate-800 dark:text-amber-200 dark:ring-amber-900/40">
                    {t(`actions.${a}`)}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>
      <div className="flex justify-end pt-2">
        <Button variant="primary" onClick={onClose}>
          {t("close")}
        </Button>
      </div>
    </div>
  )
}
