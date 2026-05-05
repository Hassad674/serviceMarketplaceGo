"use client"

import Link from "next/link"
import {
  AlertTriangle,
  ArrowRight,
  CheckCircle2,
  Loader2,
  Pencil,
  ShieldCheck,
  Wallet,
  XCircle,
} from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { ApiError } from "@/shared/lib/api-client"

import { Button } from "@/shared/components/ui/button"
// Hero card for the wallet page. Pure presentational component:
// renders the running total, the Stripe-account readiness line and
// the "Retirer" call-to-action. The actual payout flow (KYC gate,
// billing-profile gate, mutation) lives in `wallet-payout-section`
// — this file only renders the slot for the button it provides.
//
// Soleil v2 — hero card:
//   - Editorial Fraunces title (26px) + corail-soft icon square
//   - Mono uppercase eyebrow "REVENUS TOTAUX" with 0.12em tracking
//   - Big Fraunces 56px stat number (44px on small screens)
//   - Sapin-soft "Compte Stripe prêt" pill (success-soft + success)
//   - Corail "Retirer" pill button (rounded-full, 4px tinted shadow)
//   - Quick-link row separated by a calm border divider

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
    <section
      className={cn(
        "relative overflow-hidden rounded-2xl border border-border bg-card",
        "p-6 sm:p-7",
      )}
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      {/* Decorative corail blob, top-right — Soleil signature */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -right-16 -top-16 h-56 w-56 rounded-full"
        style={{
          background:
            "radial-gradient(circle, rgba(232,93,74,0.07), transparent 65%)",
        }}
      />

      <div className="relative">
        {/* Header — corail-soft icon square + serif title */}
        <div className="mb-5 flex items-start gap-3.5">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary-soft text-primary">
            <Wallet className="h-5 w-5" strokeWidth={1.6} />
          </div>
          <div className="min-w-0">
            <h1 className="font-serif text-[26px] font-medium leading-tight tracking-[-0.02em] text-foreground">
              Portefeuille
            </h1>
            <p className="mt-0.5 text-[13px] text-muted-foreground">
              Vos revenus issus des missions et commissions d&apos;apport
            </p>
          </div>
        </div>

        {/* Total earned + payout CTA row */}
        <div className="grid items-end gap-6 sm:grid-cols-[1fr_auto] sm:items-center">
          <div>
            <p className="mb-1.5 font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-subtle-foreground">
              Revenus totaux
            </p>
            <p className="font-serif text-[44px] font-normal leading-none tracking-[-0.035em] text-foreground sm:text-[56px]">
              {formatEur(totalEarned)}
            </p>
            <div className="mt-3.5">
              <StripeStatusLine
                hasAccount={hasAccount}
                payoutsEnabled={payoutsEnabled}
              />
            </div>
          </div>
          <PayoutButton
            available={available}
            canClick={canClick}
            canWithdraw={canWithdraw}
            payoutPending={payoutPending}
            onPayout={onPayout}
          />
        </div>

        {payoutMessage && (
          <p className="mt-3 text-[13px] font-medium text-success">
            {payoutMessage}
          </p>
        )}
        {payoutError && <PayoutErrorBanner error={payoutError} />}

        {/* Quick links — small, non-prominent shortcuts to the
            settings the user might want to revise from the wallet
            (billing profile + Stripe info). */}
        <QuickLinksRow />
      </div>
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
      <span
        className={cn(
          "inline-flex items-center gap-1.5 rounded-full bg-success-soft px-3 py-1.5",
          "text-[12px] font-semibold text-success",
        )}
      >
        <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
        Compte Stripe prêt — virements activés
      </span>
    )
  }
  if (hasAccount) {
    return (
      <span className="inline-flex items-center gap-2 text-[13px] text-foreground">
        <AlertTriangle
          className="h-4 w-4 text-warning"
          aria-hidden="true"
        />
        <span>Compte Stripe en cours de vérification</span>
        <Link
          href="/payment-info"
          className="font-semibold text-primary hover:text-primary-deep hover:underline"
        >
          Finaliser
        </Link>
      </span>
    )
  }
  return (
    <span className="inline-flex items-center gap-2 text-[13px] text-foreground">
      <XCircle className="h-4 w-4 text-destructive" aria-hidden="true" />
      <span>Compte Stripe non configuré</span>
      <Link
        href="/payment-info"
        className="font-semibold text-primary hover:text-primary-deep hover:underline"
      >
        Configurer
      </Link>
    </span>
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
    <div className="flex flex-col items-stretch gap-1.5 sm:items-end">
      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={onPayout}
        disabled={payoutPending || !canClick}
        title={
          !canWithdraw
            ? "Seul le propriétaire peut demander un retrait"
            : undefined
        }
        className={cn(
          "inline-flex items-center justify-center gap-2 rounded-full",
          "px-5 py-2.5 text-sm font-bold text-primary-foreground",
          "bg-primary transition-all duration-200 ease-out",
          "hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)]",
          "active:scale-[0.98]",
          "disabled:cursor-not-allowed disabled:bg-pink disabled:opacity-60",
          "disabled:hover:bg-pink disabled:hover:shadow-none",
        )}
      >
        {payoutPending ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <ArrowRight className="h-4 w-4" />
        )}
        Retirer {formatEur(available)}
      </Button>
      {available === 0 && (
        <span className="text-[11px] text-subtle-foreground sm:text-right">
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
    <div
      className={cn(
        "mt-3 flex items-start gap-2 rounded-xl border p-3",
        "border-destructive/30 bg-destructive/5 text-destructive",
      )}
    >
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
      {isStripeMissing ? (
        <p className="text-[13px]">
          Vous devez d&apos;abord configurer vos informations de paiement
          pour effectuer un retrait.{" "}
          <Link
            href="/payment-info"
            className="font-semibold underline hover:opacity-80"
          >
            Configurer maintenant
          </Link>
        </p>
      ) : (
        <p className="text-[13px]">Erreur lors du retrait</p>
      )}
    </div>
  )
}

function QuickLinksRow() {
  return (
    <div
      className={cn(
        "mt-5 flex flex-wrap items-center gap-x-5 gap-y-2 pt-4",
        "border-t border-border text-[12.5px] font-medium",
      )}
    >
      <Link
        href="/settings/billing-profile?return_to=/wallet"
        className={cn(
          "inline-flex items-center gap-1.5 text-foreground",
          "transition-colors duration-150 hover:text-primary",
        )}
      >
        <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
        Modifier mes infos de facturation
      </Link>
      <span aria-hidden="true" className="text-border-strong">
        ·
      </span>
      <Link
        href="/payment-info"
        className={cn(
          "inline-flex items-center gap-1.5 text-foreground",
          "transition-colors duration-150 hover:text-primary",
        )}
      >
        <ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />
        Mes infos de paiement Stripe
      </Link>
    </div>
  )
}
