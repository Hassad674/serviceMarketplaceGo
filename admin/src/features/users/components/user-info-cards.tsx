import { Mail, Calendar, Shield } from "lucide-react"
import { Card, CardContent } from "@/shared/components/ui/card"
import { Avatar } from "@/shared/components/ui/avatar"
import { RoleBadge, StatusBadge, Badge } from "@/shared/components/ui/badge"
import { formatDate } from "@/shared/lib/utils"
import type { AdminUser } from "../types"

// Two side-by-side cards that open the detail page: the profile
// "identity" block on the left and the factual "informations" block on
// the right. Kept in one file because they always render together in a
// `lg:grid-cols-3` container.

type UserProfileCardProps = {
  user: AdminUser
  name: string
}

export function UserProfileCard({ user, name }: UserProfileCardProps) {
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
          {user.account_type === "operator" && <Badge variant="outline">Opérateur</Badge>}
        </div>
      </CardContent>
    </Card>
  )
}

type UserDetailsCardProps = {
  user: AdminUser
}

export function UserDetailsCard({ user }: UserDetailsCardProps) {
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

type InfoRowProps = {
  icon: React.ElementType
  label: string
  value: string
  children?: React.ReactNode
}

function InfoRow({ icon: Icon, label, value, children }: InfoRowProps) {
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
