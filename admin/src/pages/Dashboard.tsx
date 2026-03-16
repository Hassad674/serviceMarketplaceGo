import { Users, Briefcase, Building2, ClipboardList, TrendingUp } from "lucide-react"
import { cn } from "@/lib/utils.ts"
import { formatCurrency } from "@/lib/utils.ts"

const stats = [
  { label: "Total utilisateurs", value: "1 248", icon: Users, trend: "+12%" },
  { label: "Prestataires actifs", value: "342", icon: Briefcase, trend: "+8%" },
  { label: "Entreprises actives", value: "89", icon: Building2, trend: "+5%" },
  { label: "Missions en cours", value: "156", icon: ClipboardList, trend: "+18%" },
  { label: "CA mensuel", value: formatCurrency(87450), icon: TrendingUp, trend: "+22%" },
]

function StatCard({
  label,
  value,
  icon: Icon,
  trend,
}: (typeof stats)[number]) {
  return (
    <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-muted-foreground">{label}</p>
          <p className="mt-1 text-2xl font-bold text-card-foreground">{value}</p>
        </div>
        <div className="rounded-lg bg-primary/10 p-3">
          <Icon className="h-5 w-5 text-primary" />
        </div>
      </div>
      <p className={cn("mt-3 text-sm font-medium", "text-success")}>
        {trend} ce mois
      </p>
    </div>
  )
}

export default function DashboardPage() {
  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold text-foreground">Tableau de bord</h1>

      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        {stats.map((stat) => (
          <StatCard key={stat.label} {...stat} />
        ))}
      </div>

      {/* Chart placeholder */}
      <div className="mt-8 grid grid-cols-1 gap-6 lg:grid-cols-2">
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold text-card-foreground">
            Inscriptions par mois
          </h2>
          <div className="flex h-64 items-center justify-center text-muted-foreground">
            Graphique Recharts (inscriptions)
          </div>
        </div>
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <h2 className="mb-4 text-lg font-semibold text-card-foreground">
            Chiffre d'affaires
          </h2>
          <div className="flex h-64 items-center justify-center text-muted-foreground">
            Graphique Recharts (revenus)
          </div>
        </div>
      </div>
    </div>
  )
}
