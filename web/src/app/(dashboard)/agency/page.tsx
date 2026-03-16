"use client"

import Link from "next/link"
import {
  Briefcase,
  MessageSquare,
  Euro,
  Users,
  ArrowRight,
  Clock,
} from "lucide-react"
import { useAuth } from "@/shared/hooks/use-auth"

export default function AgencyDashboardPage() {
  const { user } = useAuth()

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-foreground">
          Bonjour, {user?.display_name ?? "Agence"}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Vue d&apos;ensemble de votre activite
        </p>
      </div>

      {/* Stats cards */}
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
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
              <MessageSquare className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Messages non lus</p>
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
              <p className="text-sm font-medium text-muted-foreground">Revenus du mois</p>
              <p className="mt-0.5 text-2xl font-bold text-foreground">0 &euro;</p>
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-border bg-card p-6">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <Users className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Membres de l&apos;equipe</p>
              <p className="mt-0.5 text-2xl font-bold text-foreground">0</p>
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
          href="/dashboard/agency/missions"
          className="group flex items-center justify-between rounded-xl border border-border bg-card p-6 transition hover:border-primary/30 hover:shadow-sm"
        >
          <div>
            <h3 className="font-semibold text-foreground">Mes Missions</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Consultez et gerez vos missions actives
            </p>
          </div>
          <ArrowRight className="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1 group-hover:text-primary" />
        </Link>
        <Link
          href="/dashboard/agency/team"
          className="group flex items-center justify-between rounded-xl border border-border bg-card p-6 transition hover:border-primary/30 hover:shadow-sm"
        >
          <div>
            <h3 className="font-semibold text-foreground">Mon Equipe</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Gerez les membres de votre agence
            </p>
          </div>
          <ArrowRight className="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1 group-hover:text-primary" />
        </Link>
        <Link
          href="/dashboard/agency/messaging"
          className="group flex items-center justify-between rounded-xl border border-border bg-card p-6 transition hover:border-primary/30 hover:shadow-sm"
        >
          <div>
            <h3 className="font-semibold text-foreground">Messagerie</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Echangez avec vos clients et prestataires
            </p>
          </div>
          <ArrowRight className="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1 group-hover:text-primary" />
        </Link>
      </div>
    </div>
  )
}
