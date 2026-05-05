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
import type { ProposalResponse, ProposalStatus } from "../types"
import { FeePreview } from "@/shared/components/billing/fee-preview"
import { UpgradeCta } from "@/shared/components/subscription/upgrade-cta"
import { UpgradeModal } from "@/shared/components/subscription/upgrade-modal"
import { Portrait } from "@/shared/components/ui/portrait"
import { Button } from "@/shared/components/ui/button"

// Soleil v2 — Proposal detail view (covers W-10 client + W-15 provider).
// Editorial header per status (corail eyebrow + Fraunces italic-corail
// title + tabac subtitle), 2-col layout on desktop with sticky sidebar,
// Soleil card sections, milestone tracker with progress.

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

  const subscriptionRole: "freelance" | "agency" =
    user?.role === "agency" ? "agency" : "freelance"
  const monthlyPrice = subscriptionRole === "agency" ? 49 : 19

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
    <div className="mx-auto max-w-5xl px-4 pb-24 pt-6 sm:pb-12 sm:pt-8">
      {/* Back button */}
      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={() => router.push("/projects")}
        className={cn(
          "mb-6 flex items-center gap-1.5 px-2 py-1 text-[13px] font-medium",
          "text-muted-foreground hover:text-primary transition-colors duration-150",
        )}
      >
        <ArrowLeft className="h-4 w-4" strokeWidth={1.7} aria-hidden="true" />
        {t("backToProjects")}
      </Button>

      {/* Editorial header */}
      <DetailHeader proposal={proposal} />

      {/* Stepper */}
      <div
        className="mt-6 mb-8 rounded-2xl border border-border bg-card p-6"
        style={{ boxShadow: "var(--shadow-card)" }}
      >
        <ProposalStepper status={proposal.status} />
      </div>

      {/* Milestone tracker */}
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
      <div className="flex flex-col gap-6 lg:flex-row">
        {/* Left column - content */}
        <div className="flex-1 min-w-0 space-y-6">
          <ContentPanel proposal={proposal} />
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
            <div
              className="rounded-2xl border border-border bg-card p-5"
              style={{ boxShadow: "var(--shadow-card)" }}
            >
              <p className="mb-3 font-mono text-[10.5px] font-bold uppercase tracking-[0.1em] text-primary">
                {t("proposalFlow_detail_actionsHeading")}
              </p>
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

function DetailHeader({ proposal }: { proposal: ProposalResponse }) {
  const t = useTranslations("proposal")
  const eyebrow = eyebrowKeyForStatus(proposal.status)

  return (
    <div className="space-y-2">
      <p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
        {t(eyebrow)}
      </p>
      <h1 className="font-serif text-[28px] font-medium leading-[1.05] tracking-[-0.02em] text-foreground sm:text-[34px]">
        {proposal.title}{" "}
        {proposal.version > 1 && (
          <span className="font-mono text-[14px] font-medium text-subtle-foreground align-middle">
            · v{proposal.version}
          </span>
        )}
      </h1>
      <p className="max-w-2xl text-[14.5px] leading-relaxed text-muted-foreground">
        {t("proposalFlow_detail_subtitle")}
      </p>
    </div>
  )
}

function eyebrowKeyForStatus(status: ProposalStatus): string {
  switch (status) {
    case "pending":
      return "proposalFlow_detail_eyebrowPending"
    case "accepted":
    case "paid":
      return "proposalFlow_detail_eyebrowAccepted"
    case "active":
    case "completion_requested":
      return "proposalFlow_detail_eyebrowActive"
    case "completed":
      return "proposalFlow_detail_eyebrowCompleted"
    case "disputed":
      return "proposalFlow_detail_eyebrowDisputed"
    case "declined":
    case "withdrawn":
      return "proposalFlow_detail_eyebrowDeclined"
    default:
      return "proposalFlow_detail_eyebrowDefault"
  }
}

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
    <div
      className="overflow-hidden rounded-2xl border border-border bg-card"
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <div className="space-y-6 p-6">
        {/* Stats row — Geist Mono numerals */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <StatCard
            icon={DollarSign}
            label={t("totalAmount")}
            value={formatCurrency(proposal.amount / 100)}
          />
          <StatCard
            icon={Calendar}
            label={t("deadline")}
            value={
              proposal.deadline ? formatDate(proposal.deadline) : t("noDeadline")
            }
          />
        </div>

        {/* Description */}
        <div>
          <p className="mb-2 font-mono text-[10.5px] font-bold uppercase tracking-[0.1em] text-primary">
            {t("description")}
          </p>
          <p className="whitespace-pre-wrap text-[14.5px] leading-relaxed text-foreground">
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
}

function StatCard({ icon: Icon, label, value }: StatCardProps) {
  return (
    <div className="flex items-center gap-3 rounded-2xl border border-border bg-background p-4">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary-soft text-primary">
        <Icon className="h-4 w-4" strokeWidth={1.7} aria-hidden="true" />
      </div>
      <div className="min-w-0">
        <p className="font-mono text-[10.5px] font-bold uppercase tracking-[0.08em] text-subtle-foreground">
          {label}
        </p>
        <p className="truncate font-mono text-[14.5px] font-semibold text-foreground">
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
      <p className="mb-3 font-mono text-[10.5px] font-bold uppercase tracking-[0.1em] text-primary">
        {t("documents")} ({documents.length})
      </p>
      <div className="space-y-2">
        {documents.map((doc) => (
          <Button
            variant="ghost"
            size="auto"
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
              "flex w-full items-center gap-3 rounded-2xl border border-border bg-card px-4 py-3 text-left",
              "transition-colors duration-150 hover:border-primary hover:bg-primary-soft/30",
            )}
          >
            <FileText className="h-4 w-4 shrink-0 text-subtle-foreground" strokeWidth={1.7} aria-hidden="true" />
            <span className="truncate text-[13.5px] font-medium text-foreground">
              {doc.filename}
            </span>
            <Download className="ml-auto h-4 w-4 shrink-0 text-subtle-foreground" strokeWidth={1.7} aria-hidden="true" />
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
    <div
      className="rounded-2xl border border-border bg-card p-5"
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <p className="mb-3 font-mono text-[10.5px] font-bold uppercase tracking-[0.1em] text-primary">
        {t("participants")}
      </p>
      <div className="space-y-3">
        <ParticipantRow name={clientName} role={t("client")} portraitId={3} />
        <ParticipantRow name={providerName} role={t("provider")} portraitId={1} />
      </div>
    </div>
  )
}

interface ParticipantRowProps {
  name: string
  role: string
  portraitId: number
}

function ParticipantRow({ name, role, portraitId }: ParticipantRowProps) {
  const displayName = name || role

  return (
    <div className="flex items-center gap-3">
      <Portrait id={portraitId} size={36} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13.5px] font-medium text-foreground">
          {displayName}
        </p>
        <p className="font-mono text-[10.5px] font-bold uppercase tracking-[0.08em] text-subtle-foreground">
          {role}
        </p>
      </div>
    </div>
  )
}

function MobileActionBar(props: React.ComponentProps<typeof ActionsPanel>) {
  const hasActions = shouldShowActions(props.proposal, {
    isRecipient: props.isRecipient,
    isClient: props.isClient,
    isProvider: props.isProvider,
    currentMilestone: props.currentMilestone,
  })
  if (!hasActions) return null

  return (
    <div
      className={cn(
        "fixed bottom-0 left-0 right-0 z-40 border-t border-border bg-card p-4 lg:hidden",
        "glass-strong",
      )}
    >
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
      <div className="space-y-3 text-center">
        <FileText className="mx-auto h-10 w-10 text-subtle-foreground" strokeWidth={1.5} aria-hidden="true" />
        <p className="text-[13.5px] text-muted-foreground">
          {t("proposalNotFound")}
        </p>
        <Button
          variant="ghost"
          size="auto"
          type="button"
          onClick={onBack}
          className="text-[13.5px] font-bold text-primary hover:text-primary-deep"
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
      <div className="mb-6 h-5 w-32 animate-shimmer rounded bg-border" />
      <div className="mb-8 space-y-2">
        <div className="h-3 w-40 animate-shimmer rounded bg-border" />
        <div className="h-9 w-3/4 animate-shimmer rounded bg-border" />
        <div className="h-3 w-2/3 animate-shimmer rounded bg-border/60" />
      </div>
      <div
        className="mb-8 rounded-2xl border border-border bg-card p-6"
        style={{ boxShadow: "var(--shadow-card)" }}
      >
        <div className="flex items-center justify-between">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="flex flex-1 items-center last:flex-none">
              <div className="flex flex-col items-center gap-1.5">
                <div className="h-8 w-8 animate-shimmer rounded-full bg-border" />
                <div className="h-3 w-12 animate-shimmer rounded bg-border/60" />
              </div>
              {i < 5 && <div className="mx-2 mt-[-1.25rem] h-px flex-1 bg-border" />}
            </div>
          ))}
        </div>
      </div>
      <div className="flex flex-col gap-6 lg:flex-row">
        <div className="flex-1">
          <div className="space-y-6 rounded-2xl border border-border bg-card p-6">
            <div className="grid grid-cols-2 gap-3">
              <div className="h-16 animate-shimmer rounded-2xl bg-border/60" />
              <div className="h-16 animate-shimmer rounded-2xl bg-border/60" />
            </div>
            <div className="space-y-2">
              <div className="h-3 w-full animate-shimmer rounded bg-border/60" />
              <div className="h-3 w-3/4 animate-shimmer rounded bg-border/60" />
              <div className="h-3 w-1/2 animate-shimmer rounded bg-border/60" />
            </div>
          </div>
        </div>
        <div className="w-full lg:w-80">
          <div className="rounded-2xl border border-border bg-card p-5">
            <div className="h-24 animate-shimmer rounded-2xl bg-border/60" />
          </div>
        </div>
      </div>
    </div>
  )
}
