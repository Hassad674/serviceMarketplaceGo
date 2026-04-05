import { useState } from "react"
import { useParams, useNavigate, Link } from "react-router-dom"
import { ArrowLeft, Trash2, FileText, Flag } from "lucide-react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { Card, CardContent } from "@/shared/components/ui/card"
import { Button } from "@/shared/components/ui/button"
import { Badge } from "@/shared/components/ui/badge"
import { Avatar } from "@/shared/components/ui/avatar"
import { Skeleton } from "@/shared/components/ui/skeleton"
import { DataTable } from "@/shared/components/data-table/data-table"
import { DataTablePagination } from "@/shared/components/data-table/data-table-pagination"
import { EmptyState } from "@/shared/components/ui/empty-state"
import { TableSkeleton } from "@/shared/components/ui/skeleton"
import { Dialog, DialogTitle, DialogDescription, DialogFooter } from "@/shared/components/ui/dialog"
import { ReportList } from "@/shared/components/ui/report-list"
import { ResolveReportDialog } from "@/shared/components/ui/resolve-report-dialog"
import { formatDate, formatCurrency } from "@/shared/lib/utils"
import { useAdminJob, useDeleteJob, useAdminJobApplications, useDeleteJobApplication } from "../hooks/use-jobs"
import { useJobReports, useResolveReport } from "@/shared/hooks/use-reports"
import { applicationsColumns } from "./applications-columns"
import type { ApplicationFilters, AdminJobApplication } from "../types"

export function JobDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data, isLoading, error } = useAdminJob(id!)
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const deleteMutation = useDeleteJob()

  if (isLoading) return <JobDetailSkeleton />

  if (error || !data) {
    return (
      <div className="space-y-6">
        <Button variant="ghost" size="sm" onClick={() => navigate("/jobs")}>
          <ArrowLeft className="h-4 w-4" /> Retour
        </Button>
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Offre introuvable
        </div>
      </div>
    )
  }

  const job = data.data

  return (
    <div className="space-y-6">
      <Button variant="ghost" size="sm" onClick={() => navigate("/jobs")}>
        <ArrowLeft className="h-4 w-4" /> Retour aux offres
      </Button>

      <PageHeader
        title={job.title}
        actions={
          <Button variant="destructive" size="sm" onClick={() => setShowDeleteDialog(true)}>
            <Trash2 className="h-4 w-4" /> Supprimer
          </Button>
        }
      />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <JobInfoCard job={job} />
        <JobAuthorCard author={job.author} />
      </div>

      <JobApplicationsSection jobId={id!} />

      <JobReportsSection jobId={id!} />

      <Dialog open={showDeleteDialog} onClose={() => setShowDeleteDialog(false)}>
        <DialogTitle>Supprimer l&apos;offre</DialogTitle>
        <DialogDescription>
          Etes-vous sur de vouloir supprimer cette offre ? Cette action est irreversible.
        </DialogDescription>
        <DialogFooter>
          <Button variant="outline" onClick={() => setShowDeleteDialog(false)}>Annuler</Button>
          <Button
            variant="destructive"
            disabled={deleteMutation.isPending}
            onClick={() => {
              deleteMutation.mutate(id!, {
                onSuccess: () => navigate("/jobs"),
              })
            }}
          >
            <Trash2 className="h-4 w-4" />
            {deleteMutation.isPending ? "Suppression..." : "Supprimer"}
          </Button>
        </DialogFooter>
      </Dialog>
    </div>
  )
}

/* --- Sub-components --- */

function JobInfoCard({ job }: { job: { status: string; budget_type: string; min_budget: number; max_budget: number; applicant_type: string; skills: string[]; description: string; description_type: string; created_at: string; closed_at?: string; application_count: number } }) {
  const statusLabel = job.status === "open" ? "Ouvert" : "Ferme"
  const statusVariant = job.status === "open" ? "success" : "outline"

  return (
    <Card className="lg:col-span-2">
      <CardContent className="space-y-4 pt-6">
        <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
          Details de l&apos;offre
        </h3>

        <InfoRow label="Statut">
          <Badge variant={statusVariant as "success" | "outline"}>{statusLabel}</Badge>
        </InfoRow>
        <InfoRow label="Budget">
          <span className="text-sm font-medium text-foreground">
            {formatCurrency(job.min_budget)} - {formatCurrency(job.max_budget)}
          </span>
        </InfoRow>
        <InfoRow label="Type de budget">
          <span className="text-sm text-foreground">
            {job.budget_type === "one_shot" ? "Ponctuel" : "Long terme"}
          </span>
        </InfoRow>
        <InfoRow label="Type de candidat">
          <span className="text-sm text-foreground">{applicantLabel(job.applicant_type)}</span>
        </InfoRow>
        <InfoRow label="Candidatures">
          <span className="text-sm font-medium text-foreground">{job.application_count}</span>
        </InfoRow>
        <InfoRow label="Cree le">
          <span className="text-sm text-foreground">{formatDate(job.created_at)}</span>
        </InfoRow>
        {job.closed_at && (
          <InfoRow label="Ferme le">
            <span className="text-sm text-foreground">{formatDate(job.closed_at)}</span>
          </InfoRow>
        )}

        {job.skills.length > 0 && (
          <div className="space-y-2 border-t border-border pt-4">
            <span className="text-sm text-muted-foreground">Competences</span>
            <div className="flex flex-wrap gap-2">
              {job.skills.map((skill) => (
                <Badge key={skill} variant="outline">{skill}</Badge>
              ))}
            </div>
          </div>
        )}

        {job.description && (
          <div className="space-y-2 border-t border-border pt-4">
            <span className="text-sm text-muted-foreground">Description</span>
            <p className="whitespace-pre-wrap text-sm text-foreground">{job.description}</p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function JobAuthorCard({ author }: { author: { id: string; display_name: string; email: string; role: string } }) {
  return (
    <Card className="lg:col-span-1">
      <CardContent className="flex flex-col items-center gap-4 pt-6 text-center">
        <Avatar name={author.display_name} size="lg" />
        <div>
          <h2 className="text-lg font-semibold text-foreground">{author.display_name}</h2>
          <p className="text-sm text-muted-foreground">{author.email}</p>
        </div>
        <Link
          to={`/users/${author.id}`}
          className="text-sm text-primary hover:underline"
        >
          Voir le profil
        </Link>
      </CardContent>
    </Card>
  )
}

function InfoRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between border-b border-border py-3 last:border-0">
      <span className="text-sm text-muted-foreground">{label}</span>
      {children}
    </div>
  )
}

function applicantLabel(applicantType: string): string {
  switch (applicantType) {
    case "freelancers":
      return "Freelances uniquement"
    case "agencies":
      return "Prestataires uniquement"
    default:
      return "Tous"
  }
}

function JobApplicationsSection({ jobId }: { jobId: string }) {
  const [filters, setFilters] = useState<ApplicationFilters>({
    job_id: jobId,
    search: "",
    sort: "",
    filter: "",
    page: 1,
  })
  const [deleteTarget, setDeleteTarget] = useState<AdminJobApplication | null>(null)

  const { data, isLoading, error } = useAdminJobApplications(filters)
  const deleteMutation = useDeleteJobApplication()

  const applications = data?.data ?? []
  const total = data?.total ?? 0
  const totalPages = data?.total_pages ?? 0

  return (
    <Card>
      <CardContent className="space-y-4 pt-6">
        <h3 className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
          <FileText className="h-4 w-4" />
          Candidatures ({total})
        </h3>

        {isLoading && <TableSkeleton rows={5} cols={4} />}

        {error && (
          <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-4 text-center text-sm text-destructive">
            Erreur lors du chargement des candidatures
          </div>
        )}

        {!isLoading && !error && applications.length === 0 && (
          <EmptyState
            icon={FileText}
            title="Aucune candidature"
            description="Cette offre n'a pas encore recu de candidature."
          />
        )}

        {!isLoading && !error && applications.length > 0 && (
          <>
            <DataTable columns={applicationsColumns} data={applications} />
            <DataTablePagination
              currentPage={filters.page}
              totalPages={totalPages}
              totalCount={total}
              onPageChange={(p) => setFilters((f) => ({ ...f, page: p }))}
            />
          </>
        )}

        <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
          <DialogTitle>Supprimer la candidature</DialogTitle>
          <DialogDescription>
            Cette action est irreversible.
          </DialogDescription>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>Annuler</Button>
            <Button
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={() => {
                if (deleteTarget) {
                  deleteMutation.mutate(deleteTarget.id, {
                    onSuccess: () => setDeleteTarget(null),
                  })
                }
              }}
            >
              {deleteMutation.isPending ? "Suppression..." : "Supprimer"}
            </Button>
          </DialogFooter>
        </Dialog>
      </CardContent>
    </Card>
  )
}

function JobReportsSection({ jobId }: { jobId: string }) {
  const { data, isLoading } = useJobReports(jobId)
  const resolveMutation = useResolveReport()
  const [resolveTarget, setResolveTarget] = useState<{
    id: string
    defaultStatus: "resolved" | "dismissed"
  } | null>(null)

  const reports = data?.data ?? []

  if (isLoading) {
    return (
      <Card>
        <CardContent className="space-y-3 pt-6">
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-20 w-full" />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardContent className="space-y-4 pt-6">
        <div className="flex items-center gap-2">
          <h3 className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
            <Flag className="h-4 w-4" />
            Signalements
          </h3>
          {reports.length > 0 && (
            <Badge variant="destructive">{reports.length}</Badge>
          )}
        </div>
        <ReportList
          reports={reports}
          onResolve={(reportId) =>
            setResolveTarget({ id: reportId, defaultStatus: "resolved" })
          }
          onDismiss={(reportId) =>
            setResolveTarget({ id: reportId, defaultStatus: "dismissed" })
          }
          isResolving={resolveMutation.isPending}
        />
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

function JobDetailSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-8 w-40" />
      <Skeleton className="h-10 w-64" />
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Skeleton className="h-80 rounded-xl lg:col-span-2" />
        <Skeleton className="h-48 rounded-xl" />
      </div>
      <Skeleton className="h-48 rounded-xl" />
    </div>
  )
}
