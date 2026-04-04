import { Calendar, CheckCircle, XCircle } from "lucide-react"
import { Badge } from "./badge"
import { Button } from "./button"
import { formatRelativeDate } from "@/shared/lib/utils"
import type { AdminReport } from "@/shared/types/report"

const REASON_LABELS: Record<string, string> = {
  harassment: "Harcelement",
  fraud: "Fraude",
  spam: "Spam",
  inappropriate_content: "Contenu inapproprie",
  fake_profile: "Faux profil",
  unprofessional_behavior: "Comportement non professionnel",
  other: "Autre",
}

const TARGET_LABELS: Record<string, string> = {
  message: "Message",
  user: "Utilisateur",
}

const STATUS_VARIANT: Record<string, "warning" | "success" | "outline"> = {
  pending: "warning",
  reviewed: "warning",
  resolved: "success",
  dismissed: "outline",
}

const STATUS_LABELS: Record<string, string> = {
  pending: "En attente",
  reviewed: "En cours",
  resolved: "Resolu",
  dismissed: "Rejete",
}

type ReportCardProps = {
  report: AdminReport
  onResolve?: (reportId: string) => void
  onDismiss?: (reportId: string) => void
  isResolving?: boolean
}

export function ReportCard({ report, onResolve, onDismiss, isResolving }: ReportCardProps) {
  const isPending = report.status === "pending" || report.status === "reviewed"

  return (
    <div className="rounded-lg border border-border bg-white p-4 space-y-3">
      <div className="flex items-center gap-2 flex-wrap">
        <Badge variant="destructive">{REASON_LABELS[report.reason] || report.reason}</Badge>
        <Badge variant="outline">{TARGET_LABELS[report.target_type] || report.target_type}</Badge>
        <Badge variant={STATUS_VARIANT[report.status] || "outline"}>
          {STATUS_LABELS[report.status] || report.status}
        </Badge>
      </div>

      {report.description && (
        <p className="text-sm text-foreground/80">{report.description}</p>
      )}

      <div className="flex items-center gap-1 text-xs text-muted-foreground">
        <Calendar className="h-3 w-3" />
        {formatRelativeDate(report.created_at)}
      </div>

      {report.status === "resolved" && report.admin_note && (
        <ResolvedInfo note={report.admin_note} resolvedAt={report.resolved_at} />
      )}

      {report.status === "dismissed" && report.admin_note && (
        <DismissedInfo note={report.admin_note} resolvedAt={report.resolved_at} />
      )}

      {isPending && (onResolve || onDismiss) && (
        <div className="flex gap-2 pt-1">
          {onResolve && (
            <Button
              size="sm"
              variant="primary"
              onClick={() => onResolve(report.id)}
              disabled={isResolving}
            >
              <CheckCircle className="h-3.5 w-3.5" /> Resoudre
            </Button>
          )}
          {onDismiss && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => onDismiss(report.id)}
              disabled={isResolving}
            >
              <XCircle className="h-3.5 w-3.5" /> Rejeter
            </Button>
          )}
        </div>
      )}
    </div>
  )
}

function ResolvedInfo({ note, resolvedAt }: { note: string; resolvedAt?: string }) {
  return (
    <div className="rounded-md bg-success/5 border border-success/20 px-3 py-2 text-xs">
      <p className="font-medium text-success">Note admin (resolu)</p>
      <p className="mt-1 text-foreground/70">{note}</p>
      {resolvedAt && (
        <p className="mt-1 text-muted-foreground">{formatRelativeDate(resolvedAt)}</p>
      )}
    </div>
  )
}

function DismissedInfo({ note, resolvedAt }: { note: string; resolvedAt?: string }) {
  return (
    <div className="rounded-md bg-muted px-3 py-2 text-xs">
      <p className="font-medium text-muted-foreground">Note admin (rejete)</p>
      <p className="mt-1 text-foreground/70">{note}</p>
      {resolvedAt && (
        <p className="mt-1 text-muted-foreground">{formatRelativeDate(resolvedAt)}</p>
      )}
    </div>
  )
}
