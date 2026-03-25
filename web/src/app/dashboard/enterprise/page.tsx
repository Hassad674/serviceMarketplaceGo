"use client"

import Link from "next/link"
import {
  FolderOpen,
  MessageSquare,
  Wallet,
  Clock,
  ArrowRight,
  PlusCircle,
  Briefcase,
  Search,
  Receipt,
  AlertTriangle,
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

const QUICK_LINKS = [
  { href: "/dashboard/enterprise/projects/new", icon: PlusCircle, label: "Publier un projet" },
  { href: "/dashboard/enterprise/projects", icon: Briefcase, label: "Mes projets" },
  { href: "/dashboard/enterprise/search", icon: Search, label: "Chercher prestataire" },
  { href: "/dashboard/enterprise/billing", icon: Receipt, label: "Facturation" },
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

      <div className="rounded-xl border border-border bg-card shadow-sm">
        <div className="border-b border-border px-6 py-4">
          <h2 className="text-lg font-semibold text-foreground">Activit&eacute; r&eacute;cente</h2>
        </div>
        <div className="flex flex-col items-center justify-center px-6 py-16 text-center">
          <Clock className="h-16 w-16 text-muted-foreground/30" />
          <p className="mt-4 text-lg font-medium text-muted-foreground">
            Aucune activit&eacute; r&eacute;cente
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            Vos projets et messages appara&icirc;tront ici
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

      <div className="rounded-xl border border-red-200 bg-red-50/50 p-6">
        <div className="flex items-start gap-3">
          <AlertTriangle className="mt-0.5 h-5 w-5 text-red-600" />
          <div>
            <h3 className="text-base font-semibold text-red-900">Zone de danger</h3>
            <p className="mt-1 text-sm text-red-700">
              Cette action est irr&eacute;versible. Toutes vos donn&eacute;es seront supprim&eacute;es d&eacute;finitivement.
            </p>
            <button
              type="button"
              className="mt-4 rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-700"
            >
              Supprimer mon compte
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
