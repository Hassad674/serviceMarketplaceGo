import { useState } from "react"
import { useParams, Link } from "react-router-dom"
import {
  ArrowLeft, Bot, MessageSquare, Scale, Clock,
  CheckCircle2, XCircle, Loader2, FileText,
} from "lucide-react"

import { PageHeader } from "@/shared/components/layouts/page-header"
import { Card, CardHeader, CardTitle, CardContent } from "@/shared/components/ui/card"
import { Badge } from "@/shared/components/ui/badge"
import { Button } from "@/shared/components/ui/button"
import { Textarea } from "@/shared/components/ui/textarea"
import { TableSkeleton } from "@/shared/components/ui/skeleton"
import { formatCurrency, formatDate } from "@/shared/lib/utils"
import { useDispute, useResolveDispute } from "../hooks/use-disputes"
import type { AdminCounterProposal } from "../types"

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
