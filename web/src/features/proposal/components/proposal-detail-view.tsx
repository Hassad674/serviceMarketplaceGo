"use client"

import {
  ArrowLeft,
  Calendar,
  CheckCircle2,
  Clock,
  CreditCard,
  DollarSign,
  Download,
  FileText,
  Handshake,
  Loader2,
  Pencil,
  Star,
  XCircle,
} from "lucide-react"
import { useRouter } from "@i18n/navigation"
import { useTranslations } from "next-intl"
import { cn, formatCurrency, formatDate } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import {
  useProposal,
  useAcceptProposal,
  useDeclineProposal,
  useRequestCompletion,
  useCompleteProposal,
  useRejectCompletion,
} from "../hooks/use-proposals"
import type { ProposalResponse, ProposalStatus } from "../types"

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

  const isMutating = acceptMutation.isPending || declineMutation.isPending

  if (isLoading) {
    return <DetailSkeleton />
  }

  if (isError || !proposal) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="text-center space-y-3">
          <FileText className="mx-auto h-10 w-10 text-gray-300 dark:text-gray-600" />
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t("proposalNotFound")}
          </p>
          <button
            type="button"
            onClick={() => router.push("/projects")}
            className="text-sm text-rose-500 hover:text-rose-600 font-medium"
          >
            {t("backToProjects")}
          </button>
        </div>
      </div>
    )
  }

  const isRecipient = user?.id === proposal.recipient_id
  const isSender = user?.id === proposal.sender_id
  const isClient = user?.id === proposal.client_id

  function handleAccept() {
    acceptMutation.mutate(proposalId, {
      onSuccess: () => {
        if (isClient) {
          router.push(`/projects/pay?proposal=${proposalId}`)
        }
      },
    })
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

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      {/* Back button */}
      <button
        type="button"
        onClick={() => router.push("/projects")}
        className="mb-6 flex items-center gap-1.5 text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        {t("backToProjects")}
      </button>

      <div className="rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800/80 overflow-hidden">
        {/* Gradient bar */}
        <div className="h-1.5 gradient-primary" />

        <div className="px-6 pt-6 pb-8 space-y-6">
          {/* Header */}
          <div className="flex items-start justify-between gap-4">
            <div className="flex items-center gap-3 min-w-0">
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
                <Handshake className="h-6 w-6 text-rose-600 dark:text-rose-400" strokeWidth={1.5} />
              </div>
              <div className="min-w-0">
                <h1 className="text-lg font-bold text-gray-900 dark:text-white truncate">
                  {proposal.title}
                </h1>
                {proposal.version > 1 && (
                  <p className="text-xs text-gray-500 dark:text-gray-400">
                    {t("versionLabel", { version: proposal.version })}
                  </p>
                )}
              </div>
            </div>
            <StatusBadge status={proposal.status} />
          </div>

          <div className="border-t border-gray-100 dark:border-gray-700" />

          {/* Amount */}
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-green-50 dark:bg-green-500/10">
              <DollarSign className="h-5 w-5 text-green-600 dark:text-green-400" strokeWidth={1.5} />
            </div>
            <div>
              <p className="text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                {t("totalAmount")}
              </p>
              <p className="text-xl font-bold text-gray-900 dark:text-white">
                {formatCurrency(proposal.amount / 100)}
              </p>
            </div>
          </div>

          {/* Deadline */}
          {proposal.deadline && (
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-blue-50 dark:bg-blue-500/10">
                <Calendar className="h-5 w-5 text-blue-600 dark:text-blue-400" strokeWidth={1.5} />
              </div>
              <div>
                <p className="text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                  {t("deadline")}
                </p>
                <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {formatDate(proposal.deadline)}
                </p>
              </div>
            </div>
          )}

          <div className="border-t border-gray-100 dark:border-gray-700" />

          {/* Description */}
          <div>
            <p className="text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500 mb-2">
              {t("description")}
            </p>
            <p className="text-sm leading-relaxed text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
              {proposal.description}
            </p>
          </div>

          {/* Documents */}
          {proposal.documents && proposal.documents.length > 0 && (
            <>
              <div className="border-t border-gray-100 dark:border-gray-700" />
              <div>
                <p className="text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500 mb-3">
                  {t("documents")} ({proposal.documents.length})
                </p>
                <div className="space-y-2">
                  {proposal.documents.map((doc) => (
                    <a
                      key={doc.id}
                      href={doc.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className={cn(
                        "flex items-center gap-3 rounded-xl px-4 py-3",
                        "border border-gray-100 dark:border-gray-700",
                        "hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors",
                      )}
                    >
                      <FileText className="h-4 w-4 shrink-0 text-gray-400" strokeWidth={1.5} />
                      <span className="text-sm font-medium text-gray-700 dark:text-gray-300 truncate">
                        {doc.filename}
                      </span>
                      <Download className="ml-auto h-4 w-4 shrink-0 text-gray-400" strokeWidth={1.5} />
                    </a>
                  ))}
                </div>
              </div>
            </>
          )}

          <div className="border-t border-gray-100 dark:border-gray-700" />

          {/* Action section */}
          <ProposalActions
            proposal={proposal}
            isRecipient={isRecipient}
            isSender={isSender}
            isClient={isClient}
            isMutating={isMutating}
            acceptPending={acceptMutation.isPending}
            declinePending={declineMutation.isPending}
            onAccept={handleAccept}
            onDecline={handleDecline}
            onModify={handleModify}
            onPay={handlePay}
          />
        </div>
      </div>
    </div>
  )
}

interface ProposalActionsProps {
  proposal: ProposalResponse
  isRecipient: boolean
  isSender: boolean
  isClient: boolean
  isMutating: boolean
  acceptPending: boolean
  declinePending: boolean
  onAccept: () => void
  onDecline: () => void
  onModify: () => void
  onPay: () => void
}

function ProposalActions({
  proposal,
  isRecipient,
  isSender,
  isClient,
  isMutating,
  acceptPending,
  declinePending,
  onAccept,
  onDecline,
  onModify,
  onPay,
}: ProposalActionsProps) {
  const t = useTranslations("proposal")

  // Pending — recipient can accept/decline
  if (proposal.status === "pending" && isRecipient) {
    return (
      <div className="space-y-3">
        <div className="flex gap-3">
          <button
            type="button"
            onClick={onDecline}
            disabled={isMutating}
            className={cn(
              "flex-1 flex items-center justify-center gap-2 rounded-xl px-4 py-3",
              "text-sm font-medium transition-all duration-200",
              "border border-gray-200 dark:border-gray-600",
              "text-gray-700 dark:text-gray-300",
              "hover:bg-gray-50 dark:hover:bg-gray-700 hover:border-gray-300",
              "active:scale-[0.98]",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            {declinePending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <XCircle className="h-4 w-4" strokeWidth={1.5} />
            )}
            {t("decline")}
          </button>
          <button
            type="button"
            onClick={onAccept}
            disabled={isMutating}
            className={cn(
              "flex-1 flex items-center justify-center gap-2 rounded-xl px-4 py-3",
              "text-sm font-semibold text-white transition-all duration-200",
              "gradient-primary",
              "hover:shadow-glow active:scale-[0.98]",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            {acceptPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <CheckCircle2 className="h-4 w-4" strokeWidth={1.5} />
            )}
            {t("accept")}
          </button>
        </div>
        <button
          type="button"
          onClick={onModify}
          className={cn(
            "w-full flex items-center justify-center gap-2 rounded-xl px-4 py-2.5",
            "text-sm font-medium transition-all duration-200",
            "border border-gray-200 dark:border-gray-600",
            "text-gray-700 dark:text-gray-300",
            "hover:bg-gray-50 dark:hover:bg-gray-700",
            "active:scale-[0.98]",
          )}
        >
          <Pencil className="h-4 w-4" strokeWidth={1.5} />
          {t("modify")}
        </button>
      </div>
    )
  }

  // Pending — sender sees waiting state
  if (proposal.status === "pending" && isSender) {
    return (
      <div className="flex items-center justify-center gap-2 rounded-xl bg-amber-50 px-4 py-3 dark:bg-amber-500/10">
        <Clock className="h-4 w-4 text-amber-600 dark:text-amber-400" strokeWidth={1.5} />
        <span className="text-sm font-medium text-amber-700 dark:text-amber-300">
          {t("waitingForResponse")}
        </span>
      </div>
    )
  }

  // Accepted — client can proceed to payment
  if (proposal.status === "accepted" && isClient) {
    return (
      <button
        type="button"
        onClick={onPay}
        className={cn(
          "w-full flex items-center justify-center gap-2 rounded-xl px-5 py-3",
          "text-sm font-semibold text-white transition-all duration-200",
          "gradient-primary hover:shadow-glow active:scale-[0.98]",
        )}
      >
        <CreditCard className="h-4 w-4" strokeWidth={1.5} />
        {t("proceedToPayment")}
      </button>
    )
  }

  // Accepted — provider waits for payment
  if (proposal.status === "accepted" && !isClient) {
    return (
      <div className="flex items-center justify-center gap-2 rounded-xl bg-blue-50 px-4 py-3 dark:bg-blue-500/10">
        <CreditCard className="h-4 w-4 text-blue-600 dark:text-blue-400" strokeWidth={1.5} />
        <span className="text-sm font-medium text-blue-700 dark:text-blue-300">
          {t("waitingForPayment")}
        </span>
      </div>
    )
  }

  // Paid / Active
  if (proposal.status === "paid" || proposal.status === "active") {
    return (
      <div className="flex items-center justify-center gap-2 rounded-xl bg-emerald-50 px-4 py-3 dark:bg-emerald-500/10">
        <Star className="h-4 w-4 text-emerald-600 dark:text-emerald-400" strokeWidth={1.5} />
        <span className="text-sm font-medium text-emerald-700 dark:text-emerald-300">
          {t("missionActive")}
        </span>
      </div>
    )
  }

  // Declined
  if (proposal.status === "declined") {
    return (
      <div className="flex items-center justify-center gap-2 rounded-xl bg-red-50 px-4 py-3 dark:bg-red-500/10">
        <XCircle className="h-4 w-4 text-red-600 dark:text-red-400" strokeWidth={1.5} />
        <span className="text-sm font-medium text-red-700 dark:text-red-300">
          {t("proposalRefused")}
        </span>
      </div>
    )
  }

  // Withdrawn
  if (proposal.status === "withdrawn") {
    return (
      <div className="flex items-center justify-center gap-2 rounded-xl bg-gray-50 px-4 py-3 dark:bg-gray-500/10">
        <XCircle className="h-4 w-4 text-gray-500 dark:text-gray-400" strokeWidth={1.5} />
        <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
          {t("proposalWithdrawn")}
        </span>
      </div>
    )
  }

  return null
}

function StatusBadge({ status }: { status: ProposalStatus }) {
  const t = useTranslations("proposal")

  const config: Record<string, { label: string; icon: React.ElementType; className: string }> = {
    pending: {
      label: t("pending"),
      icon: Clock,
      className: "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400",
    },
    accepted: {
      label: t("accepted"),
      icon: CheckCircle2,
      className: "bg-green-50 text-green-700 dark:bg-green-500/10 dark:text-green-400",
    },
    declined: {
      label: t("declined"),
      icon: XCircle,
      className: "bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-400",
    },
    withdrawn: {
      label: t("withdrawn"),
      icon: XCircle,
      className: "bg-gray-50 text-gray-600 dark:bg-gray-500/10 dark:text-gray-400",
    },
    paid: {
      label: t("paid"),
      icon: DollarSign,
      className: "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400",
    },
    active: {
      label: t("active"),
      icon: Star,
      className: "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400",
    },
    completed: {
      label: t("completed"),
      icon: CheckCircle2,
      className: "bg-gray-50 text-gray-600 dark:bg-gray-500/10 dark:text-gray-400",
    },
  }

  const entry = config[status] ?? config.pending
  const { label, icon: StatusIcon, className } = entry

  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center gap-1 rounded-full px-3 py-1 text-xs font-medium",
        className,
      )}
    >
      <StatusIcon className="h-3.5 w-3.5" strokeWidth={2} />
      {label}
    </span>
  )
}

function DetailSkeleton() {
  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <div className="h-5 w-32 animate-pulse rounded bg-gray-200 dark:bg-gray-700 mb-6" />
      <div className="rounded-2xl border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800/80 overflow-hidden">
        <div className="h-1.5 animate-pulse bg-gray-200 dark:bg-gray-700" />
        <div className="px-6 pt-6 pb-8 space-y-6">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-3">
              <div className="h-12 w-12 animate-pulse rounded-xl bg-gray-200 dark:bg-gray-700" />
              <div className="space-y-2">
                <div className="h-5 w-48 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
                <div className="h-3 w-24 animate-pulse rounded bg-gray-100 dark:bg-gray-700" />
              </div>
            </div>
            <div className="h-6 w-20 animate-pulse rounded-full bg-gray-200 dark:bg-gray-700" />
          </div>
          <div className="border-t border-gray-100 dark:border-gray-700" />
          <div className="h-8 w-32 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
          <div className="border-t border-gray-100 dark:border-gray-700" />
          <div className="space-y-2">
            <div className="h-3 w-full animate-pulse rounded bg-gray-100 dark:bg-gray-700" />
            <div className="h-3 w-3/4 animate-pulse rounded bg-gray-100 dark:bg-gray-700" />
            <div className="h-3 w-1/2 animate-pulse rounded bg-gray-100 dark:bg-gray-700" />
          </div>
        </div>
      </div>
    </div>
  )
}
