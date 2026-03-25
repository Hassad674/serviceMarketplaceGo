"use client"

import Link from "next/link"
import {
  Handshake,
  Clock,
  CheckCircle,
  TrendingUp,
  ArrowRight,
  PlusCircle,
  FolderOpen,
  UserCircle,
  Search,
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

const QUICK_LINKS = [
  { href: "/dashboard/referrer/create", icon: PlusCircle, label: "Cr\u00E9er un apport d\u2019affaire" },
  { href: "/dashboard/referrer/referrals", icon: FolderOpen, label: "Mes apports" },
  { href: "/dashboard/referrer/profile", icon: UserCircle, label: "Profil Apporteur" },
  { href: "/dashboard/referrer/search", icon: Search, label: "Chercher un freelance" },
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

      <div className="rounded-xl border border-border bg-card shadow-sm">
        <div className="border-b border-border px-6 py-4">
          <h2 className="text-lg font-semibold text-foreground">Activit&eacute; r&eacute;cente</h2>
        </div>
        <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
          <Handshake className="h-16 w-16 text-muted-foreground/30" />
          <p className="mt-4 text-lg font-medium text-muted-foreground">
            Aucune activit&eacute; r&eacute;cente
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Vos mises en relation et missions appara&icirc;tront ici
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
