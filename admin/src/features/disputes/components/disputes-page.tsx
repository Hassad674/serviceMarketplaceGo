import { useState } from "react"
import { Scale, AlertTriangle, Clock, CheckCircle2, XCircle, ShieldAlert } from "lucide-react"
import { useNavigate } from "react-router-dom"

import { PageHeader } from "@/shared/components/layouts/page-header"
import { Badge } from "@/shared/components/ui/badge"
import { Select } from "@/shared/components/ui/select"
import { TableSkeleton } from "@/shared/components/ui/skeleton"
import { EmptyState } from "@/shared/components/ui/empty-state"
import { formatCurrency, formatRelativeDate } from "@/shared/lib/utils"
import { useDisputes, useDisputeCount } from "../hooks/use-disputes"
import type { AdminDispute, DisputeStatus } from "../types"

const STATUS_OPTIONS = [
  { value: "", label: "Tous les statuts" },
  { value: "open", label: "Ouverts" },
  { value: "negotiation", label: "En negociation" },
  { value: "escalated", label: "En mediation" },
  { value: "resolved", label: "Resolus" },
  { value: "cancelled", label: "Annules" },
]

const STATUS_BADGE: Record<DisputeStatus, { variant: "default" | "warning" | "destructive" | "success"; label: string }> = {
  open: { variant: "destructive", label: "Ouvert" },
  negotiation: { variant: "warning", label: "Negociation" },
  escalated: { variant: "destructive", label: "En mediation" },
  resolved: { variant: "success", label: "Resolu" },
  cancelled: { variant: "default", label: "Annule" },
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

export function DisputesPage() {
  const navigate = useNavigate()
  const [statusFilter, setStatusFilter] = useState("")
  const [cursor] = useState("")
  const { data, isLoading, isError } = useDisputes({ status: statusFilter, cursor })
  const { data: counts } = useDisputeCount()

  if (isLoading) return <TableSkeleton rows={6} cols={5} />
  if (isError) return <div className="p-6 text-center text-sm text-destructive">Erreur lors du chargement des litiges</div>

  const disputes = data?.data ?? []

  return (
    <div className="space-y-6">
      <PageHeader
        title="Litiges"
        description={counts ? `${counts.open} ouverts · ${counts.escalated} en mediation · ${counts.total} total` : undefined}
      />

      {/* Filters */}
      <div className="flex items-center gap-3">
        <Select
          options={STATUS_OPTIONS}
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          label=""
        />
      </div>

      {disputes.length === 0 ? (
        <EmptyState icon={Scale} title="Aucun litige" description="Aucun litige ne correspond aux filtres." />
      ) : (
        <div className="space-y-3">
          {disputes.map((d) => (
            <DisputeCard key={d.id} dispute={d} onClick={() => navigate(`/disputes/${d.id}`)} />
          ))}
        </div>
      )}
    </div>
  )
}

function DisputeCard({ dispute, onClick }: { dispute: AdminDispute; onClick: () => void }) {
  const badge = STATUS_BADGE[dispute.status]
  const icon = dispute.status === "escalated" ? ShieldAlert
    : dispute.status === "resolved" ? CheckCircle2
    : dispute.status === "cancelled" ? XCircle
    : dispute.status === "negotiation" ? Clock
    : AlertTriangle

  const Icon = icon

  return (
    <button
      type="button"
      onClick={onClick}
      className="w-full rounded-xl border border-gray-100 bg-white p-4 shadow-sm hover:shadow-md hover:border-rose-200 transition-all text-left"
    >
      <div className="flex items-start gap-3">
        <div className={`mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${
          dispute.status === "escalated" ? "bg-orange-100" : "bg-amber-100"
        }`}>
          <Icon className={`h-4.5 w-4.5 ${
            dispute.status === "escalated" ? "text-orange-600" : "text-amber-600"
          }`} />
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <Badge variant={badge.variant}>{badge.label}</Badge>
            <span className="text-xs text-muted-foreground">
              {formatRelativeDate(dispute.created_at)}
            </span>
            {dispute.ai_summary && (
              <Badge variant="default">IA</Badge>
            )}
          </div>

          <p className="mt-1 text-sm font-medium text-foreground truncate">
            {REASON_LABELS[dispute.reason] ?? dispute.reason}
          </p>

          <p className="mt-0.5 text-xs text-muted-foreground">
            {formatCurrency(dispute.requested_amount / 100)} demande sur {formatCurrency(dispute.proposal_amount / 100)} ·
            {dispute.initiator_role === "client" ? " Initie par le client" : " Initie par le prestataire"}
          </p>
        </div>

        <div className="text-right shrink-0">
          <p className="text-sm font-semibold text-foreground">
            {formatCurrency(dispute.proposal_amount / 100)}
          </p>
          <p className="text-xs text-muted-foreground">
            {dispute.counter_proposals?.length ?? 0} proposition(s)
          </p>
        </div>
      </div>
    </button>
  )
}
