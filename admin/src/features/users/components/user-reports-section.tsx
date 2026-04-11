import { useState } from "react"
import { Flag } from "lucide-react"
import { Card, CardContent } from "@/shared/components/ui/card"
import { Skeleton } from "@/shared/components/ui/skeleton"
import { ReportList } from "@/shared/components/ui/report-list"
import { ResolveReportDialog } from "@/shared/components/ui/resolve-report-dialog"
import type { AdminReport } from "@/shared/types/report"
import { useUserReports, useResolveReport } from "@/shared/hooks/use-reports"

// Reports card on the user detail page. Groups the reports targeting
// this user's profile, the reports targeting their messages, and the
// reports they filed against others. Each group has a resolve /
// dismiss affordance that opens the shared ResolveReportDialog.

type UserReportsSectionProps = {
  userId: string
}

export function UserReportsSection({ userId }: UserReportsSectionProps) {
  const { data, isLoading } = useUserReports(userId)
  const resolveMutation = useResolveReport()
  const [resolveTarget, setResolveTarget] = useState<{ id: string; defaultStatus: "resolved" | "dismissed" } | null>(null)

  const against = data?.reports_against ?? []
  const filed = data?.reports_filed ?? []

  const profileReports = against.filter((r) => r.target_type === "user")
  const messageReports = against.filter((r) => r.target_type === "message")

  if (isLoading) {
    return (
      <Card>
        <CardContent className="pt-6">
          <Skeleton className="h-24" />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardContent className="space-y-6 pt-6">
        <div className="flex items-center gap-2">
          <Flag className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
            Signalements
          </h3>
        </div>

        <div className="space-y-4">
          <ReportGroup
            title={`Profil signale (${profileReports.length})`}
            reports={profileReports}
            emptyLabel="Aucun signalement de profil"
            onResolve={(id) => setResolveTarget({ id, defaultStatus: "resolved" })}
            onDismiss={(id) => setResolveTarget({ id, defaultStatus: "dismissed" })}
            isResolving={resolveMutation.isPending}
          />

          <ReportGroup
            title={`Messages signales (${messageReports.length})`}
            reports={messageReports}
            emptyLabel="Aucun message signale"
            onResolve={(id) => setResolveTarget({ id, defaultStatus: "resolved" })}
            onDismiss={(id) => setResolveTarget({ id, defaultStatus: "dismissed" })}
            isResolving={resolveMutation.isPending}
          />

          <div>
            <h4 className="mb-2 text-sm font-medium text-foreground">
              Signalements soumis ({filed.length})
            </h4>
            <ReportList
              reports={filed}
              emptyLabel="Aucun signalement soumis"
            />
          </div>
        </div>

        {resolveTarget && (
          <ResolveReportDialog
            open={!!resolveTarget}
            onClose={() => setResolveTarget(null)}
            onConfirm={(status, adminNote) => {
              resolveMutation.mutate(
                { reportId: resolveTarget.id, status, adminNote },
                { onSuccess: () => setResolveTarget(null) },
              )
            }}
            isPending={resolveMutation.isPending}
            defaultStatus={resolveTarget.defaultStatus}
          />
        )}
      </CardContent>
    </Card>
  )
}

type ReportGroupProps = {
  title: string
  reports: AdminReport[]
  emptyLabel: string
  onResolve: (id: string) => void
  onDismiss: (id: string) => void
  isResolving: boolean
}

function ReportGroup({ title, reports, emptyLabel, onResolve, onDismiss, isResolving }: ReportGroupProps) {
  return (
    <div>
      <h4 className="mb-2 text-sm font-medium text-foreground">{title}</h4>
      <ReportList
        reports={reports}
        onResolve={onResolve}
        onDismiss={onDismiss}
        isResolving={isResolving}
        emptyLabel={emptyLabel}
      />
    </div>
  )
}
