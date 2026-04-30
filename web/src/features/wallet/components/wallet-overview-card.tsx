"use client"

import Link from "next/link"
import {
  AlertTriangle,
  ArrowDownToLine,
  CheckCircle2,
  Loader2,
  Pencil,
  ShieldCheck,
  Wallet,
  XCircle,
} from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { ApiError } from "@/shared/lib/api-client"

// Hero card for the wallet page. Pure presentational component:
// renders the running total, the Stripe-account readiness line and
// the "Retirer" call-to-action. The actual payout flow (KYC gate,
// billing-profile gate, mutation) lives in `wallet-payout-section`
// — this file only renders the slot for the button it provides.

function formatEur(centimes: number): string {
  return new Intl.NumberFormat("fr-FR", {
    style: "currency",
    currency: "EUR",
  }).format(centimes / 100)
}

export interface WalletOverviewCardProps {
  totalEarned: number
  available: number
  stripeAccountId: string
  payoutsEnabled: boolean
  canWithdraw: boolean
  payoutPending: boolean
  payoutMessage: string
  payoutError: Error | null
  onPayout: () => void
}

export function WalletOverviewCard({
  totalEarned,
  available,
  stripeAccountId,
  payoutsEnabled,
  canWithdraw,
  payoutPending,
  payoutMessage,
  payoutError,
  onPayout,
}: WalletOverviewCardProps) {
  const hasAccount = Boolean(stripeAccountId)
  const canClick = canWithdraw && available > 0

  return (
    <section className="rounded-2xl border border-slate-100 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-800/80">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
            <Wallet className="h-5 w-5 text-rose-600 dark:text-rose-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
              Portefeuille
            </h1>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              Vos revenus issus des missions et commissions d&apos;apport
            </p>
          </div>
        </div>
      </div>

      {/* Total earned — hero number */}
      <div className="mt-6">
        <p className="text-xs font-medium uppercase tracking-wide text-slate-400">
          Revenus totaux
        </p>
        <p className="mt-1 font-mono text-4xl font-extrabold tracking-tight text-slate-900 dark:text-white">
          {formatEur(totalEarned)}
        </p>
      </div>

      {/* Stripe status line + withdraw CTA row */}
      <div className="mt-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <StripeStatusLine
          hasAccount={hasAccount}
          payoutsEnabled={payoutsEnabled}
        />
        <PayoutButton
          available={available}
          canClick={canClick}
          canWithdraw={canWithdraw}
          payoutPending={payoutPending}
          onPayout={onPayout}
        />
      </div>

      {payoutMessage && (
        <p className="mt-3 text-sm text-green-600 dark:text-green-400">
          {payoutMessage}
        </p>
      )}
      {payoutError && <PayoutErrorBanner error={payoutError} />}

      {/* Quick links — small, non-prominent shortcuts to the
          settings the user might want to revise from the wallet
          (billing profile + Stripe info). */}
      <QuickLinksRow />
    </section>
  )
}

function StripeStatusLine({
  hasAccount,
  payoutsEnabled,
}: {
  hasAccount: boolean
  payoutsEnabled: boolean
}) {
  if (hasAccount && payoutsEnabled) {
    return (
      <div className="flex items-center gap-2 text-sm">
        <CheckCircle2
          className="h-4 w-4 text-green-500"
          aria-hidden="true"
        />
        <span className="text-slate-700 dark:text-slate-300">
          Compte Stripe prêt — virements activés
        </span>
      </div>
    )
  }
  if (hasAccount) {
    return (
      <div className="flex items-center gap-2 text-sm">
        <AlertTriangle
          className="h-4 w-4 text-amber-500"
          aria-hidden="true"
        />
        <span className="text-slate-700 dark:text-slate-300">
          Compte Stripe en cours de vérification
        </span>
        <Link
          href="/payment-info"
          className="text-rose-600 hover:underline dark:text-rose-400"
        >
          Finaliser
        </Link>
      </div>
    )
  }
  return (
    <div className="flex items-center gap-2 text-sm">
      <XCircle className="h-4 w-4 text-red-500" aria-hidden="true" />
      <span className="text-slate-700 dark:text-slate-300">
        Compte Stripe non configuré
      </span>
      <Link
        href="/payment-info"
        className="text-rose-600 hover:underline dark:text-rose-400"
      >
        Configurer
      </Link>
    </div>
  )
}

function PayoutButton({
  available,
  canClick,
  canWithdraw,
  payoutPending,
  onPayout,
}: {
  available: number
  canClick: boolean
  canWithdraw: boolean
  payoutPending: boolean
  onPayout: () => void
}) {
  return (
    <div className="flex flex-col items-stretch gap-1 sm:items-end">
      <button
        type="button"
        onClick={onPayout}
        disabled={payoutPending || !canClick}
        title={
          !canWithdraw
            ? "Seul le propriétaire peut demander un retrait"
            : undefined
        }
        className={cn(
          "inline-flex items-center justify-center gap-2 rounded-xl px-5 py-2.5",
          "text-sm font-semibold text-white transition-all duration-200",
          "gradient-primary hover:shadow-glow active:scale-[0.98]",
          "disabled:opacity-50 disabled:cursor-not-allowed",
        )}
      >
        {payoutPending ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <ArrowDownToLine className="h-4 w-4" />
        )}
        Retirer {formatEur(available)}
      </button>
      {available === 0 && (
        <span className="text-xs text-slate-400 dark:text-slate-500 sm:text-right">
          Aucun fonds disponible
        </span>
      )}
    </div>
  )
}

function PayoutErrorBanner({ error }: { error: Error }) {
  const isStripeMissing =
    error instanceof ApiError && error.code === "stripe_account_missing"
  return (
    <div className="mt-3 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-500/20 dark:bg-red-500/10">
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-red-500" />
      {isStripeMissing ? (
        <p className="text-sm text-red-700 dark:text-red-400">
          Vous devez d&apos;abord configurer vos informations de paiement
          pour effectuer un retrait.{" "}
          <Link
            href="/payment-info"
            className="font-semibold underline hover:text-red-900 dark:hover:text-red-300"
          >
            Configurer maintenant
          </Link>
        </p>
      ) : (
        <p className="text-sm text-red-700 dark:text-red-400">
          Erreur lors du retrait
        </p>
      )}
    </div>
  )
}

function QuickLinksRow() {
  return (
    <div className="mt-5 flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-slate-100 pt-4 text-xs dark:border-slate-700">
      <Link
        href="/settings/billing-profile?return_to=/wallet"
        className="inline-flex items-center gap-1.5 text-slate-600 hover:text-rose-600 dark:text-slate-400 dark:hover:text-rose-400"
      >
        <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
        Modifier mes infos de facturation
      </Link>
      <span aria-hidden="true" className="text-slate-300 dark:text-slate-600">
        ·
      </span>
      <Link
        href="/payment-info"
        className="inline-flex items-center gap-1.5 text-slate-600 hover:text-rose-600 dark:text-slate-400 dark:hover:text-rose-400"
      >
        <ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />
        Mes infos de paiement Stripe
      </Link>
    </div>
  )
}
