"use client"

import { Mail, RotateCcw, X, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { useCancelInvitation, useResendInvitation } from "../hooks/use-team"
import type { TeamInvitation } from "../types"

import { Button } from "@/shared/components/ui/button"
// Pending-invitations table. Each row has two actions (resend +
// cancel) that are visible only when the caller has team.invite,
// which is resolved upstream by the parent.

type TeamInvitationsListProps = {
  orgID: string
  invitations: TeamInvitation[]
  canInvite: boolean
}

export function TeamInvitationsList({
  orgID,
  invitations,
  canInvite,
}: TeamInvitationsListProps) {
  const t = useTranslations("team")

  if (invitations.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-gray-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-8 text-center text-sm text-gray-500 dark:text-gray-400">
        {t("emptyInvitations")}
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded-xl border border-gray-100 dark:border-slate-700 bg-white dark:bg-slate-800">
      <table className="w-full">
        <thead>
          <tr className="border-b border-gray-100 dark:border-slate-700 bg-gray-50 dark:bg-slate-900/50 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400">
            <th className="px-6 py-3">{t("columns.recipient")}</th>
            <th className="px-6 py-3">{t("columns.role")}</th>
            <th className="px-6 py-3">{t("columns.sentAt")}</th>
            <th className="px-6 py-3">{t("columns.expiresAt")}</th>
            {canInvite && <th className="px-6 py-3 text-right">{t("columns.actions")}</th>}
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100 dark:divide-slate-700">
          {invitations.map((inv) => (
            <InvitationRow
              key={inv.id}
              orgID={orgID}
              invitation={inv}
              canInvite={canInvite}
            />
          ))}
        </tbody>
      </table>
    </div>
  )
}

type InvitationRowProps = {
  orgID: string
  invitation: TeamInvitation
  canInvite: boolean
}

function InvitationRow({ orgID, invitation, canInvite }: InvitationRowProps) {
  const t = useTranslations("team")
  const resendMutation = useResendInvitation(orgID)
  const cancelMutation = useCancelInvitation(orgID)

  const fullName = `${invitation.first_name} ${invitation.last_name}`.trim()

  return (
    <tr className="text-sm">
      <td className="px-6 py-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gray-100 dark:bg-slate-700">
            <Mail className="h-4 w-4 text-gray-500" />
          </div>
          <div>
            <p className="font-medium text-gray-900 dark:text-white">
              {fullName || invitation.email}
            </p>
            <p className="text-xs text-gray-500 dark:text-gray-400">{invitation.email}</p>
          </div>
        </div>
      </td>
      <td className="px-6 py-4">
        <span className="inline-flex items-center rounded-full bg-gray-100 dark:bg-slate-700 px-2.5 py-0.5 text-xs font-semibold text-gray-700 dark:text-gray-300">
          {t(`roles.${invitation.role}`)}
        </span>
      </td>
      <td className="px-6 py-4 text-gray-500 dark:text-gray-400">
        {new Date(invitation.created_at).toLocaleDateString()}
      </td>
      <td className="px-6 py-4 text-gray-500 dark:text-gray-400">
        {new Date(invitation.expires_at).toLocaleDateString()}
      </td>
      {canInvite && (
        <td className="px-6 py-4 text-right">
          <div className="inline-flex gap-2">
            <Button variant="ghost" size="auto"
              type="button"
              onClick={() => resendMutation.mutate(invitation.id)}
              disabled={resendMutation.isPending}
              className="inline-flex items-center gap-1 rounded-lg border border-slate-200 dark:border-slate-600 px-3 py-1.5 text-xs font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50"
            >
              {resendMutation.isPending ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <RotateCcw className="h-3 w-3" />
              )}
              {t("resendAction")}
            </Button>
            <Button variant="ghost" size="auto"
              type="button"
              onClick={() => cancelMutation.mutate(invitation.id)}
              disabled={cancelMutation.isPending}
              className="inline-flex items-center gap-1 rounded-lg border border-rose-200 dark:border-rose-500/30 px-3 py-1.5 text-xs font-medium text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-500/10 disabled:opacity-50"
            >
              {cancelMutation.isPending ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <X className="h-3 w-3" />
              )}
              {t("cancelAction")}
            </Button>
          </div>
        </td>
      )}
    </tr>
  )
}
