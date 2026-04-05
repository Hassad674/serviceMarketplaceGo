import { Flag } from "lucide-react"
import { EmptyState } from "./empty-state"
import { ReportCard } from "./report-card"
import type { AdminReport } from "@/shared/types/report"

type ReportListProps = {
  reports: AdminReport[]
  onResolve?: (reportId: string) => void
  onDismiss?: (reportId: string) => void
  isResolving?: boolean
  emptyLabel?: string
}

export function ReportList({
  reports,
  onResolve,
  onDismiss,
  isResolving,
  emptyLabel = "Aucun signalement",
}: ReportListProps) {
  if (reports.length === 0) {
    return (
      <EmptyState
        icon={Flag}
        title={emptyLabel}
        className="py-8"
      />
    )
  }

  return (
    <div className="space-y-3">
      {reports.map((report) => (
        <ReportCard
          key={report.id}
          report={report}
          onResolve={onResolve}
          onDismiss={onDismiss}
          isResolving={isResolving}
        />
      ))}
    </div>
  )
}
