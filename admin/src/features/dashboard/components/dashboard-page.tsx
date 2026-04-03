import { useNavigate } from "react-router-dom"
import {
  Users,
  Briefcase,
  Building2,
  UserCheck,
  FolderKanban,
  AlertTriangle,
} from "lucide-react"
import { Card } from "@/shared/components/ui/card"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { Skeleton } from "@/shared/components/ui/skeleton"
import { RoleBadge } from "@/shared/components/ui/badge"
import { Avatar } from "@/shared/components/ui/avatar"
import { formatRelativeDate } from "@/shared/lib/utils"
import { useDashboardStats } from "../hooks/use-dashboard"
import type { DashboardStats } from "../api/dashboard-api"

function StatCardSkeleton() {
  return (
    <Card className="p-6">
      <div className="flex items-center justify-between">
        <div className="space-y-2">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-7 w-16" />
        </div>
        <Skeleton className="h-11 w-11 rounded-lg" />
      </div>
    </Card>
  )
}

type StatCardProps = {
  label: string
  value: number
  icon: React.ElementType
  color?: string
}

function StatCard({ label, value, icon: Icon, color = "primary" }: StatCardProps) {
  const bgMap: Record<string, string> = {
    primary: "bg-primary/10",
    blue: "bg-blue-50",
    violet: "bg-violet-50",
    rose: "bg-rose-50",
    emerald: "bg-emerald-50",
  }
  const textMap: Record<string, string> = {
    primary: "text-primary",
    blue: "text-blue-600",
    violet: "text-violet-600",
    rose: "text-rose-600",
    emerald: "text-emerald-600",
  }

  return (
    <Card className="p-6 transition-all duration-200 ease-out hover:-translate-y-0.5 hover:shadow-md">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-muted-foreground">{label}</p>
          <p className="mt-1 font-mono text-2xl font-bold text-card-foreground">
            {value.toLocaleString("fr-FR")}
          </p>
        </div>
        <div className={`rounded-lg p-3 ${bgMap[color] || bgMap.primary}`}>
          <Icon className={`h-5 w-5 ${textMap[color] || textMap.primary}`} />
        </div>
      </div>
    </Card>
  )
}

function StatusDot({ color }: { color: string }) {
  return <span className={`inline-block h-2 w-2 rounded-full ${color}`} />
}

function StatusOverview({ stats }: { stats: DashboardStats }) {
  const items = [
    { label: "Actifs", value: stats.active_users, dotColor: "bg-success" },
    { label: "Suspendus", value: stats.suspended_users, dotColor: "bg-warning" },
    { label: "Bannis", value: stats.banned_users, dotColor: "bg-destructive" },
    { label: "Jobs ouverts", value: stats.open_jobs, dotColor: "bg-blue-500" },
  ]

  return (
    <Card className="p-6">
      <h2 className="mb-4 text-lg font-semibold text-card-foreground">
        Vue d&apos;ensemble
      </h2>
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        {items.map((item) => (
          <div key={item.label} className="space-y-1">
            <div className="flex items-center gap-2">
              <StatusDot color={item.dotColor} />
              <span className="text-sm text-muted-foreground">{item.label}</span>
            </div>
            <p className="font-mono text-xl font-bold text-card-foreground">
              {item.value.toLocaleString("fr-FR")}
            </p>
          </div>
        ))}
      </div>
    </Card>
  )
}

function RecentSignupsTable({ stats }: { stats: DashboardStats }) {
  const navigate = useNavigate()

  return (
    <Card className="p-6">
      <h2 className="mb-4 text-lg font-semibold text-card-foreground">
        Inscriptions recentes
      </h2>
      {stats.recent_signups.length === 0 ? (
        <p className="py-8 text-center text-sm text-muted-foreground">
          Aucune inscription
        </p>
      ) : (
        <div className="divide-y divide-border">
          {stats.recent_signups.map((signup) => (
            <button
              key={signup.id}
              type="button"
              onClick={() => navigate(`/users/${signup.id}`)}
              className="flex w-full items-center gap-3 px-2 py-3 text-left transition-colors duration-200 hover:bg-muted/50"
            >
              <Avatar name={signup.display_name} size="sm" />
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium text-foreground">
                  {signup.display_name}
                </p>
                <p className="truncate text-xs text-muted-foreground">
                  {signup.email}
                </p>
              </div>
              <RoleBadge role={signup.role} />
              <span className="shrink-0 text-xs text-muted-foreground">
                {formatRelativeDate(signup.created_at)}
              </span>
            </button>
          ))}
        </div>
      )}
    </Card>
  )
}

function RecentSignupsSkeleton() {
  return (
    <Card className="p-6">
      <Skeleton className="mb-4 h-6 w-48" />
      <div className="space-y-3">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3 py-2">
            <Skeleton className="h-8 w-8 rounded-full" />
            <div className="flex-1 space-y-1.5">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-3 w-48" />
            </div>
            <Skeleton className="h-5 w-20 rounded-full" />
          </div>
        ))}
      </div>
    </Card>
  )
}

function DashboardError() {
  return (
    <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center">
      <AlertTriangle className="mx-auto mb-2 h-8 w-8 text-destructive" />
      <p className="text-sm font-medium text-destructive">
        Erreur lors du chargement des statistiques
      </p>
      <p className="mt-1 text-xs text-muted-foreground">
        Veuillez rafraichir la page ou reessayer plus tard
      </p>
    </div>
  )
}

export function DashboardPage() {
  const { data: stats, isLoading, isError } = useDashboardStats()

  if (isError) {
    return (
      <div className="space-y-8">
        <PageHeader title="Tableau de bord" />
        <DashboardError />
      </div>
    )
  }

  if (isLoading || !stats) {
    return (
      <div className="space-y-8">
        <PageHeader title="Tableau de bord" />
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
          {Array.from({ length: 5 }).map((_, i) => (
            <StatCardSkeleton key={i} />
          ))}
        </div>
        <Skeleton className="h-32 w-full rounded-xl" />
        <RecentSignupsSkeleton />
      </div>
    )
  }

  return (
    <div className="space-y-8">
      <PageHeader title="Tableau de bord" />

      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        <StatCard
          label="Total utilisateurs"
          value={stats.total_users}
          icon={Users}
        />
        <StatCard
          label="Prestataires"
          value={stats.users_by_role?.agency ?? 0}
          icon={Briefcase}
          color="blue"
        />
        <StatCard
          label="Entreprises"
          value={stats.users_by_role?.enterprise ?? 0}
          icon={Building2}
          color="violet"
        />
        <StatCard
          label="Freelances"
          value={stats.users_by_role?.provider ?? 0}
          icon={UserCheck}
          color="rose"
        />
        <StatCard
          label="Projets actifs"
          value={stats.active_proposals}
          icon={FolderKanban}
          color="emerald"
        />
      </div>

      <StatusOverview stats={stats} />

      <RecentSignupsTable stats={stats} />
    </div>
  )
}
