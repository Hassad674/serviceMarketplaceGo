"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { Trash2, Download, AlertTriangle } from "lucide-react"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"
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
 * right-to-erasure flow. It renders three sections, all under the
 * Soleil v2 surface:
 *
 *   1. Export — single click triggers a ZIP download. No confirmation
 *      because the action is non-destructive.
 *   2. Delete — opens a password modal. On submit we POST
 *      /me/account/request-deletion and surface 401 / 409 errors
 *      inline (wrong password / org-owner-blocked). The destructive
 *      card carries a corail-deep border-l 4px, distinctive.
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
        <h2 className="font-serif text-[22px] font-semibold tracking-[-0.01em] text-foreground">
          {t("title")}
        </h2>
        <p className="mt-1.5 text-sm text-muted-foreground">
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

      <div className="rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)]">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] bg-primary-soft text-primary">
            <Download className="h-4 w-4" aria-hidden="true" />
          </div>
          <div className="flex-1 min-w-0">
            <h3 className="font-serif text-base font-medium text-foreground">
              {t("export.title")}
            </h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("export.description")}
            </p>
            {exportError && (
              <p
                className="mt-2 text-sm text-[var(--destructive)]"
                role="alert"
              >
                {exportError}
              </p>
            )}
          </div>
          <Button
            type="button"
            variant="outline"
            onClick={handleExport}
            disabled={exporting}
            className="rounded-full sm:shrink-0"
          >
            {exporting ? t("export.preparing") : t("export.button")}
          </Button>
        </div>
      </div>

      {!pendingDeletionAt && (
        <div className="rounded-2xl border border-[var(--primary-deep)]/25 border-l-4 border-l-[var(--primary-deep)] bg-[var(--primary-soft)]/40 p-6 shadow-[var(--shadow-card)]">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-start">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] bg-[var(--primary-deep)]/15 text-[var(--primary-deep)]">
              <Trash2 className="h-4 w-4" aria-hidden="true" />
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="font-serif text-base font-medium text-[var(--primary-deep)]">
                {t("delete.title")}
              </h3>
              <p className="mt-1 text-sm text-[var(--primary-deep)]/85">
                {t("delete.description")}
              </p>
            </div>
            <Button
              type="button"
              variant="destructive"
              onClick={() => setModalOpen(true)}
              className="rounded-full sm:shrink-0"
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
  deletedAt: _deletedAt,
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
      className="rounded-2xl border border-[var(--warning)]/40 border-l-4 border-l-[var(--warning)] bg-[var(--warning)]/10 p-5"
    >
      <div className="flex items-start gap-3">
        <AlertTriangle
          className="mt-0.5 h-5 w-5 text-[var(--warning)]"
          aria-hidden="true"
        />
        <div className="flex-1 min-w-0">
          <h3 className="font-serif text-base font-medium text-foreground">
            {t("title")}
          </h3>
          <p className="mt-1 text-sm text-muted-foreground">
            {hardDate
              ? t("body", { date: hardDate.toLocaleDateString() })
              : t("bodyNoDate")}
          </p>
          {cancelError && (
            <p
              className="mt-2 text-sm text-[var(--destructive)]"
              role="alert"
            >
              {cancelError}
            </p>
          )}
          <Button
            type="button"
            variant="primary"
            className="mt-3 rounded-full"
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
          <p className="text-sm text-foreground">
            {t("intro")}
          </p>

          <ul className="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
            <li>{t("bullet1")}</li>
            <li>{t("bullet2")}</li>
            <li>{t("bullet3")}</li>
          </ul>

          <div>
            <label
              htmlFor="gdpr-password"
              className="mb-1.5 block text-sm font-medium text-foreground"
            >
              {t("passwordLabel")}
            </label>
            <Input
              id="gdpr-password"
              type="password"
              required
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          <label className="flex items-start gap-2 text-sm text-foreground">
            {/* eslint-disable-next-line react/forbid-elements -- native checkbox; no Checkbox primitive in shared/ui yet, follow-up F.6 */}
            <input
              type="checkbox"
              checked={confirmed}
              onChange={(e) => setConfirmed(e.target.checked)}
              className="mt-0.5 h-4 w-4 rounded border-border-strong text-primary focus:ring-2 focus:ring-primary/30"
            />
            <span>{t("confirmCheckbox")}</span>
          </label>

          {error && (
            <p className="text-sm text-[var(--destructive)]" role="alert">
              {error}
            </p>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="ghost"
              onClick={handleClose}
              className="rounded-full"
            >
              {t("cancel")}
            </Button>
            <Button
              type="submit"
              variant="destructive"
              disabled={!confirmed || !password || submitting}
              className="rounded-full"
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
      <p className="text-sm text-foreground">{t("intro")}</p>
      <p className="rounded-xl bg-primary-soft px-3 py-2 font-mono text-sm font-medium text-[var(--primary-deep)]">
        {email}
      </p>
      <p className="text-sm text-muted-foreground">{t("ttl")}</p>
      <div className="flex justify-end pt-2">
        <Button variant="primary" onClick={onDone} className="rounded-full">
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
      <p className="text-sm text-foreground">{t("intro")}</p>
      <div className="space-y-3">
        {orgs.map((org) => (
          <div
            key={org.org_id}
            className="rounded-xl border border-[var(--warning)]/40 bg-[var(--warning)]/10 p-3"
          >
            <p className="font-medium text-foreground">{org.org_name}</p>
            <p className="text-xs text-muted-foreground">
              {t("memberCount", { count: org.member_count })}
            </p>
            <ul className="mt-2 flex flex-wrap gap-2">
              {org.actions.map((a) => (
                <li key={a}>
                  <span className="inline-flex items-center rounded-full bg-card px-2 py-1 text-xs font-medium text-foreground ring-1 ring-[var(--warning)]/40">
                    {t(`actions.${a}`)}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>
      <div className="flex justify-end pt-2">
        <Button variant="primary" onClick={onClose} className="rounded-full">
          {t("close")}
        </Button>
      </div>
    </div>
  )
}
