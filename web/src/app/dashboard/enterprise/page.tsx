"use client"

import {
  FolderOpen,
  MessageSquare,
  Wallet,
} from "lucide-react"
import { useAuth } from "@/shared/hooks/use-auth"

const STATS = [
  {
    icon: FolderOpen,
    title: "Projets en cours",
    value: "0",
    subtitle: "Projets actifs",
    color: "blue",
  },
  {
    icon: MessageSquare,
    title: "Messages non lus",
    value: "0",
    subtitle: "Conversations",
    color: "violet",
  },
  {
    icon: Wallet,
    title: "Budget total",
    value: "0 \u20AC",
    subtitle: "D\u00E9pens\u00E9 ce mois",
    color: "emerald",
  },
] as const

const COLOR_MAP: Record<string, string> = {
  blue: "bg-blue-50 text-blue-600",
  violet: "bg-violet-50 text-violet-600",
  emerald: "bg-emerald-50 text-emerald-600",
}

export default function EnterpriseDashboardPage() {
  const { user } = useAuth()

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">
          Bonjour, {user?.display_name ?? "Entreprise"}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Trouvez les meilleurs prestataires pour vos projets
        </p>
      </div>

      <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
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
