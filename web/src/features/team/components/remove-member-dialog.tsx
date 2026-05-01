"use client"

import { Loader2, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRemoveMember } from "../hooks/use-team"
import type { TeamMember } from "../types"

import { Button } from "@/shared/components/ui/button"
// Confirmation dialog for the "Retirer" action. Stateless — the
// mutation + its loading state live in the hook. Operator accounts
// get deleted server-side; marketplace-owner accounts just lose
// their membership.

type RemoveMemberDialogProps = {
  open: boolean
  onClose: () => void
  orgID: string
  member: TeamMember
}

export function RemoveMemberDialog({
  open,
  onClose,
  orgID,
  member,
}: RemoveMemberDialogProps) {
  const t = useTranslations("team")
  const mutation = useRemoveMember(orgID, member.user_id)

  if (!open) return null

  const displayName =
    member.user?.display_name ||
    `${member.user?.first_name ?? ""} ${member.user?.last_name ?? ""}`.trim() ||
    t("memberFallbackName")

  function handleConfirm() {
    mutation.mutate(undefined, {
      onSuccess: () => onClose(),
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
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
            {t("removeMemberTitle")}
          </h3>
          <Button variant="ghost" size="auto"
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 hover:bg-slate-100 dark:hover:bg-slate-700"
          >
            <X className="h-5 w-5 text-slate-400" />
          </Button>
        </div>

        <p className="text-sm text-slate-600 dark:text-slate-300">
          {t("removeMemberConfirm", { name: displayName })}
        </p>

        {mutation.isError && (
          <p className="mt-3 text-sm text-rose-600 dark:text-rose-400">
            {t("errors.generic")}
          </p>
        )}

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
            disabled={mutation.isPending}
            className="inline-flex items-center gap-2 rounded-lg bg-rose-500 px-4 py-2 text-sm font-semibold text-white hover:bg-rose-600 disabled:opacity-50"
          >
            {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("remove")}
          </Button>
        </div>
      </div>
    </div>
  )
}
