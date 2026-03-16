"use client"

import Link from "next/link"
import { useState } from "react"
import {
  Briefcase,
  MessageSquare,
  Euro,
  Handshake,
  Users,
  ArrowRight,
  Clock,
} from "lucide-react"
import { useAuth } from "@/shared/hooks/use-auth"
import { cn } from "@/shared/lib/utils"

export default function ProviderDashboardPage() {
  const { user } = useAuth()
  const [referrerMode, setReferrerMode] = useState(user?.referrer_enabled ?? false)

  return (
    <div className="space-y-8">
      {/* Header with toggle */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">
            Bonjour, {user?.display_name ?? "Prestataire"}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Vue d&apos;ensemble de votre activite
          </p>
        </div>

        {/* Referrer toggle */}
        <div
          className={cn(
            "flex items-center gap-3 rounded-xl border px-4 py-3 transition-colors",
            referrerMode
              ? "border-primary/30 bg-primary/5"
              : "border-border bg-card",
          )}
        >
          <Handshake
            className={cn(
              "h-5 w-5",
              referrerMode ? "text-primary" : "text-muted-foreground",
            )}
          />
          <span className="text-sm font-medium text-foreground">
            Mode apporteur d&apos;affaires
          </span>
          <button
            type="button"
            role="switch"
            aria-checked={referrerMode}
            aria-label="Activer le mode apporteur d'affaires"
            onClick={() => setReferrerMode((prev) => !prev)}
            className={cn(
              "relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors",
              referrerMode ? "bg-primary" : "bg-muted",
            )}
          >
            <span
              className={cn(
                "pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform",
                referrerMode ? "translate-x-5" : "translate-x-0",
              )}
            />
          </button>
        </div>
      </div>

      {/* Referrer badge */}
      {referrerMode && (
        <div className="flex items-center gap-2 rounded-lg bg-primary/10 px-4 py-2.5">
          <Handshake className="h-4 w-4 text-primary" />
          <span className="text-sm font-medium text-primary">
            Mode Apporteur d&apos;affaires actif
          </span>
          <span className="text-sm text-primary/70">
            — Vous pouvez recommander des prestataires et recevoir des commissions
          </span>
        </div>
      )}

      {/* Stats cards */}
      <div className={cn("grid gap-6 sm:grid-cols-2", referrerMode ? "lg:grid-cols-5" : "lg:grid-cols-3")}>
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
        {referrerMode && (
          <>
            <div className="rounded-xl border border-border bg-card p-6">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                  <Euro className="h-5 w-5 text-primary" />
                </div>
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Commissions</p>
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
                  <p className="text-sm font-medium text-muted-foreground">Mises en relation</p>
                  <p className="mt-0.5 text-2xl font-bold text-foreground">0</p>
                </div>
              </div>
            </div>
          </>
        )}
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
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Link
          href="/dashboard/provider/missions"
          className="group flex items-center justify-between rounded-xl border border-border bg-card p-6 transition hover:border-primary/30 hover:shadow-sm"
        >
          <div>
            <h3 className="font-semibold text-foreground">Mes Missions</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Consultez vos missions en cours
            </p>
          </div>
          <ArrowRight className="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1 group-hover:text-primary" />
        </Link>
        {referrerMode && (
          <Link
            href="/dashboard/provider/referral"
            className="group flex items-center justify-between rounded-xl border border-primary/20 bg-primary/5 p-6 transition hover:border-primary/40 hover:shadow-sm"
          >
            <div>
              <h3 className="font-semibold text-foreground">
                Apport d&apos;affaires
              </h3>
              <p className="mt-1 text-sm text-muted-foreground">
                Gerez vos recommandations et commissions
              </p>
            </div>
            <ArrowRight className="h-5 w-5 text-primary transition-transform group-hover:translate-x-1" />
          </Link>
        )}
        <Link
          href="/dashboard/provider/messaging"
          className="group flex items-center justify-between rounded-xl border border-border bg-card p-6 transition hover:border-primary/30 hover:shadow-sm"
        >
          <div>
            <h3 className="font-semibold text-foreground">Messagerie</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              Echangez avec vos clients
            </p>
          </div>
          <ArrowRight className="h-5 w-5 text-muted-foreground transition-transform group-hover:translate-x-1 group-hover:text-primary" />
        </Link>
      </div>
    </div>
  )
}
