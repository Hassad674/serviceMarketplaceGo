"use client"

import Link from "next/link"
import {
  FolderKanban,
  FileText,
  Briefcase,
  Euro,
  ArrowRight,
  Clock,
} from "lucide-react"
import { useAuth } from "@/shared/hooks/use-auth"

export default function EnterpriseDashboardPage() {
  const { user } = useAuth()

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-foreground">
          Bonjour, {user?.display_name ?? "Entreprise"}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Vue d&apos;ensemble de vos projets
        </p>
      </div>

      {/* Stats cards */}
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-xl border border-border bg-card p-6">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <FolderKanban className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Projets publies</p>
              <p className="mt-0.5 text-2xl font-bold text-foreground">0</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-border bg-card p-6">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <FileText className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Candidatures recues</p>
              <p className="mt-0.5 text-2xl font-bold text-foreground">0</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-border bg-card p-6">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Briefcase className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Missions en cours</p>
              <p className="mt-0.5 text-2xl font-bold text-foreground">0</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-border bg-card p-6">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Euro className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Budget depense</p>
              <p className="mt-0.5 text-2xl font-bold text-foreground">0 &euro;</p>
            </div>
          </div>
        </div>
      </div>

      {/* Recent activity */}
      <div className="rounded-xl border border-border bg-card">
        <div className="flex items-center justify-between border-b border-border px-6 py-4">
          <h2 className="text-lg font-semibold text-foreground">Activite recente</h2>
        </div>
        <div className="flex flex-col items-center justify-center px-6 py-12 text-center">
          <Clock className="h-10 w-10 text-muted-foreground/40" />
          <p className="mt-3 text-sm font-medium text-muted-foreground">
            Aucune activite recente
          </p>
          <p className="mt-1 text-xs text-muted-foreground/70">
            Vos dernieres actions apparaitront ici
          </p>
        </div>
      </div>

      {/* Quick links */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Link
          href="/dashboard/enterprise/projects"
          className="group flex items-center justify-between rounded-xl border border-border bg-card p-6 transition hover:border-primary/30 hover:shadow-sm"
        >
          <div>
            <h3 className="font-semibold text-foreground">Mes Projets</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Publiez et gerez vos projets
            </p>
          </div>
          <ArrowRight className="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1 group-hover:text-primary" />
        </Link>
        <Link
          href="/dashboard/enterprise/search"
          className="group flex items-center justify-between rounded-xl border border-border bg-card p-6 transition hover:border-primary/30 hover:shadow-sm"
        >
          <div>
            <h3 className="font-semibold text-foreground">Rechercher</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Trouvez les meilleurs prestataires
            </p>
          </div>
          <ArrowRight className="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1 group-hover:text-primary" />
        </Link>
        <Link
          href="/dashboard/enterprise/messaging"
          className="group flex items-center justify-between rounded-xl border border-border bg-card p-6 transition hover:border-primary/30 hover:shadow-sm"
        >
          <div>
            <h3 className="font-semibold text-foreground">Messagerie</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Echangez avec vos prestataires
            </p>
          </div>
          <ArrowRight className="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1 group-hover:text-primary" />
        </Link>
      </div>
    </div>
  )
}
