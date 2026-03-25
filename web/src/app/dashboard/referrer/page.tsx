"use client"

import {
  Handshake,
  Clock,
  CheckCircle,
  TrendingUp,
} from "lucide-react"
import { useAuth } from "@/shared/hooks/use-auth"

const STATS = [
  {
    icon: Handshake,
    title: "Mises en relation",
    value: "0",
    subtitle: "En attente de r\u00E9ponse",
    color: "teal",
  },
  {
    icon: Clock,
    title: "Missions en cours",
    value: "0",
    subtitle: "Contrats actifs",
    color: "amber",
  },
  {
    icon: CheckCircle,
    title: "Missions termin\u00E9es",
    value: "0",
    subtitle: "Total historique",
    color: "emerald",
  },
  {
    icon: TrendingUp,
    title: "Commissions",
    value: "0 \u20AC",
    subtitle: "Total gagn\u00E9",
    color: "rose",
  },
] as const

const COLOR_MAP: Record<string, string> = {
  teal: "bg-teal-50 text-teal-600",
  amber: "bg-amber-50 text-amber-600",
  emerald: "bg-emerald-50 text-emerald-600",
  rose: "bg-rose-50 text-rose-600",
}

export default function ReferrerDashboardPage() {
  const { user } = useAuth()

  const displayName = user?.first_name || user?.display_name || "Apporteur"

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">
          Bonjour, {displayName}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          G&eacute;rez vos apports d&apos;affaires et vos commissions
        </p>
      </div>

      <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
        {STATS.map((stat) => (
          <div
            key={stat.title}
            className="rounded-xl border border-border bg-card p-6 shadow-sm"
          >
            <div className="flex items-center gap-4">
              <div
                className={`flex h-12 w-12 items-center justify-center rounded-full ${COLOR_MAP[stat.color]}`}
              >
                <stat.icon className="h-5 w-5" />
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">{stat.title}</p>
                <p className="text-3xl font-bold text-foreground">{stat.value}</p>
                <p className="text-xs text-muted-foreground">{stat.subtitle}</p>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
