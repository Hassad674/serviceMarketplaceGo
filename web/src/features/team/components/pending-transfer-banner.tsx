"use client"

import { Crown, Loader2, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { useAcceptTransfer, useDeclineTransfer, useCancelTransfer } from "../hooks/use-team"

import { Button } from "@/shared/components/ui/button"
// Banner shown at the top of /team whenever a transfer is pending.
//
// Two flavours driven by `viewerRole`:
//   - "target": the current user is the proposed new Owner → shows
//     Accept/Decline.
//   - "initiator": the current user is the Owner who initiated the
//     transfer → shows "En attente" + a Cancel button.
//
// All other roles never see this banner (Admin/Member/Viewer who are
// not the target are not stakeholders).

type PendingTransferBannerProps = {
  orgID: string
  viewerRole: "target" | "initiator"
  expiresAt?: string
}

export function PendingTransferBanner({
  orgID,
  viewerRole,
  expiresAt,
}: PendingTransferBannerProps) {
  const t = useTranslations("team")
  const acceptMutation = useAcceptTransfer(orgID)
  const declineMutation = useDeclineTransfer(orgID)
  const cancelMutation = useCancelTransfer(orgID)

  const formattedExpiry = expiresAt ? new Date(expiresAt).toLocaleDateString() : undefined

  if (viewerRole === "target") {
    return (
      <div className="rounded-xl border border-amber-200 dark:border-amber-500/30 bg-amber-50 dark:bg-amber-500/10 p-5">
        <div className="flex items-start gap-3">
          <Crown className="h-5 w-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <h3 className="text-sm font-semibold text-amber-900 dark:text-amber-100">
              {t("pendingTransferTargetTitle")}
            </h3>
            <p className="mt-1 text-sm text-amber-800 dark:text-amber-200">
              {t("pendingTransferTargetDescription")}
            </p>
            {formattedExpiry && (
              <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
                {t("pendingTransferExpiresOn", { date: formattedExpiry })}
              </p>
            )}
            <div className="mt-4 flex gap-3">
              <Button variant="ghost" size="auto"
                type="button"
                onClick={() => acceptMutation.mutate()}
                disabled={acceptMutation.isPending}
                className="inline-flex items-center gap-2 rounded-lg bg-amber-500 px-4 py-2 text-sm font-semibold text-white hover:bg-amber-600 disabled:opacity-50"
              >
                {acceptMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {t("acceptTransfer")}
              </Button>
              <Button variant="ghost" size="auto"
                type="button"
                onClick={() => declineMutation.mutate()}
                disabled={declineMutation.isPending}
                className="inline-flex items-center gap-2 rounded-lg border border-amber-300 dark:border-amber-500/40 px-4 py-2 text-sm font-semibold text-amber-800 dark:text-amber-200 hover:bg-amber-100 dark:hover:bg-amber-500/20 disabled:opacity-50"
              >
                {declineMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
                {t("declineTransfer")}
              </Button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // initiator view
  return (
    <div className="rounded-xl border border-amber-200 dark:border-amber-500/30 bg-amber-50 dark:bg-amber-500/10 p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-3">
          <Crown className="h-5 w-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
          <div>
            <h3 className="text-sm font-semibold text-amber-900 dark:text-amber-100">
              {t("pendingTransferInitiatorTitle")}
            </h3>
            <p className="mt-1 text-sm text-amber-800 dark:text-amber-200">
              {t("pendingTransferInitiatorDescription")}
            </p>
            {formattedExpiry && (
              <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
                {t("pendingTransferExpiresOn", { date: formattedExpiry })}
              </p>
            )}
          </div>
        </div>
        <Button variant="ghost" size="auto"
          type="button"
          onClick={() => cancelMutation.mutate()}
          disabled={cancelMutation.isPending}
          className="inline-flex items-center gap-1 rounded-lg border border-amber-300 dark:border-amber-500/40 px-3 py-1.5 text-xs font-medium text-amber-800 dark:text-amber-200 hover:bg-amber-100 dark:hover:bg-amber-500/20 disabled:opacity-50"
        >
          {cancelMutation.isPending ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <X className="h-3 w-3" />
          )}
          {t("cancelTransfer")}
        </Button>
      </div>
    </div>
  )
}
