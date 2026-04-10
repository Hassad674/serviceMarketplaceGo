import { useState } from "react"
import { useParams, Link } from "react-router-dom"
import {
  ArrowLeft, Bot, MessageSquare, Scale, Clock,
  CheckCircle2, XCircle, Loader2, FileText, Ban,
} from "lucide-react"

import { PageHeader } from "@/shared/components/layouts/page-header"
import { Card, CardHeader, CardTitle, CardContent } from "@/shared/components/ui/card"
import { Badge } from "@/shared/components/ui/badge"
import { Button } from "@/shared/components/ui/button"
import { Textarea } from "@/shared/components/ui/textarea"
import { TableSkeleton } from "@/shared/components/ui/skeleton"
import { formatCurrency, formatDate } from "@/shared/lib/utils"
import { useDispute, useResolveDispute, useForceEscalateDispute } from "../hooks/use-disputes"
import type { AdminCounterProposal, AdminDispute } from "../types"
import { AIBudgetPanel } from "./ai-budget-panel"
import { AIChatPanel } from "./ai-chat-panel"

export function DisputeDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { data: dispute, isLoading, isError } = useDispute(id!)

  if (isLoading) return <TableSkeleton rows={8} cols={2} />
  if (isError || !dispute) return <div className="p-6 text-destructive">Erreur lors du chargement</div>

  return (
    <div className="space-y-6">
      <Link to="/disputes" className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ArrowLeft className="h-4 w-4" /> Retour aux litiges
      </Link>

      <PageHeader
        title={`Litige — ${REASON_LABELS[dispute.reason] ?? dispute.reason}`}
        description={`Ouvert le ${formatDate(dispute.created_at)} · ${dispute.initiator_role === "client" ? "Client" : "Prestataire"}`}
      />

      {/* Heads-up banner: a cancellation request is currently pending between
          the parties. The admin should consider waiting for the other party's
          decision before rendering a final ruling, since the dispute may
          self-resolve in the next minutes. */}
      {dispute.cancellation_requested_by && dispute.status === "escalated" && (
        <CancellationPendingBanner dispute={dispute} />
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main content — 2 cols */}
        <div className="lg:col-span-2 space-y-6">
          {/* AI Summary */}
          {dispute.ai_summary && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Bot className="h-5 w-5 text-violet-500" />
                  Resume IA
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="prose prose-sm max-w-none text-muted-foreground whitespace-pre-wrap">
                  {dispute.ai_summary}
                </div>
              </CardContent>
            </Card>
          )}

          {/* AI chat panel — admin can ask follow-up questions about the
              dispute. Chat history is persisted server-side and loaded
              with the dispute detail, so refreshes and multi-admin
              handoffs preserve the full conversation. */}
          {dispute.ai_budget && (
            <AIChatPanel
              disputeId={dispute.id}
              history={dispute.ai_chat_history ?? []}
              budgetExceeded={
                dispute.ai_budget.chat_used_tokens >=
                dispute.ai_budget.chat_max_tokens
              }
            />
          )}

          {/* Initial description */}
          <Card>
            <CardHeader>
              <CardTitle>Description du litige</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground whitespace-pre-wrap">{dispute.description}</p>
              {dispute.evidence.length > 0 && (
                <div className="mt-4 space-y-2">
                  <p className="text-xs font-medium text-muted-foreground">Pieces jointes</p>
                  {dispute.evidence.map((e) => (
                    <a
                      key={e.id}
                      href={e.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-2 rounded-lg border p-2 text-sm hover:bg-muted/50 transition-colors"
                    >
                      <FileText className="h-4 w-4 text-muted-foreground" />
                      {e.filename}
                      <span className="text-xs text-muted-foreground ml-auto">{formatSize(e.size)}</span>
                    </a>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Counter-proposals timeline */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <MessageSquare className="h-5 w-5 text-blue-500" />
                Propositions ({dispute.counter_proposals.length})
              </CardTitle>
            </CardHeader>
            <CardContent>
              {dispute.counter_proposals.length === 0 ? (
                <p className="text-sm text-muted-foreground">Aucune proposition echangee</p>
              ) : (
                <div className="space-y-3">
                  {dispute.counter_proposals.map((cp) => (
                    <CounterProposalItem key={cp.id} cp={cp} dispute={dispute} />
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Sidebar — 1 col */}
        <div className="space-y-6">
          {/* Info card */}
          <Card>
            <CardHeader><CardTitle>Informations</CardTitle></CardHeader>
            <CardContent className="space-y-3 text-sm">
              <InfoRow label="Montant mission" value={formatCurrency(dispute.proposal_amount / 100)} />
              <InfoRow label="Montant demande" value={formatCurrency(dispute.requested_amount / 100)} />
              <InfoRow label="Statut" value={<StatusBadge status={dispute.status} />} />
              <InfoRow label="Initiateur" value={dispute.initiator_role === "client" ? "Client" : "Prestataire"} />
              {dispute.escalated_at && <InfoRow label="Escalade" value={formatDate(dispute.escalated_at)} />}
              {dispute.resolved_at && <InfoRow label="Resolu" value={formatDate(dispute.resolved_at)} />}
            </CardContent>
          </Card>

          {/* Link to conversation */}
          <Card>
            <CardContent className="pt-6">
              <Link
                to={`/conversations/${dispute.conversation_id}`}
                className="flex items-center gap-2 text-sm text-primary hover:underline"
              >
                <MessageSquare className="h-4 w-4" />
                Voir la conversation
              </Link>
            </CardContent>
          </Card>

          {/* AI budget panel — live token usage + cost in EUR + bonus button */}
          {dispute.ai_budget && (
            <AIBudgetPanel disputeId={dispute.id} budget={dispute.ai_budget} />
          )}

          {/* DEV ONLY — instant escalation to test the admin resolution flow
              without waiting 7 days for the inactivity window. The button is
              hidden in production builds, and the backend endpoint also
              returns 404 in production for defense-in-depth. */}
          {import.meta.env.MODE !== "production" &&
            (dispute.status === "open" || dispute.status === "negotiation") && (
              <ForceEscalateButton disputeId={dispute.id} />
            )}

          {/* Resolution form (only for escalated) */}
          {dispute.status === "escalated" && (
            <ResolutionForm disputeId={dispute.id} proposalAmount={dispute.proposal_amount} />
          )}

          {/* Resolution result (if resolved) */}
          {dispute.status === "resolved" && dispute.resolution_amount_client != null && (
            <Card>
              <CardHeader><CardTitle className="flex items-center gap-2"><CheckCircle2 className="h-5 w-5 text-success" /> Decision</CardTitle></CardHeader>
              <CardContent className="space-y-2 text-sm">
                <InfoRow label="Client" value={formatCurrency(dispute.resolution_amount_client / 100)} />
                <InfoRow label="Prestataire" value={formatCurrency((dispute.resolution_amount_provider ?? 0) / 100)} />
                {dispute.resolution_note && (
                  <p className="mt-2 text-muted-foreground italic">{dispute.resolution_note}</p>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function ResolutionForm({ disputeId, proposalAmount }: { disputeId: string; proposalAmount: number }) {
  const mutation = useResolveDispute(disputeId)
  const [clientAmount, setClientAmount] = useState(0)
  const [note, setNote] = useState("")

  const providerAmount = proposalAmount - clientAmount

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    mutation.mutate({ amount_client: clientAmount, amount_provider: providerAmount, note })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Scale className="h-5 w-5 text-rose-500" />
          Rendre la decision
        </CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="mb-1 block text-xs font-medium text-muted-foreground">Repartition</label>
            <input
              type="range"
              min={0}
              max={proposalAmount}
              step={100}
              value={clientAmount}
              onChange={(e) => setClientAmount(Number(e.target.value))}
              className="w-full accent-rose-500"
            />
            <div className="mt-1 flex justify-between text-xs text-muted-foreground">
              <span>Client: {formatCurrency(clientAmount / 100)}</span>
              <span>Prestataire: {formatCurrency(providerAmount / 100)}</span>
            </div>
          </div>

          <Textarea
            label="Message aux parties"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            rows={3}
            placeholder="Justification de la decision..."
          />

          <Button
            type="submit"
            variant="primary"
            size="md"
            disabled={mutation.isPending}
            className="w-full"
          >
            {mutation.isPending ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
            Valider la decision
          </Button>

          {mutation.isSuccess && (
            <p className="text-sm text-success">Decision rendue avec succes.</p>
          )}
          {mutation.isError && (
            <p className="text-sm text-destructive">Erreur lors de la resolution.</p>
          )}
        </form>
      </CardContent>
    </Card>
  )
}

function CounterProposalItem({ cp, dispute }: { cp: AdminCounterProposal; dispute: { client_id: string; proposal_amount: number } }) {
  const isClient = cp.proposer_id === dispute.client_id
  const StatusIcon = cp.status === "accepted" ? CheckCircle2 : cp.status === "rejected" ? XCircle : Clock

  return (
    <div className="flex items-start gap-3 rounded-lg border p-3">
      <StatusIcon className={`mt-0.5 h-4 w-4 ${
        cp.status === "accepted" ? "text-success" : cp.status === "rejected" ? "text-destructive" : "text-muted-foreground"
      }`} />
      <div className="flex-1">
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium">{isClient ? "Client" : "Prestataire"}</span>
          <Badge variant={cp.status === "accepted" ? "success" : cp.status === "rejected" ? "destructive" : "default"}>
            {cp.status}
          </Badge>
          <span className="text-xs text-muted-foreground">{formatDate(cp.created_at)}</span>
        </div>
        <p className="mt-1 text-sm">
          Client: {formatCurrency(cp.amount_client / 100)} · Prestataire: {formatCurrency(cp.amount_provider / 100)}
        </p>
        {cp.message && <p className="mt-1 text-xs text-muted-foreground italic">{cp.message}</p>}
      </div>
    </div>
  )
}

function CancellationPendingBanner({ dispute }: { dispute: AdminDispute }) {
  // Identify which role asked to cancel so the admin understands who's
  // pushing for the dispute to drop.
  const requesterRole =
    dispute.cancellation_requested_by === dispute.client_id
      ? "client"
      : dispute.cancellation_requested_by === dispute.provider_id
        ? "prestataire"
        : "une partie"

  return (
    <div
      role="alert"
      className="flex items-start gap-3 rounded-xl border border-orange-300 bg-orange-50 p-4 dark:border-orange-500/30 dark:bg-orange-500/10"
    >
      <Ban className="mt-0.5 h-5 w-5 shrink-0 text-orange-600" aria-hidden />
      <div className="flex-1">
        <p className="text-sm font-semibold text-orange-900 dark:text-orange-200">
          Demande d&apos;annulation en attente
        </p>
        <p className="mt-1 text-xs text-orange-800/90 dark:text-orange-200/80">
          Le {requesterRole} demande l&apos;annulation du litige et attend la
          reponse de l&apos;autre partie. Le dossier peut se cloturer
          spontanement avant votre decision — il est recommande d&apos;attendre
          quelques minutes avant de trancher pour eviter toute action croisee.
        </p>
      </div>
    </div>
  )
}

function ForceEscalateButton({ disputeId }: { disputeId: string }) {
  const mutation = useForceEscalateDispute(disputeId)

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-orange-700">
          <Scale className="h-5 w-5" />
          Outil de developpement
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-xs text-muted-foreground">
          Force l&apos;escalation immediate de ce litige a la mediation, sans
          attendre les 7 jours d&apos;inactivite. Visible uniquement en
          environnement de developpement.
        </p>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={mutation.isPending}
          onClick={() => mutation.mutate()}
        >
          {mutation.isPending ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
          Forcer l&apos;escalation (DEV)
        </Button>
        {mutation.isError && (
          <p className="text-xs text-destructive">Echec de l&apos;escalation forcee.</p>
        )}
      </CardContent>
    </Card>
  )
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex justify-between">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium text-foreground">{value}</span>
    </div>
  )
}

function StatusBadge({ status }: { status: string }) {
  const config: Record<string, { variant: "default" | "warning" | "destructive" | "success"; label: string }> = {
    open: { variant: "destructive", label: "Ouvert" },
    negotiation: { variant: "warning", label: "Negociation" },
    escalated: { variant: "destructive", label: "En mediation" },
    resolved: { variant: "success", label: "Resolu" },
    cancelled: { variant: "default", label: "Annule" },
  }
  const c = config[status] ?? { variant: "default" as const, label: status }
  return <Badge variant={c.variant}>{c.label}</Badge>
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

const REASON_LABELS: Record<string, string> = {
  work_not_conforming: "Travail non conforme",
  non_delivery: "Non-livraison",
  insufficient_quality: "Qualite insuffisante",
  client_ghosting: "Client injoignable",
  scope_creep: "Hors du scope",
  refusal_to_validate: "Refus de valider",
  harassment: "Harcelement",
  other: "Autre",
}
