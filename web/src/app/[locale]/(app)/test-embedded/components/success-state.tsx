"use client"

import { ArrowRight, Check, CreditCard, FileText, Settings } from "lucide-react"

import { findCountry } from "../lib/countries"

type SuccessStateProps = {
  accountId: string
  country: string | null
  chargesEnabled: boolean
  payoutsEnabled: boolean
  requirementsCount: number
  onRestart?: () => void
}

export function SuccessState({
  accountId,
  country,
  chargesEnabled,
  payoutsEnabled,
  requirementsCount,
  onRestart,
}: SuccessStateProps) {
  const countryData = country ? findCountry(country) : null
  const fullyActive = chargesEnabled && payoutsEnabled && requirementsCount === 0

  return (
    <div className="flex flex-col gap-8">
      {/* Hero celebration */}
      <div className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-rose-500 via-rose-600 to-purple-600 px-8 py-10 text-white shadow-xl">
        <div
          className="absolute inset-0 opacity-20"
          style={{
            backgroundImage:
              "radial-gradient(circle at 20% 20%, white 1px, transparent 1px), radial-gradient(circle at 70% 60%, white 1px, transparent 1px)",
            backgroundSize: "48px 48px, 32px 32px",
          }}
          aria-hidden
        />
        <div className="relative flex flex-col items-center text-center">
          <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-white/20 backdrop-blur-sm">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-white text-rose-600 shadow-lg animate-scale-in">
              <Check className="h-6 w-6 stroke-[3]" aria-hidden />
            </div>
          </div>
          <h2 className="text-2xl font-extrabold tracking-tight sm:text-3xl">
            {fullyActive ? "Compte entièrement activé" : "Informations envoyées"}
          </h2>
          <p className="mt-2 max-w-md text-[15px] leading-relaxed text-white/90">
            {fullyActive
              ? "Vous pouvez désormais recevoir des paiements et être rémunéré par virement."
              : "Nos équipes et Stripe examinent vos informations. Vous serez notifié dès validation."}
          </p>
        </div>
      </div>

      {/* Account summary */}
      <div className="rounded-2xl border border-slate-100 bg-white p-6 shadow-sm">
        <h3 className="mb-4 text-[13px] font-semibold uppercase tracking-wider text-slate-500">
          Récapitulatif
        </h3>
        <dl className="grid gap-4 sm:grid-cols-2">
          <SummaryItem
            label="Identifiant compte"
            value={<code className="font-mono text-[13px]">{accountId}</code>}
          />
          <SummaryItem
            label="Pays"
            value={
              countryData ? (
                <span className="flex items-center gap-2">
                  <span className="text-lg leading-none" aria-hidden>
                    {countryData.flag}
                  </span>
                  <span>{countryData.labelFr}</span>
                </span>
              ) : (
                country ?? "—"
              )
            }
          />
          <SummaryItem
            label="Encaissements"
            value={
              <StatusBadge
                tone={chargesEnabled ? "success" : "warning"}
                label={chargesEnabled ? "Activés" : "En attente"}
              />
            }
          />
          <SummaryItem
            label="Virements sortants"
            value={
              <StatusBadge
                tone={payoutsEnabled ? "success" : "warning"}
                label={payoutsEnabled ? "Activés" : "En attente"}
              />
            }
          />
        </dl>
        {requirementsCount > 0 ? (
          <div className="mt-4 flex items-start gap-2.5 rounded-lg border border-amber-200 bg-amber-50 p-3">
            <div className="text-[12px] font-semibold text-amber-900">
              {requirementsCount} information{requirementsCount > 1 ? "s" : ""} en attente
            </div>
          </div>
        ) : null}
      </div>

      {/* Next steps */}
      <div>
        <h3 className="mb-3 text-[13px] font-semibold uppercase tracking-wider text-slate-500">
          Prochaines étapes
        </h3>
        <div className="grid gap-3 sm:grid-cols-3">
          <NextStepCard
            icon={FileText}
            title="Compléter votre profil"
            description="Ajoutez des compétences et références pour attirer les clients"
          />
          <NextStepCard
            icon={CreditCard}
            title="Consulter vos paiements"
            description="Suivez vos encaissements et virements en temps réel"
          />
          <NextStepCard
            icon={Settings}
            title="Paramètres compte"
            description="Modifiez vos informations bancaires ou personnelles"
          />
        </div>
      </div>

      {onRestart ? (
        <div className="flex justify-center">
          <button
            onClick={onRestart}
            className="text-[13px] font-medium text-slate-500 transition-colors hover:text-rose-600"
          >
            Tester à nouveau
          </button>
        </div>
      ) : null}
    </div>
  )
}

function SummaryItem({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div>
      <dt className="text-[12px] font-medium text-slate-500">{label}</dt>
      <dd className="mt-0.5 text-[14px] font-semibold text-slate-900">{value}</dd>
    </div>
  )
}

function StatusBadge({ tone, label }: { tone: "success" | "warning"; label: string }) {
  const styles = {
    success: "bg-emerald-50 text-emerald-700 border-emerald-200",
    warning: "bg-amber-50 text-amber-800 border-amber-200",
  }[tone]
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-semibold ${styles}`}
    >
      <span
        className={`h-1.5 w-1.5 rounded-full ${tone === "success" ? "bg-emerald-500" : "bg-amber-500"}`}
        aria-hidden
      />
      {label}
    </span>
  )
}

function NextStepCard({
  icon: Icon,
  title,
  description,
}: {
  icon: typeof FileText
  title: string
  description: string
}) {
  return (
    <div className="group cursor-pointer rounded-xl border border-slate-100 bg-white p-4 transition-all hover:-translate-y-0.5 hover:border-rose-200 hover:shadow-md">
      <div className="mb-2 flex h-9 w-9 items-center justify-center rounded-lg bg-rose-50 text-rose-600 transition-colors group-hover:bg-gradient-to-br group-hover:from-rose-500 group-hover:to-rose-600 group-hover:text-white">
        <Icon className="h-4 w-4" aria-hidden />
      </div>
      <div className="mb-0.5 flex items-center justify-between">
        <h4 className="text-[13px] font-semibold text-slate-900">{title}</h4>
        <ArrowRight
          className="h-3.5 w-3.5 text-slate-300 transition-all group-hover:translate-x-0.5 group-hover:text-rose-500"
          aria-hidden
        />
      </div>
      <p className="text-[12px] leading-snug text-slate-500">{description}</p>
    </div>
  )
}
