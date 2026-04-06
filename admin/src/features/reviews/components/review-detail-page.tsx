import { useState } from "react"
import { useParams, useNavigate, Link } from "react-router-dom"
import { ArrowLeft, Trash2, Star, Flag } from "lucide-react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { Card, CardContent } from "@/shared/components/ui/card"
import { Button } from "@/shared/components/ui/button"
import { Badge } from "@/shared/components/ui/badge"
import { RoleBadge } from "@/shared/components/ui/badge"
import { Avatar } from "@/shared/components/ui/avatar"
import { Skeleton } from "@/shared/components/ui/skeleton"
import { Dialog, DialogTitle, DialogDescription, DialogFooter } from "@/shared/components/ui/dialog"
import { ReportList } from "@/shared/components/ui/report-list"
import { ResolveReportDialog } from "@/shared/components/ui/resolve-report-dialog"
import { formatDate } from "@/shared/lib/utils"
import { useAdminReview, useDeleteReview } from "../hooks/use-reviews"
import { useReviewReports, useResolveReport } from "@/shared/hooks/use-reports"

export function ReviewDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data, isLoading, error } = useAdminReview(id!)
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const deleteMutation = useDeleteReview()

  if (isLoading) return <ReviewDetailSkeleton />

  if (error || !data) {
    return (
      <div className="space-y-6">
        <Button variant="ghost" size="sm" onClick={() => navigate("/reviews")}>
          <ArrowLeft className="h-4 w-4" /> Retour
        </Button>
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Avis introuvable
        </div>
      </div>
    )
  }

  const review = data.data

  return (
    <div className="space-y-6">
      <Button variant="ghost" size="sm" onClick={() => navigate("/reviews")}>
        <ArrowLeft className="h-4 w-4" /> Retour aux avis
      </Button>

      <PageHeader
        title="Detail de l'avis"
        actions={
          <Button variant="destructive" size="sm" onClick={() => setShowDeleteDialog(true)}>
            <Trash2 className="h-4 w-4" /> Supprimer
          </Button>
        }
      />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <ReviewContentCard review={review} />
        <ReviewUsersCard reviewer={review.reviewer} reviewed={review.reviewed} />
      </div>

      <ReviewReportsSection reviewId={id!} />

      <Dialog open={showDeleteDialog} onClose={() => setShowDeleteDialog(false)}>
        <DialogTitle>Supprimer l&apos;avis</DialogTitle>
        <DialogDescription>
          Etes-vous sur de vouloir supprimer cet avis ? Cette action est irreversible.
        </DialogDescription>
        <DialogFooter>
          <Button variant="outline" onClick={() => setShowDeleteDialog(false)}>Annuler</Button>
          <Button
            variant="destructive"
            disabled={deleteMutation.isPending}
            onClick={() => {
              deleteMutation.mutate(id!, {
                onSuccess: () => navigate("/reviews"),
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

function StarRating({ rating, label }: { rating: number; label: string }) {
  return (
    <div className="flex items-center justify-between border-b border-border py-3 last:border-0">
      <span className="text-sm text-muted-foreground">{label}</span>
      <div className="flex items-center gap-1">
        {Array.from({ length: 5 }, (_, i) => (
          <Star
            key={i}
            className={`h-4 w-4 ${
              i < rating
                ? "fill-amber-400 text-amber-400"
                : "fill-none text-gray-300"
            }`}
          />
        ))}
        <span className="ml-1 text-sm font-medium text-foreground">{rating}/5</span>
      </div>
    </div>
  )
}

type ReviewContentProps = {
  review: {
    global_rating: number
    timeliness?: number
    communication?: number
    quality?: number
    comment: string
    video_url?: string
    created_at: string
    proposal_id: string
  }
}

function ReviewContentCard({ review }: ReviewContentProps) {
  return (
    <Card className="lg:col-span-2">
      <CardContent className="space-y-4 pt-6">
        <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
          Evaluation
        </h3>

        <StarRating rating={review.global_rating} label="Note globale" />
        {review.timeliness != null && (
          <StarRating rating={review.timeliness} label="Ponctualite" />
        )}
        {review.communication != null && (
          <StarRating rating={review.communication} label="Communication" />
        )}
        {review.quality != null && (
          <StarRating rating={review.quality} label="Qualite" />
        )}

        <InfoRow label="Date">
          <span className="text-sm text-foreground">{formatDate(review.created_at)}</span>
        </InfoRow>

        {review.video_url && (
          <div className="space-y-2 border-t border-border pt-4">
            <span className="text-sm text-muted-foreground">Video</span>
            <video
              src={review.video_url}
              controls
              className="w-full max-w-lg rounded-lg border border-border"
            >
              Votre navigateur ne supporte pas la lecture video.
            </video>
          </div>
        )}

        {review.comment && (
          <div className="space-y-2 border-t border-border pt-4">
            <span className="text-sm text-muted-foreground">Commentaire</span>
            <p className="whitespace-pre-wrap text-sm text-foreground">{review.comment}</p>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

type ReviewUsersCardProps = {
  reviewer: { id: string; display_name: string; email: string; role: string }
  reviewed: { id: string; display_name: string; email: string; role: string }
}

function ReviewUsersCard({ reviewer, reviewed }: ReviewUsersCardProps) {
  return (
    <Card className="lg:col-span-1">
      <CardContent className="space-y-6 pt-6">
        <UserSection label="Auteur" user={reviewer} />
        <div className="border-t border-border" />
        <UserSection label="Destinataire" user={reviewed} />
      </CardContent>
    </Card>
  )
}

function UserSection({ label, user }: {
  label: string
  user: { id: string; display_name: string; email: string; role: string }
}) {
  return (
    <div className="space-y-3">
      <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{label}</h4>
      <div className="flex flex-col items-center gap-2 text-center">
        <Avatar name={user.display_name} size="lg" />
        <div>
          <p className="text-sm font-semibold text-foreground">{user.display_name}</p>
          <p className="text-xs text-muted-foreground">{user.email}</p>
        </div>
        <RoleBadge role={user.role} />
        <Link
          to={`/users/${user.id}`}
          className="text-xs text-primary hover:underline"
        >
          Voir le profil
        </Link>
      </div>
    </div>
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

function ReviewReportsSection({ reviewId }: { reviewId: string }) {
  const { data, isLoading } = useReviewReports(reviewId)
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

function ReviewDetailSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-8 w-40" />
      <Skeleton className="h-10 w-64" />
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Skeleton className="h-80 rounded-xl lg:col-span-2" />
        <Skeleton className="h-60 rounded-xl" />
      </div>
      <Skeleton className="h-48 rounded-xl" />
    </div>
  )
}
