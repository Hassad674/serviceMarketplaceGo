"use client"

import { Crown, Loader2, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { useAcceptTransfer, useDeclineTransfer, useCancelTransfer } from "../hooks/use-team"

// Soleil v2 — Pending ownership transfer banner. Two flavours driven
// by `viewerRole`:
//   - "target": the current user is the proposed new Owner -> Accept/Decline
//   - "initiator": the current user is the Owner who initiated the
//     transfer -> "En attente" + Cancel
// All other roles never see this banner. Uses the warm corail-soft +
// amber-soft palette to signal both urgency and calm.

type PendingTransferBannerProps = {
  orgID: string
  viewerRole: "target" | "initiator"
  expiresAt?: string
}

export function PendingTransferBanner({
  orgID: _orgID,
  viewerRole,
  expiresAt,
}: PendingTransferBannerProps) {
  const t = useTranslations("team")
  const acceptMutation = useAcceptTransfer(_orgID)
  const declineMutation = useDeclineTransfer(_orgID)
  const cancelMutation = useCancelTransfer(_orgID)

  const formattedExpiry = expiresAt ? new Date(expiresAt).toLocaleDateString() : undefined

  if (viewerRole === "target") {
    return (
      <div className="rounded-2xl border border-[var(--primary-soft)] bg-gradient-to-br from-[var(--amber-soft)] to-[var(--primary-soft)] p-5 shadow-[var(--shadow-card)]">
        <div className="flex items-start gap-3">
          <span
            aria-hidden="true"
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-[var(--surface)] text-[var(--warning)]"
          >
            <Crown className="h-5 w-5" strokeWidth={1.8} />
          </span>
          <div className="min-w-0 flex-1">
            <h3 className="font-serif text-[18px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
              {t("pendingTransferTargetTitle")}
            </h3>
            <p className="mt-1 font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
              {t("pendingTransferTargetDescription")}
            </p>
            {formattedExpiry && (
              <p className="mt-1 font-mono text-[11px] uppercase tracking-[0.05em] text-[var(--subtle-foreground)]">
                {t("pendingTransferExpiresOn", { date: formattedExpiry })}
              </p>
            )}
            <div className="mt-4 flex flex-wrap gap-2">
              <button
                type="button"
                onClick={() => acceptMutation.mutate()}
                disabled={acceptMutation.isPending}
                className="inline-flex items-center gap-2 rounded-full bg-[var(--primary)] px-4 py-2 text-[13px] font-semibold text-[var(--primary-foreground)] shadow-[var(--shadow-message)] transition-colors hover:bg-[var(--primary-deep)] disabled:opacity-50"
              >
                {acceptMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {t("acceptTransfer")}
              </button>
              <button
                type="button"
                onClick={() => declineMutation.mutate()}
                disabled={declineMutation.isPending}
                className="inline-flex items-center gap-2 rounded-full border border-[var(--border-strong)] bg-[var(--surface)] px-4 py-2 text-[13px] font-semibold text-[var(--foreground)] transition-colors hover:bg-[var(--background)] disabled:opacity-50"
              >
                {declineMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {t("declineTransfer")}
              </button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // initiator view
  return (
    <div className="rounded-2xl border border-[var(--primary-soft)] bg-gradient-to-br from-[var(--amber-soft)] to-[var(--primary-soft)] p-5 shadow-[var(--shadow-card)]">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <span
            aria-hidden="true"
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-[var(--surface)] text-[var(--warning)]"
          >
            <Crown className="h-5 w-5" strokeWidth={1.8} />
          </span>
          <div>
            <h3 className="font-serif text-[18px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
              {t("pendingTransferInitiatorTitle")}
            </h3>
            <p className="mt-1 font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
              {t("pendingTransferInitiatorDescription")}
            </p>
            {formattedExpiry && (
              <p className="mt-1 font-mono text-[11px] uppercase tracking-[0.05em] text-[var(--subtle-foreground)]">
                {t("pendingTransferExpiresOn", { date: formattedExpiry })}
              </p>
            )}
          </div>
        </div>
        <button
          type="button"
          onClick={() => cancelMutation.mutate()}
          disabled={cancelMutation.isPending}
          className="inline-flex shrink-0 items-center gap-1.5 rounded-full border border-[var(--border-strong)] bg-[var(--surface)] px-3 py-1.5 text-[12px] font-semibold text-[var(--foreground)] transition-colors hover:bg-[var(--background)] disabled:opacity-50"
        >
          {cancelMutation.isPending ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <X className="h-3 w-3" strokeWidth={2} />
          )}
          {t("cancelTransfer")}
        </button>
      </div>
    </div>
  )
}
