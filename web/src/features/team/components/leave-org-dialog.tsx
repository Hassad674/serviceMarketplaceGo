"use client"

import { Loader2, X, LogOut } from "lucide-react"
import { useTranslations } from "next-intl"
import { useLeaveOrganization } from "../hooks/use-team"

// Self-leave confirmation dialog. On success we hard-redirect to
// /dashboard — the user's session has been invalidated (operator
// accounts are deleted entirely) so a re-auth may be required.

type LeaveOrgDialogProps = {
  open: boolean
  onClose: () => void
  orgID: string
}

export function LeaveOrgDialog({ open, onClose, orgID }: LeaveOrgDialogProps) {
  const t = useTranslations("team")
  const mutation = useLeaveOrganization(orgID)

  if (!open) return null

  function handleConfirm() {
    mutation.mutate(undefined, {
      onSuccess: () => {
        // Hard nav so the query cache + auth cookie are both re-evaluated.
        window.location.href = "/"
      },
    })
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-6 w-full max-w-md mx-4 animate-scale-in"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <LogOut className="h-5 w-5 text-rose-500" />
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
              {t("leaveTitle")}
            </h3>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 hover:bg-slate-100 dark:hover:bg-slate-700"
          >
            <X className="h-5 w-5 text-slate-400" />
          </button>
        </div>

        <p className="text-sm text-slate-600 dark:text-slate-300">
          {t("leaveConfirm")}
        </p>

        {mutation.isError && (
          <p className="mt-3 text-sm text-rose-600 dark:text-rose-400">
            {t("errors.leaveFailed")}
          </p>
        )}

        <div className="mt-6 flex justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            disabled={mutation.isPending}
            className="rounded-lg border border-slate-200 dark:border-slate-600 px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50"
          >
            {t("cancel")}
          </button>
          <button
            type="button"
            onClick={handleConfirm}
            disabled={mutation.isPending}
            className="inline-flex items-center gap-2 rounded-lg bg-rose-500 px-4 py-2 text-sm font-semibold text-white hover:bg-rose-600 disabled:opacity-50"
          >
            {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("leaveConfirmButton")}
          </button>
        </div>
      </div>
    </div>
  )
}
