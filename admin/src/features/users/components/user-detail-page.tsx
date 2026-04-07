import { useState } from "react"
import { useParams, useNavigate } from "react-router-dom"
import { ArrowLeft, Mail, Calendar, Shield, ShieldOff, Ban, UserCheck, Flag } from "lucide-react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { Card, CardContent } from "@/shared/components/ui/card"
import { Button } from "@/shared/components/ui/button"
import { RoleBadge, StatusBadge, Badge } from "@/shared/components/ui/badge"
import { Avatar } from "@/shared/components/ui/avatar"
import { Skeleton } from "@/shared/components/ui/skeleton"
import { Dialog, DialogTitle, DialogDescription, DialogFooter } from "@/shared/components/ui/dialog"
import { Textarea } from "@/shared/components/ui/textarea"
import { Input } from "@/shared/components/ui/input"
import { ReportList } from "@/shared/components/ui/report-list"
import { ResolveReportDialog } from "@/shared/components/ui/resolve-report-dialog"
import type { AdminReport } from "@/shared/types/report"
import { formatDate } from "@/shared/lib/utils"
import { useUser, useSuspendUser, useUnsuspendUser, useBanUser, useUnbanUser } from "../hooks/use-users"
import { useUserReports, useResolveReport } from "@/shared/hooks/use-reports"

export function UserDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data, isLoading, error } = useUser(id!)

  const [showSuspendDialog, setShowSuspendDialog] = useState(false)
  const [showBanDialog, setShowBanDialog] = useState(false)

  const suspendMutation = useSuspendUser(id!)
  const unsuspendMutation = useUnsuspendUser(id!)
  const banMutation = useBanUser(id!)
  const unbanMutation = useUnbanUser(id!)

  if (isLoading) return <UserDetailSkeleton />

  if (error || !data) {
    return (
      <div className="space-y-6">
        <Button variant="ghost" size="sm" onClick={() => navigate(-1)}>
          <ArrowLeft className="h-4 w-4" /> Retour
        </Button>
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Utilisateur introuvable
        </div>
      </div>
    )
  }

  const user = data.data
  const name = user.display_name || `${user.first_name} ${user.last_name}`
  const hasError = suspendMutation.isError || banMutation.isError || unsuspendMutation.isError || unbanMutation.isError

  return (
    <div className="space-y-6">
      <Button variant="ghost" size="sm" onClick={() => navigate(-1)}>
        <ArrowLeft className="h-4 w-4" /> Retour aux utilisateurs
      </Button>

      <PageHeader title={name} />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <ProfileCard user={user} name={name} />
        <DetailsCard user={user} />
      </div>

      {/* Suspension info */}
      {user.status === "suspended" && user.suspension_reason && (
        <SuspensionInfoCard
          reason={user.suspension_reason}
          suspendedAt={user.suspended_at}
          expiresAt={user.suspension_expires_at}
        />
      )}

      {/* Ban info */}
      {user.status === "banned" && user.ban_reason && (
        <BanInfoCard reason={user.ban_reason} bannedAt={user.banned_at} />
      )}

      {/* Actions */}
      <Card>
        <CardContent className="space-y-4 pt-6">
          <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
            Actions
          </h3>
          <div className="flex flex-wrap gap-3">
            {user.status === "active" && (
              <>
                <Button variant="outline" onClick={() => setShowSuspendDialog(true)}>
                  <Shield className="h-4 w-4" /> Suspendre
                </Button>
                <Button variant="destructive" onClick={() => setShowBanDialog(true)}>
                  <Ban className="h-4 w-4" /> Bannir
                </Button>
              </>
            )}
            {user.status === "suspended" && (
              <Button
                variant="outline"
                onClick={() => unsuspendMutation.mutate()}
                disabled={unsuspendMutation.isPending}
              >
                <ShieldOff className="h-4 w-4" />
                {unsuspendMutation.isPending ? "En cours..." : "Lever la suspension"}
              </Button>
            )}
            {user.status === "banned" && (
              <Button
                variant="outline"
                onClick={() => unbanMutation.mutate()}
                disabled={unbanMutation.isPending}
              >
                <UserCheck className="h-4 w-4" />
                {unbanMutation.isPending ? "En cours..." : "Lever le bannissement"}
              </Button>
            )}
          </div>
          {hasError && (
            <p className="text-sm text-destructive">
              Une erreur est survenue. Veuillez r&eacute;essayer.
            </p>
          )}
        </CardContent>
      </Card>

      <UserReportsSection userId={id!} />

      <SuspendDialog
        open={showSuspendDialog}
        onClose={() => setShowSuspendDialog(false)}
        onConfirm={(reason, expiresAt) => {
          suspendMutation.mutate(
            { reason, expires_at: expiresAt || undefined },
            { onSuccess: () => setShowSuspendDialog(false) },
          )
        }}
        isPending={suspendMutation.isPending}
      />

      <BanDialog
        open={showBanDialog}
        onClose={() => setShowBanDialog(false)}
        onConfirm={(reason) => {
          banMutation.mutate(
            { reason },
            { onSuccess: () => setShowBanDialog(false) },
          )
        }}
        isPending={banMutation.isPending}
      />
    </div>
  )
}

/* ─── Sub-components ────────────────────────────────────────────────── */

function ProfileCard({ user, name }: { user: { email: string; role: string; status: string; is_admin: boolean; referrer_enabled: boolean }; name: string }) {
  return (
    <Card className="lg:col-span-1">
      <CardContent className="flex flex-col items-center gap-4 pt-6 text-center">
        <Avatar name={name} size="lg" />
        <div>
          <h2 className="text-lg font-semibold text-foreground">{name}</h2>
          <p className="text-sm text-muted-foreground">{user.email}</p>
        </div>
        <div className="flex flex-wrap justify-center gap-2">
          <RoleBadge role={user.role} />
          <StatusBadge status={user.status} />
          {user.is_admin && <Badge variant="default">Admin</Badge>}
          {user.referrer_enabled && <Badge variant="outline">Apporteur</Badge>}
        </div>
      </CardContent>
    </Card>
  )
}

function DetailsCard({ user }: { user: { email: string; email_verified: boolean; created_at: string; status: string; id: string; updated_at: string; display_name: string } }) {
  return (
    <Card className="lg:col-span-2">
      <CardContent className="space-y-4 pt-6">
        <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
          Informations
        </h3>
        <InfoRow icon={Mail} label="Email" value={user.email}>
          {user.email_verified
            ? <Badge variant="success">V&eacute;rifi&eacute;</Badge>
            : <Badge variant="warning">Non v&eacute;rifi&eacute;</Badge>}
        </InfoRow>
        <InfoRow icon={Calendar} label="Inscrit le" value={formatDate(user.created_at)} />
        <InfoRow icon={Shield} label="Statut" value="">
          <StatusBadge status={user.status} />
        </InfoRow>
      </CardContent>
    </Card>
  )
}

function InfoRow({ icon: Icon, label, value, children }: {
  icon: React.ElementType
  label: string
  value: string
  children?: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between border-b border-border py-3 last:border-0">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Icon className="h-4 w-4" />
        {label}
      </div>
      <div className="flex items-center gap-2">
        {value && <span className="text-sm font-medium text-foreground">{value}</span>}
        {children}
      </div>
    </div>
  )
}

function SuspensionInfoCard({ reason, suspendedAt, expiresAt }: {
  reason: string
  suspendedAt?: string
  expiresAt?: string
}) {
  return (
    <div className="rounded-xl border border-amber-200 bg-amber-50 p-6">
      <h2 className="mb-3 text-sm font-semibold text-amber-800">Suspension active</h2>
      <dl className="space-y-2 text-sm">
        <div>
          <dt className="text-amber-600">Raison</dt>
          <dd className="font-medium text-amber-900">{reason}</dd>
        </div>
        {suspendedAt && (
          <div>
            <dt className="text-amber-600">Suspendu le</dt>
            <dd className="text-amber-900">{formatDate(suspendedAt)}</dd>
          </div>
        )}
        {expiresAt && (
          <div>
            <dt className="text-amber-600">Expire le</dt>
            <dd className="text-amber-900">{formatDate(expiresAt)}</dd>
          </div>
        )}
      </dl>
    </div>
  )
}

function BanInfoCard({ reason, bannedAt }: { reason: string; bannedAt?: string }) {
  return (
    <div className="rounded-xl border border-red-200 bg-red-50 p-6">
      <h2 className="mb-3 text-sm font-semibold text-red-800">Bannissement actif</h2>
      <dl className="space-y-2 text-sm">
        <div>
          <dt className="text-red-600">Raison</dt>
          <dd className="font-medium text-red-900">{reason}</dd>
        </div>
        {bannedAt && (
          <div>
            <dt className="text-red-600">Banni le</dt>
            <dd className="text-red-900">{formatDate(bannedAt)}</dd>
          </div>
        )}
      </dl>
    </div>
  )
}

function SuspendDialog({ open, onClose, onConfirm, isPending }: {
  open: boolean
  onClose: () => void
  onConfirm: (reason: string, expiresAt: string) => void
  isPending: boolean
}) {
  const [reason, setReason] = useState("")
  const [expiresAt, setExpiresAt] = useState("")

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>Suspendre l&apos;utilisateur</DialogTitle>
      <DialogDescription>
        Indiquez la raison et la dur&eacute;e de la suspension.
      </DialogDescription>
      <div className="mt-4 space-y-4">
        <Textarea
          label="Raison"
          required
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Raison de la suspension..."
          rows={3}
        />
        <Input
          type="datetime-local"
          label="Date d'expiration (optionnel)"
          value={expiresAt}
          onChange={(e) => setExpiresAt(e.target.value)}
        />
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>Annuler</Button>
        <Button
          variant="destructive"
          onClick={() => {
            if (!reason.trim()) return
            const iso = expiresAt ? new Date(expiresAt).toISOString() : ""
            onConfirm(reason.trim(), iso)
          }}
          disabled={!reason.trim() || isPending}
        >
          {isPending ? "En cours..." : "Suspendre"}
        </Button>
      </DialogFooter>
    </Dialog>
  )
}

function BanDialog({ open, onClose, onConfirm, isPending }: {
  open: boolean
  onClose: () => void
  onConfirm: (reason: string) => void
  isPending: boolean
}) {
  const [reason, setReason] = useState("")

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>Bannir l&apos;utilisateur</DialogTitle>
      <DialogDescription>
        Cette action est s&eacute;v&egrave;re. L&apos;utilisateur ne pourra plus acc&eacute;der &agrave; la plateforme.
      </DialogDescription>
      <div className="mt-4">
        <Textarea
          label="Raison"
          required
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Raison du bannissement..."
          rows={3}
        />
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>Annuler</Button>
        <Button
          variant="destructive"
          onClick={() => {
            if (!reason.trim()) return
            onConfirm(reason.trim())
          }}
          disabled={!reason.trim() || isPending}
        >
          {isPending ? "En cours..." : "Bannir"}
        </Button>
      </DialogFooter>
    </Dialog>
  )
}

function UserReportsSection({ userId }: { userId: string }) {
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

function ReportGroup({ title, reports, emptyLabel, onResolve, onDismiss, isResolving }: {
  title: string
  reports: AdminReport[]
  emptyLabel: string
  onResolve: (id: string) => void
  onDismiss: (id: string) => void
  isResolving: boolean
}) {
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

function UserDetailSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-8 w-40" />
      <Skeleton className="h-10 w-64" />
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Skeleton className="h-64 rounded-xl" />
        <Skeleton className="h-64 rounded-xl lg:col-span-2" />
      </div>
    </div>
  )
}
