"use client"

import { use, useState } from "react"
import { AlertTriangle } from "lucide-react"
import { useTranslations } from "next-intl"

import { useUser } from "@/shared/hooks/use-user"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { ProposalDetailView } from "@/features/proposal/components/proposal-detail-view"
import { useProposal } from "@/features/proposal/hooks/use-proposals"
import { DisputeBanner } from "@/features/dispute/components/dispute-banner"
import { DisputeForm } from "@/features/dispute/components/dispute-form"
import { DisputeCounterForm } from "@/features/dispute/components/dispute-counter-form"
import { DisputeResolutionCard } from "@/features/dispute/components/dispute-resolution-card"
import {
  useDispute,
  useCancelDispute,
  useRespondToCounter,
  useRespondToCancellation,
} from "@/features/dispute/hooks/use-disputes"

import { Button } from "@/shared/components/ui/button"
export default function ProjectDetailPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = use(params)
  const t = useTranslations("disputes")
  const { data: user } = useUser()
  const { data: proposal } = useProposal(id)
  const { data: dispute, refetch: refetchDispute } = useDispute(proposal?.active_dispute_id ?? undefined)
  // Historical dispute fetch — only triggered when there is NO active
  // dispute but a past one exists. Used to render the resolution card so
  // both parties can always see how the dispute ended (split + admin note).
  const historicalDisputeId =
    !proposal?.active_dispute_id && proposal?.last_dispute_id
      ? proposal.last_dispute_id
      : undefined
  const { data: historicalDispute } = useDispute(historicalDisputeId)
  const cancelMutation = useCancelDispute()
  const respondMutation = useRespondToCounter(dispute?.id ?? "")
  const cancelResponseMutation = useRespondToCancellation(dispute?.id ?? "")

  const canRespondProposal = useHasPermission("proposals.respond")
  const [showDisputeForm, setShowDisputeForm] = useState(false)
  const [showCounterForm, setShowCounterForm] = useState(false)

  const canOpenDispute =
    proposal &&
    user &&
    !proposal.active_dispute_id &&
    (proposal.status === "active" || proposal.status === "completion_requested")

  const userRole: "client" | "provider" =
    user?.id === proposal?.client_id ? "client" : "provider"

  return (
    <div>
      {/* Dispute banner — shown when dispute is active */}
      {dispute && proposal?.status === "disputed" && (
        <div className="mx-auto max-w-5xl px-4 pt-8">
          <DisputeBanner
            dispute={dispute}
            currentUserId={user?.id ?? ""}
            onCounterPropose={canRespondProposal ? () => setShowCounterForm(true) : undefined}
            onAcceptProposal={canRespondProposal ? (cpId) =>
              respondMutation.mutate({ cpId, accept: true }, { onSuccess: () => refetchDispute() })
            : undefined}
            onRejectProposal={canRespondProposal ? (cpId) =>
              respondMutation.mutate({ cpId, accept: false }, { onSuccess: () => refetchDispute() })
            : undefined}
            onCancel={
              canRespondProposal && (dispute.status === "open" ||
              dispute.status === "negotiation" ||
              dispute.status === "escalated")
                ? () => cancelMutation.mutate(dispute.id, { onSuccess: () => refetchDispute() })
                : undefined
            }
            onAcceptCancellation={canRespondProposal ? () =>
              cancelResponseMutation.mutate(true, { onSuccess: () => refetchDispute() })
            : undefined}
            onRefuseCancellation={canRespondProposal ? () =>
              cancelResponseMutation.mutate(false, { onSuccess: () => refetchDispute() })
            : undefined}
          />

          {/* Counter-proposal form */}
          {showCounterForm && canRespondProposal && (
            <div className="mt-4 rounded-xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800">
              <DisputeCounterForm
                disputeId={dispute.id}
                proposalAmount={dispute.proposal_amount}
                onSuccess={() => {
                  setShowCounterForm(false)
                  refetchDispute()
                }}
                onCancel={() => setShowCounterForm(false)}
              />
            </div>
          )}
        </div>
      )}

      {/* Historical resolution card — shown when there's no active dispute
          but a past one exists. Lets both parties always see how the
          dispute ended (split + admin note + date). */}
      {historicalDispute && user && (
        <div className="mx-auto max-w-5xl px-4 pt-8">
          <DisputeResolutionCard dispute={historicalDispute} currentUserId={user.id} />
        </div>
      )}

      {/* "Report a problem" button — shown when no dispute exists on active mission */}
      {canOpenDispute && canRespondProposal && !showDisputeForm && (
        <div className="mx-auto max-w-5xl px-4 pt-8">
          <Button variant="ghost" size="auto"
            type="button"
            onClick={() => setShowDisputeForm(true)}
            className="flex items-center gap-2 rounded-lg border border-orange-200 bg-orange-50 px-4 py-2.5 text-sm font-medium text-orange-700 hover:bg-orange-100 transition-colors dark:border-orange-500/20 dark:bg-orange-500/10 dark:text-orange-400"
          >
            <AlertTriangle className="h-4 w-4" />
            {t("openDispute")}
          </Button>
        </div>
      )}

      {/* Dispute form — inline when opening a new dispute.
          A dispute is always scoped to the CURRENT ACTIVE milestone
          (the one in funded / submitted state), not the full proposal:
          resolutions split the escrow that has actually been paid in,
          and the user should only be able to request a refund up to
          that milestone's amount. We resolve it client-side here and
          pass it through to the form. If the current milestone cannot
          be resolved we fall back to the proposal total (legacy
          one-time proposals). */}
      {showDisputeForm && proposal && (
        <div className="mx-auto max-w-5xl px-4 pt-8">
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800">
            <h3 className="mb-4 text-lg font-semibold text-slate-900 dark:text-white">
              {t("openDispute")}
            </h3>
            <DisputeForm
              proposalId={proposal.id}
              proposalAmount={
                proposal.milestones?.find(
                  (m) => m.sequence === proposal.current_milestone_sequence,
                )?.amount ?? proposal.amount
              }
              userRole={userRole}
              onSuccess={() => {
                setShowDisputeForm(false)
                refetchDispute()
              }}
              onCancel={() => setShowDisputeForm(false)}
            />
          </div>
        </div>
      )}

      {/* Existing proposal detail view */}
      <ProposalDetailView proposalId={id} />
    </div>
  )
}
