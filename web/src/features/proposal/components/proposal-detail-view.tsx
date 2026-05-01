"use client"

import { useState } from "react"
import {
  ArrowLeft,
  Calendar,
  DollarSign,
  Download,
  FileText,
} from "lucide-react"
import { useRouter } from "@i18n/navigation"
import { useTranslations } from "next-intl"
import { cn, formatCurrency, formatDate } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import {
  useProposal,
  useAcceptProposal,
  useDeclineProposal,
  useSubmitMilestone,
  useApproveMilestone,
  useRejectMilestone,
} from "../hooks/use-proposals"
import { ProposalStepper } from "./proposal-stepper"
import { ActionsPanel, type ActionsPanelProps } from "./proposal-actions-panel"
import { MilestoneTracker } from "./milestone-tracker"
import type { ProposalResponse } from "../types"
import { FeePreview } from "@/features/billing/components/fee-preview"
import { UpgradeCta } from "@/features/subscription/components/upgrade-cta"
import { UpgradeModal } from "@/features/subscription/components/upgrade-modal"
import { Button } from "@/shared/components/ui/button"

interface ProposalDetailViewProps {
  proposalId: string
}

export function ProposalDetailView({ proposalId }: ProposalDetailViewProps) {
  const t = useTranslations("proposal")
  const router = useRouter()
  const { data: user } = useUser()
  const { data: proposal, isLoading, isError } = useProposal(proposalId)
  const acceptMutation = useAcceptProposal()
  const declineMutation = useDeclineProposal()
  const submitMilestoneMutation = useSubmitMilestone()
  const approveMilestoneMutation = useApproveMilestone()
  const rejectMilestoneMutation = useRejectMilestone()
  const [upgradeOpen, setUpgradeOpen] = useState(false)

  const isMutating =
    acceptMutation.isPending ||
    declineMutation.isPending ||
    submitMilestoneMutation.isPending ||
    approveMilestoneMutation.isPending ||
    rejectMilestoneMutation.isPending

  if (isLoading) {
    return <DetailSkeleton />
  }

  if (isError || !proposal) {
    return <ErrorState onBack={() => router.push("/projects")} />
  }

  const isRecipient = user?.id === proposal.recipient_id
  const isSender = user?.id === proposal.sender_id
  const isClient = user?.id === proposal.client_id
  const isProvider = user?.id === proposal.provider_id

  // Subscription role pricing for the upgrade CTA. Only providers /
  // agencies ever reach this code path (isProvider is true), so the
  // fallback branch is defensive — enterprises can never own a
  // subscription.
  const subscriptionRole: "freelance" | "agency" =
    user?.role === "agency" ? "agency" : "freelance"
  const monthlyPrice = subscriptionRole === "agency" ? 49 : 19

  // The backend sets current_milestone_sequence to the active milestone's
  // sequence while the proposal is in a milestone-driven state (active,
  // completion_requested, etc.). Grab the matching milestone so action
  // handlers can pass its id to the per-milestone endpoints — a stale
  // client view returns 409 on the action, and the TanStack Query
  // invalidation on success will refetch the fresh sequence.
  const currentMilestone = proposal.milestones?.find(
    (m) => m.sequence === proposal.current_milestone_sequence,
  )

  function handleAccept() {
    acceptMutation.mutate(proposalId)
  }

  function handleDecline() {
    declineMutation.mutate(proposalId)
  }

  function handleModify() {
    if (!proposal) return
    const params = new URLSearchParams({
      modify: proposalId,
      conversation: proposal.conversation_id,
      to: isSender ? proposal.recipient_id : proposal.sender_id,
    })
    router.push(`/projects/new?${params.toString()}`)
  }

  function handlePay() {
    router.push(`/projects/pay?proposal=${proposalId}`)
  }

  function handleRequestCompletion() {
    if (!currentMilestone) return
    submitMilestoneMutation.mutate({ proposalID: proposalId, milestoneID: currentMilestone.id })
  }

  function handleCompleteProposal() {
    if (!currentMilestone) return
    approveMilestoneMutation.mutate({ proposalID: proposalId, milestoneID: currentMilestone.id })
  }

  function handleRejectCompletion() {
    if (!currentMilestone) return
    rejectMilestoneMutation.mutate({ proposalID: proposalId, milestoneID: currentMilestone.id })
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      {/* Back button */}
      <Button variant="ghost" size="auto"
        type="button"
        onClick={() => router.push("/projects")}
        className="mb-6 flex items-center gap-1.5 text-sm text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200 transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        {t("backToProjects")}
      </Button>

      {/* Stepper (proposal-level macro state) */}
      <div className="mb-8 rounded-2xl border border-slate-100 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
        <ProposalStepper status={proposal.status} />
      </div>

      {/* Milestone tracker (phase 11). Renders the project's milestone
          list as a vertical timeline for milestone-mode proposals, or
          collapses to a compact single card for one-time mode. */}
      {proposal.milestones && proposal.milestones.length > 0 && (
        <div className="mb-8">
          <MilestoneTracker
            milestones={proposal.milestones}
            paymentMode={proposal.payment_mode}
            currentSequence={proposal.current_milestone_sequence}
          />
        </div>
      )}

      {/* Split layout */}
      <div className="flex flex-col lg:flex-row gap-6">
        {/* Left column - content */}
        <div className="flex-1 min-w-0 space-y-6">
          <ContentPanel proposal={proposal} />
          {/* Platform fee preview — only the designated provider ever
              sees this panel. The backend also fails closed via
              `viewer_is_provider`, so even a stale client view cannot
              reveal fee data to a client. */}
          {isProvider && (
            <FeePreview
              mode={proposal.payment_mode}
              milestones={buildDetailFeeMilestones(proposal)}
              heading="Frais plateforme estimés pour ta mission"
              renderPremiumCta={
                <UpgradeCta
                  variant="inline"
                  onClick={() => setUpgradeOpen(true)}
                  monthlyPrice={monthlyPrice}
                />
              }
            />
          )}
        </div>

        {/* Right column - actions (sticky on desktop) */}
        <div className="w-full lg:w-80 shrink-0">
          <div className="lg:sticky lg:top-24 space-y-4">
            <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
              <ActionsPanel
                proposal={proposal}
                currentMilestone={currentMilestone}
                isRecipient={isRecipient}
                isSender={isSender}
                isClient={isClient}
                isProvider={isProvider}
                isMutating={isMutating}
                acceptPending={acceptMutation.isPending}
                declinePending={declineMutation.isPending}
                requestCompletionPending={submitMilestoneMutation.isPending}
                completePending={approveMilestoneMutation.isPending}
                rejectCompletionPending={rejectMilestoneMutation.isPending}
                onAccept={handleAccept}
                onDecline={handleDecline}
                onModify={handleModify}
                onPay={handlePay}
                onRequestCompletion={handleRequestCompletion}
                onCompleteProposal={handleCompleteProposal}
                onRejectCompletion={handleRejectCompletion}
              />
            </div>
            <ParticipantsCard
              clientName={proposal.client_name}
              providerName={proposal.provider_name}
            />
          </div>
        </div>
      </div>

      {/* Mobile sticky action bar */}
      <MobileActionBar
        proposal={proposal}
        currentMilestone={currentMilestone}
        isRecipient={isRecipient}
        isSender={isSender}
        isClient={isClient}
        isProvider={isProvider}
        isMutating={isMutating}
        acceptPending={acceptMutation.isPending}
        declinePending={declineMutation.isPending}
        requestCompletionPending={submitMilestoneMutation.isPending}
        completePending={approveMilestoneMutation.isPending}
        rejectCompletionPending={rejectMilestoneMutation.isPending}
        onAccept={handleAccept}
        onDecline={handleDecline}
        onModify={handleModify}
        onPay={handlePay}
        onRequestCompletion={handleRequestCompletion}
        onCompleteProposal={handleCompleteProposal}
        onRejectCompletion={handleRejectCompletion}
      />
      {isProvider && (
        <UpgradeModal
          open={upgradeOpen}
          role={subscriptionRole}
          onClose={() => setUpgradeOpen(false)}
        />
      )}
    </div>
  )
}

// buildDetailFeeMilestones maps the received proposal into the shape
// expected by <FeePreview>. In milestone mode each milestone is
// forwarded with its sequence/title and amount; in one-time mode a
// single synthetic "Paiement unique" entry is emitted with the total
// proposal amount so the one-time summary fires.
function buildDetailFeeMilestones(
  proposal: ProposalResponse,
): { key: string; label: string; amountCents: number }[] {
  if (proposal.payment_mode === "milestone" && proposal.milestones.length > 0) {
    return proposal.milestones.map((m) => ({
      key: m.id,
      label: m.title || `Jalon ${m.sequence}`,
      amountCents: m.amount,
    }))
  }
  return [
    {
      key: "one-time",
      label: "Paiement unique",
      amountCents: proposal.amount,
    },
  ]
}

function ContentPanel({ proposal }: { proposal: ProposalResponse }) {
  const t = useTranslations("proposal")

  return (
    <div className="rounded-2xl border border-slate-100 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800/80 overflow-hidden">
      <div className="h-1 gradient-primary" />
      <div className="p-6 space-y-6">
        {/* Title */}
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
            {proposal.title}
          </h1>
          {proposal.version > 1 && (
            <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
              {t("versionLabel", { version: proposal.version })}
            </p>
          )}
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <StatCard
            icon={DollarSign}
            label={t("totalAmount")}
            value={formatCurrency(proposal.amount / 100)}
            iconBg="bg-green-50 dark:bg-green-500/10"
            iconColor="text-green-600 dark:text-green-400"
          />
          <StatCard
            icon={Calendar}
            label={t("deadline")}
            value={
              proposal.deadline
                ? formatDate(proposal.deadline)
                : t("noDeadline")
            }
            iconBg="bg-blue-50 dark:bg-blue-500/10"
            iconColor="text-blue-600 dark:text-blue-400"
          />
        </div>

        {/* Description */}
        <div>
          <p className="text-xs font-medium uppercase tracking-wide text-slate-400 dark:text-slate-500 mb-2">
            {t("description")}
          </p>
          <p className="text-sm leading-relaxed text-slate-700 dark:text-slate-300 whitespace-pre-wrap">
            {proposal.description}
          </p>
        </div>

        {/* Documents */}
        {proposal.documents && proposal.documents.length > 0 && (
          <DocumentsList documents={proposal.documents} />
        )}
      </div>
    </div>
  )
}

interface StatCardProps {
  icon: React.ElementType
  label: string
  value: string
  iconBg: string
  iconColor: string
}

function StatCard({ icon: Icon, label, value, iconBg, iconColor }: StatCardProps) {
  return (
    <div className="flex items-center gap-3 rounded-xl bg-slate-50 p-4 border border-slate-100 dark:bg-slate-800 dark:border-slate-700">
      <div className={cn("flex h-10 w-10 shrink-0 items-center justify-center rounded-lg", iconBg)}>
        <Icon className={cn("h-5 w-5", iconColor)} strokeWidth={1.5} />
      </div>
      <div className="min-w-0">
        <p className="text-xs text-slate-500 dark:text-slate-400">{label}</p>
        <p className="text-sm font-semibold text-slate-900 dark:text-white truncate">
          {value}
        </p>
      </div>
    </div>
  )
}

function DocumentsList({ documents }: { documents: ProposalResponse["documents"] }) {
  const t = useTranslations("proposal")

  return (
    <div>
      <p className="text-xs font-medium uppercase tracking-wide text-slate-400 dark:text-slate-500 mb-3">
        {t("documents")} ({documents.length})
      </p>
      <div className="space-y-2">
        {documents.map((doc) => (
          <Button variant="ghost" size="auto"
            key={doc.id}
            type="button"
            onClick={async () => {
              try {
                const res = await fetch(doc.url)
                const blob = await res.blob()
                const blobUrl = URL.createObjectURL(blob)
                const link = document.createElement("a")
                link.href = blobUrl
                link.download = doc.filename
                document.body.appendChild(link)
                link.click()
                document.body.removeChild(link)
                URL.revokeObjectURL(blobUrl)
              } catch {
                window.open(doc.url, "_blank")
              }
            }}
            className={cn(
              "flex w-full items-center gap-3 rounded-xl px-4 py-3 text-left",
              "border border-slate-100 dark:border-slate-700",
              "hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors",
            )}
          >
            <FileText className="h-4 w-4 shrink-0 text-slate-400" strokeWidth={1.5} />
            <span className="text-sm font-medium text-slate-700 dark:text-slate-300 truncate">
              {doc.filename}
            </span>
            <Download className="ml-auto h-4 w-4 shrink-0 text-slate-400" strokeWidth={1.5} />
          </Button>
        ))}
      </div>
    </div>
  )
}

interface ParticipantsCardProps {
  clientName: string
  providerName: string
}

function ParticipantsCard({ clientName, providerName }: ParticipantsCardProps) {
  const t = useTranslations("proposal")

  return (
    <div className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
      <p className="text-xs font-medium uppercase tracking-wide text-slate-400 dark:text-slate-500 mb-3">
        {t("participants")}
      </p>
      <div className="space-y-3">
        <ParticipantRow
          name={clientName}
          role={t("client")}
          badgeClass="bg-purple-100 text-purple-700 dark:bg-purple-500/20 dark:text-purple-400"
          avatarClass="bg-purple-100 text-purple-600 dark:bg-purple-500/20 dark:text-purple-400"
        />
        <ParticipantRow
          name={providerName}
          role={t("provider")}
          badgeClass="bg-rose-100 text-rose-700 dark:bg-rose-500/20 dark:text-rose-400"
          avatarClass="bg-rose-100 text-rose-600 dark:bg-rose-500/20 dark:text-rose-400"
        />
      </div>
    </div>
  )
}

interface ParticipantRowProps {
  name: string
  role: string
  badgeClass: string
  avatarClass: string
}

function ParticipantRow({ name, role, badgeClass, avatarClass }: ParticipantRowProps) {
  const displayName = name || role
  const initial = displayName.charAt(0).toUpperCase()

  return (
    <div className="flex items-center gap-3">
      <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-sm font-semibold", avatarClass)}>
        {initial}
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
          {displayName}
        </p>
      </div>
      <span className={cn("shrink-0 rounded-full px-2 py-0.5 text-xs font-medium", badgeClass)}>
        {role}
      </span>
    </div>
  )
}

function MobileActionBar(props: React.ComponentProps<typeof ActionsPanel>) {
  // Only show on mobile when there are actions
  const hasActions = shouldShowActions(props.proposal, {
    isRecipient: props.isRecipient,
    isClient: props.isClient,
    isProvider: props.isProvider,
    currentMilestone: props.currentMilestone,
  })
  if (!hasActions) return null

  return (
    <div className="fixed bottom-0 left-0 right-0 z-40 border-t border-slate-200 bg-white/90 backdrop-blur-xl p-4 lg:hidden dark:border-slate-700 dark:bg-slate-900/90">
      <ActionsPanel {...props} />
    </div>
  )
}

function shouldShowActions(
  proposal: ProposalResponse,
  flags: {
    isRecipient: boolean
    isClient: boolean
    isProvider: boolean
    currentMilestone: ActionsPanelProps["currentMilestone"]
  },
): boolean {
  if (proposal.status === "pending" && flags.isRecipient) return true
  if (proposal.status === "accepted" && flags.isClient) return true
  if (proposal.status === "active") {
    // In the milestone world, `active` maps to two different CTAs
    // depending on the current milestone's sub-state:
    //   pending_funding → client funds the next milestone
    //   funded          → provider submits it for approval
    if (
      flags.isClient &&
      flags.currentMilestone?.status === "pending_funding"
    ) {
      return true
    }
    if (
      flags.isProvider &&
      flags.currentMilestone?.status === "funded"
    ) {
      return true
    }
  }
  if (proposal.status === "completion_requested" && flags.isClient) return true
  return false
}

function ErrorState({ onBack }: { onBack: () => void }) {
  const t = useTranslations("proposal")

  return (
    <div className="flex min-h-[60vh] items-center justify-center">
      <div className="text-center space-y-3">
        <FileText className="mx-auto h-10 w-10 text-slate-300 dark:text-slate-600" />
        <p className="text-sm text-slate-500 dark:text-slate-400">
          {t("proposalNotFound")}
        </p>
        <Button variant="ghost" size="auto"
          type="button"
          onClick={onBack}
          className="text-sm text-rose-500 hover:text-rose-600 font-medium"
        >
          {t("backToProjects")}
        </Button>
      </div>
    </div>
  )
}

function DetailSkeleton() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <div className="h-5 w-32 animate-shimmer rounded bg-slate-200 dark:bg-slate-700 mb-6" />
      {/* Stepper skeleton */}
      <div className="mb-8 rounded-2xl border border-slate-100 bg-white p-6 dark:border-slate-700 dark:bg-slate-800/80">
        <div className="flex items-center justify-between">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="flex items-center flex-1 last:flex-none">
              <div className="flex flex-col items-center gap-1.5">
                <div className="h-8 w-8 animate-shimmer rounded-full bg-slate-200 dark:bg-slate-700" />
                <div className="h-3 w-12 animate-shimmer rounded bg-slate-100 dark:bg-slate-700" />
              </div>
              {i < 5 && <div className="flex-1 h-0.5 mx-2 mt-[-1.25rem] bg-slate-200 dark:bg-slate-700" />}
            </div>
          ))}
        </div>
      </div>
      {/* Content skeleton */}
      <div className="flex flex-col lg:flex-row gap-6">
        <div className="flex-1">
          <div className="rounded-2xl border border-slate-100 bg-white p-6 dark:border-slate-700 dark:bg-slate-800/80 space-y-6">
            <div className="h-7 w-3/4 animate-shimmer rounded bg-slate-200 dark:bg-slate-700" />
            <div className="grid grid-cols-2 gap-3">
              <div className="h-16 animate-shimmer rounded-xl bg-slate-100 dark:bg-slate-700" />
              <div className="h-16 animate-shimmer rounded-xl bg-slate-100 dark:bg-slate-700" />
            </div>
            <div className="space-y-2">
              <div className="h-3 w-full animate-shimmer rounded bg-slate-100 dark:bg-slate-700" />
              <div className="h-3 w-3/4 animate-shimmer rounded bg-slate-100 dark:bg-slate-700" />
              <div className="h-3 w-1/2 animate-shimmer rounded bg-slate-100 dark:bg-slate-700" />
            </div>
          </div>
        </div>
        <div className="w-full lg:w-80">
          <div className="rounded-2xl border border-slate-100 bg-white p-5 dark:border-slate-700 dark:bg-slate-800/80">
            <div className="h-24 animate-shimmer rounded-xl bg-slate-100 dark:bg-slate-700" />
          </div>
        </div>
      </div>
    </div>
  )
}
