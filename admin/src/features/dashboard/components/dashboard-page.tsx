import { Users, Briefcase, Building2, ClipboardList, TrendingUp } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { formatCurrency } from "@/shared/lib/utils"
import { Card } from "@/shared/components/ui/card"
import { PageHeader } from "@/shared/components/layouts/page-header"

const stats = [
  { label: "Total utilisateurs", value: "1 248", icon: Users, trend: "+12%" },
  { label: "Prestataires actifs", value: "342", icon: Briefcase, trend: "+8%" },
  { label: "Entreprises actives", value: "89", icon: Building2, trend: "+5%" },
  { label: "Missions en cours", value: "156", icon: ClipboardList, trend: "+18%" },
  { label: "CA mensuel", value: formatCurrency(87450), icon: TrendingUp, trend: "+22%" },
]

function StatCard({ label, value, icon: Icon, trend }: (typeof stats)[number]) {
  return (
    <Card className="p-6">
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
    </Card>
  )
}

export function DashboardPage() {
  return (
    <div className="space-y-8">
      <PageHeader title="Tableau de bord" />

      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        {stats.map((stat) => (
          <StatCard key={stat.label} {...stat} />
        ))}
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card className="p-6">
          <h2 className="mb-4 text-lg font-semibold text-card-foreground">
            Inscriptions par mois
          </h2>
          <div className="flex h-64 items-center justify-center text-muted-foreground">
            Graphique Recharts (inscriptions)
          </div>
        </Card>
        <Card className="p-6">
          <h2 className="mb-4 text-lg font-semibold text-card-foreground">
            Chiffre d&apos;affaires
          </h2>
          <div className="flex h-64 items-center justify-center text-muted-foreground">
            Graphique Recharts (revenus)
          </div>
        </Card>
      </div>
    </div>
  )
}
