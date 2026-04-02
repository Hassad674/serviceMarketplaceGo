import { useState } from "react"
import { useParams, useNavigate } from "react-router-dom"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { adminApi } from "@/lib/api-client.ts"
import { formatDate } from "@/lib/utils.ts"
import { ArrowLeft, Shield, ShieldOff, Ban, UserCheck } from "lucide-react"

interface AdminUser {
  id: string
  email: string
  first_name: string
  last_name: string
  display_name: string
  role: string
  referrer_enabled: boolean
  is_admin: boolean
  status: string
  suspended_at?: string
  suspension_reason?: string
  suspension_expires_at?: string
  banned_at?: string
  ban_reason?: string
  email_verified: boolean
  created_at: string
  updated_at: string
}

interface UserResponse {
  data: AdminUser
}

const roleLabels: Record<string, string> = {
  agency: "Prestataire",
  enterprise: "Entreprise",
  provider: "Freelance",
}

const roleBadgeColors: Record<string, string> = {
  agency: "bg-blue-100 text-blue-700",
  enterprise: "bg-purple-100 text-purple-700",
  provider: "bg-rose-100 text-rose-700",
}

const statusLabels: Record<string, string> = {
  active: "Actif",
  suspended: "Suspendu",
  banned: "Banni",
}

const statusBadgeColors: Record<string, string> = {
  active: "bg-green-100 text-green-700",
  suspended: "bg-amber-100 text-amber-700",
  banned: "bg-red-100 text-red-700",
}

export default function UserDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const [showSuspendDialog, setShowSuspendDialog] = useState(false)
  const [showBanDialog, setShowBanDialog] = useState(false)

  const { data, isLoading, isError } = useQuery<UserResponse>({
    queryKey: ["admin-user", id],
    queryFn: () => adminApi<UserResponse>(`/api/v1/admin/users/${id}`),
    enabled: !!id,
  })

  const invalidateUser = () => {
    queryClient.invalidateQueries({ queryKey: ["admin-user", id] })
    queryClient.invalidateQueries({ queryKey: ["admin-users"] })
  }

  const suspendMutation = useMutation({
    mutationFn: (body: { reason: string; expires_at?: string }) =>
      adminApi(`/api/v1/admin/users/${id}/suspend`, { method: "POST", body }),
    onSuccess: () => {
      invalidateUser()
      setShowSuspendDialog(false)
    },
  })

  const unsuspendMutation = useMutation({
    mutationFn: () =>
      adminApi(`/api/v1/admin/users/${id}/unsuspend`, { method: "POST", body: {} }),
    onSuccess: invalidateUser,
  })

  const banMutation = useMutation({
    mutationFn: (body: { reason: string }) =>
      adminApi(`/api/v1/admin/users/${id}/ban`, { method: "POST", body }),
    onSuccess: () => {
      invalidateUser()
      setShowBanDialog(false)
    },
  })

  const unbanMutation = useMutation({
    mutationFn: () =>
      adminApi(`/api/v1/admin/users/${id}/unban`, { method: "POST", body: {} }),
    onSuccess: invalidateUser,
  })

  if (isLoading) return <DetailSkeleton />

  if (isError || !data) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-muted-foreground">
        <p className="text-sm">Utilisateur introuvable.</p>
        <button
          onClick={() => navigate("/users")}
          className="mt-4 text-sm text-rose-600 hover:underline"
        >
          Retour aux utilisateurs
        </button>
      </div>
    )
  }

  const user = data.data
  const initials = (user.first_name?.[0] ?? "") + (user.last_name?.[0] ?? "")

  return (
    <div className="mx-auto max-w-3xl">
      {/* Back button */}
      <button
        onClick={() => navigate("/users")}
        className="mb-6 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Retour aux utilisateurs
      </button>

      {/* Profile card */}
      <div className="mb-6 rounded-xl border border-border bg-card p-6 shadow-sm">
        <div className="flex items-start gap-4">
          <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full bg-rose-100 text-xl font-bold text-rose-600">
            {initials.toUpperCase() || "?"}
          </div>
          <div className="flex-1">
            <h1 className="text-xl font-bold text-foreground">
              {user.first_name} {user.last_name}
            </h1>
            <p className="text-sm text-muted-foreground">{user.email}</p>
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <span
                className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${roleBadgeColors[user.role] ?? "bg-slate-100 text-slate-700"}`}
              >
                {roleLabels[user.role] ?? user.role}
              </span>
              <span
                className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${statusBadgeColors[user.status] ?? "bg-slate-100 text-slate-700"}`}
              >
                {statusLabels[user.status] ?? user.status}
              </span>
              {user.is_admin && (
                <span className="inline-flex rounded-full bg-slate-800 px-2.5 py-0.5 text-xs font-medium text-white">
                  Admin
                </span>
              )}
              {user.referrer_enabled && (
                <span className="inline-flex rounded-full bg-indigo-100 px-2.5 py-0.5 text-xs font-medium text-indigo-700">
                  Apporteur d'affaire
                </span>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Details */}
      <div className="mb-6 rounded-xl border border-border bg-card p-6 shadow-sm">
        <h2 className="mb-4 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
          Informations
        </h2>
        <dl className="grid grid-cols-2 gap-4 text-sm">
          <DetailRow label="Email" value={user.email} />
          <DetailRow label="Email verifie" value={user.email_verified ? "Oui" : "Non"} />
          <DetailRow label="Nom affiche" value={user.display_name || "-"} />
          <DetailRow label="Date d'inscription" value={formatDate(user.created_at)} />
          <DetailRow label="Derniere mise a jour" value={formatDate(user.updated_at)} />
          <DetailRow label="ID" value={user.id} mono />
        </dl>
      </div>

      {/* Suspension info */}
      {user.status === "suspended" && (
        <div className="mb-6 rounded-xl border border-amber-200 bg-amber-50 p-6">
          <h2 className="mb-3 text-sm font-semibold text-amber-800">Suspension active</h2>
          <dl className="space-y-2 text-sm">
            <div>
              <dt className="text-amber-600">Raison</dt>
              <dd className="font-medium text-amber-900">{user.suspension_reason || "-"}</dd>
            </div>
            {user.suspended_at && (
              <div>
                <dt className="text-amber-600">Suspendu le</dt>
                <dd className="text-amber-900">{formatDate(user.suspended_at)}</dd>
              </div>
            )}
            {user.suspension_expires_at && (
              <div>
                <dt className="text-amber-600">Expire le</dt>
                <dd className="text-amber-900">{formatDate(user.suspension_expires_at)}</dd>
              </div>
            )}
          </dl>
        </div>
      )}

      {/* Ban info */}
      {user.status === "banned" && (
        <div className="mb-6 rounded-xl border border-red-200 bg-red-50 p-6">
          <h2 className="mb-3 text-sm font-semibold text-red-800">Bannissement actif</h2>
          <dl className="space-y-2 text-sm">
            <div>
              <dt className="text-red-600">Raison</dt>
              <dd className="font-medium text-red-900">{user.ban_reason || "-"}</dd>
            </div>
            {user.banned_at && (
              <div>
                <dt className="text-red-600">Banni le</dt>
                <dd className="text-red-900">{formatDate(user.banned_at)}</dd>
              </div>
            )}
          </dl>
        </div>
      )}

      {/* Actions */}
      <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
        <h2 className="mb-4 text-sm font-semibold uppercase tracking-wider text-muted-foreground">
          Actions
        </h2>
        <div className="flex flex-wrap gap-3">
          {user.status === "active" && (
            <>
              <button
                onClick={() => setShowSuspendDialog(true)}
                className="inline-flex items-center gap-2 rounded-lg border border-amber-300 bg-amber-50 px-4 py-2 text-sm font-medium text-amber-700 transition-colors hover:bg-amber-100"
              >
                <Shield className="h-4 w-4" />
                Suspendre
              </button>
              <button
                onClick={() => setShowBanDialog(true)}
                className="inline-flex items-center gap-2 rounded-lg border border-red-300 bg-red-50 px-4 py-2 text-sm font-medium text-red-700 transition-colors hover:bg-red-100"
              >
                <Ban className="h-4 w-4" />
                Bannir
              </button>
            </>
          )}
          {user.status === "suspended" && (
            <button
              onClick={() => unsuspendMutation.mutate()}
              disabled={unsuspendMutation.isPending}
              className="inline-flex items-center gap-2 rounded-lg border border-green-300 bg-green-50 px-4 py-2 text-sm font-medium text-green-700 transition-colors hover:bg-green-100 disabled:opacity-50"
            >
              <ShieldOff className="h-4 w-4" />
              {unsuspendMutation.isPending ? "En cours..." : "Lever la suspension"}
            </button>
          )}
          {user.status === "banned" && (
            <button
              onClick={() => unbanMutation.mutate()}
              disabled={unbanMutation.isPending}
              className="inline-flex items-center gap-2 rounded-lg border border-green-300 bg-green-50 px-4 py-2 text-sm font-medium text-green-700 transition-colors hover:bg-green-100 disabled:opacity-50"
            >
              <UserCheck className="h-4 w-4" />
              {unbanMutation.isPending ? "En cours..." : "Lever le bannissement"}
            </button>
          )}
        </div>

        {/* Mutation errors */}
        {(suspendMutation.isError || banMutation.isError || unsuspendMutation.isError || unbanMutation.isError) && (
          <p className="mt-3 text-sm text-red-600">
            Une erreur est survenue. Veuillez reessayer.
          </p>
        )}
      </div>

      {/* Suspend dialog */}
      {showSuspendDialog && (
        <SuspendDialog
          onClose={() => setShowSuspendDialog(false)}
          onConfirm={(reason, expiresAt) =>
            suspendMutation.mutate({ reason, expires_at: expiresAt || undefined })
          }
          isPending={suspendMutation.isPending}
        />
      )}

      {/* Ban dialog */}
      {showBanDialog && (
        <BanDialog
          onClose={() => setShowBanDialog(false)}
          onConfirm={(reason) => banMutation.mutate({ reason })}
          isPending={banMutation.isPending}
        />
      )}
    </div>
  )
}

function DetailRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className={`font-medium text-foreground ${mono ? "font-mono text-xs" : ""}`}>
        {value}
      </dd>
    </div>
  )
}

function SuspendDialog({
  onClose,
  onConfirm,
  isPending,
}: {
  onClose: () => void
  onConfirm: (reason: string, expiresAt: string) => void
  isPending: boolean
}) {
  const [reason, setReason] = useState("")
  const [expiresAt, setExpiresAt] = useState("")

  return (
    <DialogOverlay onClose={onClose}>
      <h3 className="mb-4 text-lg font-semibold text-foreground">Suspendre l'utilisateur</h3>
      <div className="space-y-4">
        <div>
          <label className="mb-1 block text-sm font-medium text-foreground">
            Raison <span className="text-red-500">*</span>
          </label>
          <textarea
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            rows={3}
            className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm shadow-xs outline-none transition-colors focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10"
            placeholder="Raison de la suspension..."
          />
        </div>
        <div>
          <label className="mb-1 block text-sm font-medium text-foreground">
            Date d'expiration (optionnel)
          </label>
          <input
            type="datetime-local"
            value={expiresAt}
            onChange={(e) => setExpiresAt(e.target.value)}
            className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm shadow-xs outline-none transition-colors focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10"
          />
        </div>
      </div>
      <div className="mt-6 flex justify-end gap-3">
        <button
          onClick={onClose}
          className="rounded-lg border border-border px-4 py-2 text-sm font-medium transition-colors hover:bg-muted"
        >
          Annuler
        </button>
        <button
          onClick={() => {
            if (!reason.trim()) return
            const isoExpiry = expiresAt ? new Date(expiresAt).toISOString() : ""
            onConfirm(reason.trim(), isoExpiry)
          }}
          disabled={!reason.trim() || isPending}
          className="rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-amber-700 disabled:opacity-50"
        >
          {isPending ? "En cours..." : "Suspendre"}
        </button>
      </div>
    </DialogOverlay>
  )
}

function BanDialog({
  onClose,
  onConfirm,
  isPending,
}: {
  onClose: () => void
  onConfirm: (reason: string) => void
  isPending: boolean
}) {
  const [reason, setReason] = useState("")

  return (
    <DialogOverlay onClose={onClose}>
      <h3 className="mb-4 text-lg font-semibold text-foreground">Bannir l'utilisateur</h3>
      <p className="mb-4 text-sm text-muted-foreground">
        Cette action est severe. L'utilisateur ne pourra plus acceder a la plateforme.
      </p>
      <div>
        <label className="mb-1 block text-sm font-medium text-foreground">
          Raison <span className="text-red-500">*</span>
        </label>
        <textarea
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          rows={3}
          className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm shadow-xs outline-none transition-colors focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10"
          placeholder="Raison du bannissement..."
        />
      </div>
      <div className="mt-6 flex justify-end gap-3">
        <button
          onClick={onClose}
          className="rounded-lg border border-border px-4 py-2 text-sm font-medium transition-colors hover:bg-muted"
        >
          Annuler
        </button>
        <button
          onClick={() => {
            if (!reason.trim()) return
            onConfirm(reason.trim())
          }}
          disabled={!reason.trim() || isPending}
          className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-700 disabled:opacity-50"
        >
          {isPending ? "En cours..." : "Bannir"}
        </button>
      </div>
    </DialogOverlay>
  )
}

function DialogOverlay({
  onClose,
  children,
}: {
  onClose: () => void
  children: React.ReactNode
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative w-full max-w-md rounded-2xl border border-border bg-card p-6 shadow-lg">
        {children}
      </div>
    </div>
  )
}

function DetailSkeleton() {
  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div className="h-5 w-48 animate-pulse rounded bg-muted" />
      <div className="rounded-xl border border-border bg-card p-6">
        <div className="flex items-start gap-4">
          <div className="h-16 w-16 animate-pulse rounded-full bg-muted" />
          <div className="flex-1 space-y-3">
            <div className="h-6 w-48 animate-pulse rounded bg-muted" />
            <div className="h-4 w-64 animate-pulse rounded bg-muted" />
            <div className="flex gap-2">
              <div className="h-5 w-20 animate-pulse rounded-full bg-muted" />
              <div className="h-5 w-16 animate-pulse rounded-full bg-muted" />
            </div>
          </div>
        </div>
      </div>
      <div className="rounded-xl border border-border bg-card p-6">
        <div className="h-4 w-32 animate-pulse rounded bg-muted mb-4" />
        <div className="grid grid-cols-2 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="space-y-2">
              <div className="h-3 w-20 animate-pulse rounded bg-muted" />
              <div className="h-4 w-40 animate-pulse rounded bg-muted" />
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
