import { useParams, useNavigate } from "react-router-dom"
import { ArrowLeft, Mail, Calendar, Shield } from "lucide-react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { Card, CardContent } from "@/shared/components/ui/card"
import { Button } from "@/shared/components/ui/button"
import { RoleBadge, StatusBadge, Badge } from "@/shared/components/ui/badge"
import { Avatar } from "@/shared/components/ui/avatar"
import { Skeleton } from "@/shared/components/ui/skeleton"
import { formatDate } from "@/shared/lib/utils"
import { useUser } from "../hooks/use-users"

export function UserDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data, isLoading, error } = useUser(id!)

  if (isLoading) return <UserDetailSkeleton />

  if (error || !data) {
    return (
      <div className="space-y-6">
        <Button variant="ghost" size="sm" onClick={() => navigate("/users")}>
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

  return (
    <div className="space-y-6">
      <Button variant="ghost" size="sm" onClick={() => navigate("/users")}>
        <ArrowLeft className="h-4 w-4" /> Retour aux utilisateurs
      </Button>

      <PageHeader title={name} />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Profile card */}
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

        {/* Details card */}
        <Card className="lg:col-span-2">
          <CardContent className="space-y-4 pt-6">
            <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
              Informations
            </h3>

            <InfoRow icon={Mail} label="Email" value={user.email}>
              {user.email_verified
                ? <Badge variant="success">Vérifié</Badge>
                : <Badge variant="warning">Non vérifié</Badge>}
            </InfoRow>

            <InfoRow icon={Calendar} label="Inscrit le" value={formatDate(user.created_at)} />

            <InfoRow icon={Shield} label="Statut" value="">
              <StatusBadge status={user.status} />
            </InfoRow>

            {user.status === "suspended" && user.suspension_reason && (
              <div className="rounded-lg bg-warning/10 p-3">
                <p className="text-xs font-medium text-warning">Raison de la suspension</p>
                <p className="mt-1 text-sm text-foreground">{user.suspension_reason}</p>
                {user.suspension_expires_at && (
                  <p className="mt-1 text-xs text-muted-foreground">
                    Expire le {formatDate(user.suspension_expires_at)}
                  </p>
                )}
              </div>
            )}

            {user.status === "banned" && user.ban_reason && (
              <div className="rounded-lg bg-destructive/10 p-3">
                <p className="text-xs font-medium text-destructive">Raison du bannissement</p>
                <p className="mt-1 text-sm text-foreground">{user.ban_reason}</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function InfoRow({
  icon: Icon,
  label,
  value,
  children,
}: {
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
