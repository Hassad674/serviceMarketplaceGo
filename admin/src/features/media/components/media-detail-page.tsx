import { useState } from "react"
import { useParams, useNavigate, Link } from "react-router-dom"
import { ArrowLeft, Check, X, Trash2, ExternalLink } from "lucide-react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/components/ui/card"
import { Button } from "@/shared/components/ui/button"
import { Badge } from "@/shared/components/ui/badge"
import { Dialog, DialogTitle, DialogDescription, DialogFooter } from "@/shared/components/ui/dialog"
import { Skeleton } from "@/shared/components/ui/skeleton"
import { MediaPreview } from "./media-preview"
import { ScoreBadge } from "./media-columns"
import { useAdminMediaDetail, useApproveMedia, useRejectMedia, useDeleteMedia } from "../hooks/use-media"
import { formatDate } from "@/shared/lib/utils"
import type { ModerationLabel } from "../types"

const statusVariants = {
  pending: "warning",
  approved: "success",
  flagged: "destructive",
  rejected: "outline",
} as const

const statusLabels: Record<string, string> = {
  pending: "En attente",
  approved: "Approuvé",
  flagged: "Signalé",
  rejected: "Rejeté",
}

const contextLabels: Record<string, string> = {
  profile_photo: "Photo de profil",
  profile_video: "Vidéo de profil",
  message_attachment: "Pièce jointe",
  review_video: "Vidéo d'avis",
  job_video: "Vidéo d'offre",
  referrer_video: "Vidéo parrain",
  identity_document: "Pièce d'identité",
}

export function MediaDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data, isLoading, error } = useAdminMediaDetail(id || "")
  const approveMutation = useApproveMedia(id || "")
  const rejectMutation = useRejectMedia(id || "")
  const deleteMutation = useDeleteMedia(id || "")

  const [confirmAction, setConfirmAction] = useState<"approve" | "reject" | "delete" | null>(null)

  function handleConfirm() {
    if (!confirmAction) return
    const action = {
      approve: approveMutation,
      reject: rejectMutation,
      delete: deleteMutation,
    }[confirmAction]

    action.mutate(undefined, {
      onSuccess: () => {
        setConfirmAction(null)
        if (confirmAction === "delete") navigate("/media")
      },
    })
  }

  if (isLoading) return <MediaDetailSkeleton />

  if (error || !data) {
    return (
      <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
        Erreur lors du chargement du m{"é"}dia
      </div>
    )
  }

  const media = data.data

  return (
    <div className="space-y-6">
      <PageHeader
        title="Détail du média"
        actions={
          <Button variant="ghost" size="sm" onClick={() => navigate("/media")}>
            <ArrowLeft className="h-4 w-4" />
            Retour
          </Button>
        }
      />

      <div className="grid gap-6 lg:grid-cols-2">
        <PreviewCard media={media} />
        <div className="space-y-6">
          <UploaderCard media={media} />
          <FileInfoCard media={media} />
        </div>
      </div>

      <ModerationCard media={media} onAction={setConfirmAction} />

      <ConfirmDialog
        action={confirmAction}
        isPending={approveMutation.isPending || rejectMutation.isPending || deleteMutation.isPending}
        onConfirm={handleConfirm}
        onClose={() => setConfirmAction(null)}
      />
    </div>
  )
}

type MediaProp = {
  media: {
    id: string
    file_url: string
    file_type: string
    file_name: string
    file_size: number
    context: string
    moderation_status: string
    moderation_score: number
    moderation_labels: ModerationLabel[]
    uploader_id: string
    uploader_display_name: string
    uploader_email: string
    uploader_role: string
    created_at: string
    reviewed_at?: string
  }
}

function PreviewCard({ media }: MediaProp) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Aper{"ç"}u</CardTitle>
      </CardHeader>
      <CardContent className="flex items-center justify-center">
        <MediaPreview
          fileUrl={media.file_url}
          fileType={media.file_type}
          fileName={media.file_name}
          size="lg"
        />
      </CardContent>
    </Card>
  )
}

function UploaderCard({ media }: MediaProp) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Utilisateur</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Nom</span>
          <Link
            to={`/users/${media.uploader_id}`}
            className="text-sm font-medium text-primary hover:underline inline-flex items-center gap-1"
          >
            {media.uploader_display_name}
            <ExternalLink className="h-3 w-3" />
          </Link>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Email</span>
          <span className="text-sm">{media.uploader_email}</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">R{"ô"}le</span>
          <span className="text-sm capitalize">{media.uploader_role}</span>
        </div>
      </CardContent>
    </Card>
  )
}

function FileInfoCard({ media }: MediaProp) {
  const sizeStr = formatFileSize(media.file_size)

  return (
    <Card>
      <CardHeader>
        <CardTitle>Informations</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        <InfoRow label="Nom" value={media.file_name} />
        <InfoRow label="Type" value={media.file_type} />
        <InfoRow label="Taille" value={sizeStr} />
        <InfoRow label="Contexte" value={contextLabels[media.context] || media.context} />
        <InfoRow label="Date" value={formatDate(media.created_at)} />
      </CardContent>
    </Card>
  )
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className="max-w-[200px] truncate text-sm">{value}</span>
    </div>
  )
}

type ModerationCardProps = MediaProp & {
  onAction: (action: "approve" | "reject" | "delete") => void
}

function ModerationCard({ media, onAction }: ModerationCardProps) {
  const status = media.moderation_status
  const variant = statusVariants[status as keyof typeof statusVariants] || ("outline" as const)

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle>Mod{"é"}ration</CardTitle>
          <Badge variant={variant}>
            {statusLabels[status] || status}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div>
          <p className="mb-1 text-sm text-muted-foreground">Score de risque</p>
          <div className="mt-1">
            {media.moderation_score === 0 ? (
              <p className="text-lg font-mono font-semibold text-muted-foreground">-</p>
            ) : (
              <ScoreBadge score={media.moderation_score} />
            )}
          </div>
        </div>

        {media.moderation_labels.length > 0 && (
          <div>
            <p className="mb-2 text-sm text-muted-foreground">Labels d{"é"}tect{"é"}s</p>
            <div className="space-y-2">
              {media.moderation_labels.map((label, idx) => (
                <LabelBar key={idx} label={label} />
              ))}
            </div>
          </div>
        )}

        {media.reviewed_at && (
          <div>
            <p className="text-sm text-muted-foreground">
              R{"é"}vis{"é"} le {formatDate(media.reviewed_at)}
            </p>
          </div>
        )}

        <div className="flex gap-3 pt-2">
          <Button variant="primary" size="sm" onClick={() => onAction("approve")}>
            <Check className="h-4 w-4" />
            Approuver
          </Button>
          <Button variant="outline" size="sm" onClick={() => onAction("reject")}>
            <X className="h-4 w-4" />
            Rejeter
          </Button>
          <Button variant="destructive" size="sm" onClick={() => onAction("delete")}>
            <Trash2 className="h-4 w-4" />
            Supprimer
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function LabelBar({ label }: { label: ModerationLabel }) {
  const widthPercent = Math.min(label.confidence, 100)
  return (
    <div>
      <div className="flex items-center justify-between text-xs">
        <span className="font-medium">{label.name}</span>
        <span className="text-muted-foreground">{label.confidence.toFixed(1)}%</span>
      </div>
      <div className="mt-1 h-2 w-full rounded-full bg-muted">
        <div
          className="h-2 rounded-full bg-destructive/70 transition-all duration-300"
          style={{ width: `${widthPercent}%` }}
        />
      </div>
    </div>
  )
}

type ConfirmDialogProps = {
  action: "approve" | "reject" | "delete" | null
  isPending: boolean
  onConfirm: () => void
  onClose: () => void
}

const dialogConfig = {
  approve: {
    title: "Approuver ce média ?",
    description: "Ce média sera marqué comme approuvé et visible.",
    confirmLabel: "Approuver",
    variant: "primary" as const,
  },
  reject: {
    title: "Rejeter ce média ?",
    description: "Ce média sera marqué comme rejeté.",
    confirmLabel: "Rejeter",
    variant: "outline" as const,
  },
  delete: {
    title: "Supprimer ce média ?",
    description: "Cette action est irréversible. Le fichier sera définitivement supprimé.",
    confirmLabel: "Supprimer",
    variant: "destructive" as const,
  },
}

function ConfirmDialog({ action, isPending, onConfirm, onClose }: ConfirmDialogProps) {
  if (!action) return null
  const config = dialogConfig[action]

  return (
    <Dialog open={!!action} onClose={onClose}>
      <DialogTitle>{config.title}</DialogTitle>
      <DialogDescription>{config.description}</DialogDescription>
      <DialogFooter>
        <Button variant="ghost" size="sm" onClick={onClose} disabled={isPending}>
          Annuler
        </Button>
        <Button variant={config.variant} size="sm" onClick={onConfirm} disabled={isPending}>
          {isPending ? "..." : config.confirmLabel}
        </Button>
      </DialogFooter>
    </Dialog>
  )
}

function MediaDetailSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-8 w-48" />
      <div className="grid gap-6 lg:grid-cols-2">
        <Skeleton className="h-64" />
        <div className="space-y-6">
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
        </div>
      </div>
      <Skeleton className="h-48" />
    </div>
  )
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} o`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} Ko`
  return `${(bytes / (1024 * 1024)).toFixed(1)} Mo`
}
