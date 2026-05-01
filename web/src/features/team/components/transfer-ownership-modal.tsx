"use client"

import { useState } from "react"
import { Loader2, X, AlertTriangle } from "lucide-react"
import { useTranslations } from "next-intl"
import { useInitiateTransfer } from "../hooks/use-team"
import type { TeamMember } from "../types"

import { Button } from "@/shared/components/ui/button"
import { Select } from "@/shared/components/ui/select"
// Initiates the 2-step ownership transfer flow. Only the current
// Owner sees the button that opens this modal (gated upstream via
// permissions). The target MUST be an existing Admin in the org —
// the backend rejects other roles. We filter the member list here
// to match so the user can't even pick an invalid target.

type TransferOwnershipModalProps = {
  open: boolean
  onClose: () => void
  orgID: string
  members: TeamMember[]
  currentOwnerID: string
}

export function TransferOwnershipModal({
  open,
  onClose,
  orgID,
  members,
  currentOwnerID,
}: TransferOwnershipModalProps) {
  const t = useTranslations("team")
  const mutation = useInitiateTransfer(orgID)

  const eligible = members.filter(
    (m) => m.role === "admin" && m.user_id !== currentOwnerID,
  )

  const [targetUserID, setTargetUserID] = useState("")

  if (!open) return null

  function handleConfirm() {
    if (!targetUserID) return
    mutation.mutate(
      { target_user_id: targetUserID },
      {
        onSuccess: () => {
          setTargetUserID("")
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
            <AlertTriangle className="h-5 w-5 text-amber-500" />
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
              {t("transferTitle")}
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

        <div className="rounded-lg bg-amber-50 dark:bg-amber-500/10 border border-amber-200 dark:border-amber-500/30 p-3 text-xs text-amber-800 dark:text-amber-300 mb-4">
          {t("transferWarning")}
        </div>

        <div className="space-y-4">
          {eligible.length === 0 ? (
            <div className="rounded-lg border border-dashed border-gray-200 dark:border-slate-700 p-4 text-center text-sm text-gray-500 dark:text-gray-400">
              {t("transferNoEligible")}
            </div>
          ) : (
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1.5">
                {t("transferTargetLabel")}
              </label>
              <Select
                value={targetUserID}
                onChange={(e) => setTargetUserID(e.target.value)}
                className="w-full rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-2 text-sm text-slate-900 dark:text-white focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-500/20"
              >
                <option value="">{t("transferSelectPlaceholder")}</option>
                {eligible.map((m) => {
                  const name =
                    m.user?.display_name ||
                    `${m.user?.first_name ?? ""} ${m.user?.last_name ?? ""}`.trim() ||
                    m.user_id.slice(0, 8)
                  return (
                    <option key={m.user_id} value={m.user_id}>
                      {name} {m.title ? `· ${m.title}` : ""}
                    </option>
                  )
                })}
              </Select>
            </div>
          )}

          {mutation.isError && (
            <p className="text-sm text-rose-600 dark:text-rose-400">
              {t("errors.generic")}
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
            onClick={handleConfirm}
            disabled={mutation.isPending || !targetUserID || eligible.length === 0}
            className="inline-flex items-center gap-2 rounded-lg bg-amber-500 px-4 py-2 text-sm font-semibold text-white hover:bg-amber-600 disabled:opacity-50"
          >
            {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("transferConfirm")}
          </Button>
        </div>
      </div>
    </div>
  )
}
