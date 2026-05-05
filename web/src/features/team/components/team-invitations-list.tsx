"use client"

import { Mail, RotateCcw, X, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { useCancelInvitation, useResendInvitation } from "../hooks/use-team"
import type { TeamInvitation } from "../types"

// Pending-invitations list. Each row is a Soleil card with corail-soft
// border (signals the "in flight" state) and ghost pill actions for
// resend / cancel. Rendered as a vertical stack — same anatomy as the
// members list — instead of a table for visual harmony.

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
      <div className="rounded-2xl border border-dashed border-[var(--border)] bg-[var(--surface)] p-10 text-center font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
        {t("emptyInvitations")}
      </div>
    )
  }

  return (
    <ul className="space-y-2">
      {invitations.map((inv) => (
        <li key={inv.id}>
          <InvitationRow
            orgID={orgID}
            invitation={inv}
            canInvite={canInvite}
          />
        </li>
      ))}
    </ul>
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
    <div className="flex flex-wrap items-center gap-3 rounded-2xl border border-[var(--primary-soft)] bg-[var(--surface)] px-4 py-3.5 shadow-[var(--shadow-card)]">
      <span
        aria-hidden="true"
        className="flex h-11 w-11 items-center justify-center rounded-full bg-[var(--primary-soft)] text-[var(--primary)]"
      >
        <Mail className="h-4 w-4" strokeWidth={1.8} />
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <p className="truncate font-serif text-[15px] font-medium text-[var(--foreground)]">
            {fullName || invitation.email}
          </p>
          <span className="inline-flex items-center rounded-full bg-[var(--background)] px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-[0.04em] text-[var(--muted-foreground)]">
            {t(`roles.${invitation.role}`)}
          </span>
        </div>
        <div className="mt-0.5 flex flex-wrap items-center gap-x-2 text-[12px] text-[var(--muted-foreground)]">
          <span className="truncate">{invitation.email}</span>
          <span aria-hidden="true" className="text-[var(--subtle-foreground)]">·</span>
          <span className="font-mono text-[11px] uppercase tracking-[0.05em] text-[var(--subtle-foreground)]">
            {t("w22.invitations.sent", {
              date: new Date(invitation.created_at).toLocaleDateString(),
            })}
          </span>
          <span aria-hidden="true" className="text-[var(--subtle-foreground)]">·</span>
          <span className="font-mono text-[11px] uppercase tracking-[0.05em] text-[var(--subtle-foreground)]">
            {t("w22.invitations.expires", {
              date: new Date(invitation.expires_at).toLocaleDateString(),
            })}
          </span>
        </div>
      </div>
      {canInvite && (
        <div className="flex shrink-0 items-center gap-2">
          <button
            type="button"
            onClick={() => resendMutation.mutate(invitation.id)}
            disabled={resendMutation.isPending}
            className="inline-flex items-center gap-1.5 rounded-full border border-[var(--border)] bg-[var(--background)] px-3 py-1.5 text-[12px] font-medium text-[var(--foreground)] transition-colors hover:border-[var(--border-strong)] hover:bg-[var(--surface)] disabled:opacity-50"
          >
            {resendMutation.isPending ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <RotateCcw className="h-3 w-3" strokeWidth={2} />
            )}
            {t("resendAction")}
          </button>
          <button
            type="button"
            onClick={() => cancelMutation.mutate(invitation.id)}
            disabled={cancelMutation.isPending}
            className="inline-flex items-center gap-1.5 rounded-full border border-[var(--primary-soft)] bg-[var(--surface)] px-3 py-1.5 text-[12px] font-medium text-[var(--primary-deep)] transition-colors hover:bg-[var(--primary-soft)] disabled:opacity-50"
          >
            {cancelMutation.isPending ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <X className="h-3 w-3" strokeWidth={2} />
            )}
            {t("cancelAction")}
          </button>
        </div>
      )}
    </div>
  )
}
