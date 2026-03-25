"use client"

import Link from "next/link"
import {
  Briefcase,
  MessageSquare,
  TrendingUp,
  Building2,
  ArrowRight,
  Users,
  Receipt,
  UserCircle,
} from "lucide-react"
import { useAuth } from "@/shared/hooks/use-auth"

const STATS = [
  {
    icon: Briefcase,
    title: "Missions en cours",
    value: "0",
    subtitle: "Contrats actifs",
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
    icon: TrendingUp,
    title: "Revenus du mois",
    value: "0 \u20AC",
    subtitle: "Ce mois-ci",
    color: "emerald",
  },
] as const

const QUICK_LINKS = [
  { href: "/dashboard/agency/profile", icon: UserCircle, label: "Mon profil" },
  { href: "/dashboard/agency/missions", icon: Briefcase, label: "Mes missions" },
  { href: "/dashboard/agency/team", icon: Users, label: "Mon \u00E9quipe" },
  { href: "/dashboard/agency/billing", icon: Receipt, label: "Facturation" },
] as const

const COLOR_MAP: Record<string, string> = {
  blue: "bg-blue-50 text-blue-600",
  violet: "bg-violet-50 text-violet-600",
  emerald: "bg-emerald-50 text-emerald-600",
}

export default function AgencyDashboardPage() {
  const { user } = useAuth()

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">
          Bonjour, {user?.display_name ?? "Agence"}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          G&eacute;rez votre agence et vos missions
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

      <div className="rounded-xl border border-border bg-card shadow-sm">
        <div className="border-b border-border px-6 py-4">
          <h2 className="text-lg font-semibold text-foreground">Activit&eacute; r&eacute;cente</h2>
        </div>
        <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
          <Building2 className="h-16 w-16 text-muted-foreground/30" />
          <p className="mt-4 text-lg font-medium text-muted-foreground">
            Aucune activit&eacute; r&eacute;cente
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Vos missions et messages appara&icirc;tront ici
          </p>
        </div>
      </div>

      <div>
        <h2 className="mb-4 text-lg font-semibold text-foreground">Acc&egrave;s rapides</h2>
        <div className="grid gap-4 grid-cols-1 sm:grid-cols-2 lg:grid-cols-4">
          {QUICK_LINKS.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className="group flex items-center justify-between rounded-xl border border-border bg-card p-5 transition-all hover:border-primary/50 hover:shadow-md"
            >
              <div className="flex items-center gap-3">
                <link.icon className="h-5 w-5 text-muted-foreground group-hover:text-primary" />
                <span className="font-medium text-foreground">{link.label}</span>
              </div>
              <ArrowRight className="h-4 w-4 text-muted-foreground transition-transform group-hover:translate-x-1 group-hover:text-primary" />
            </Link>
          ))}
        </div>
      </div>
    </div>
  )
}
