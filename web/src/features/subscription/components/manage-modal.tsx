"use client"

import { useCallback, useState } from "react"
import { cn, formatCurrency, formatDate } from "@/shared/lib/utils"
import { Modal } from "@/shared/components/ui/modal"
import { useSubscription } from "../hooks/use-subscription"
import { useSubscriptionStats } from "../hooks/use-subscription-stats"
import { useToggleAutoRenew } from "../hooks/use-toggle-auto-renew"
import { useChangeCycle } from "../hooks/use-change-cycle"
import { useCyclePreview } from "../hooks/use-cycle-preview"
import { usePortalURL } from "../hooks/use-portal"
import type { BillingCycle, CyclePreview, Subscription } from "../types"

import { Button } from "@/shared/components/ui/button"
type ManageModalProps = {
	open: boolean
	onClose: () => void
}

/**
 * Premium management panel. Reads the cached subscription + stats,
 * orchestrates toggle / cycle-change / portal actions. Each action has
 * its own hook so failures are isolated — a portal error never breaks
 * the auto-renew toggle and vice versa.
 *
 * Delegates dialog plumbing (portal, backdrop, escape, focus trap) to
 * the shared `Modal` primitive.
 */
export function ManageModal({ open, onClose }: ManageModalProps) {
	const { data: subscription } = useSubscription()

	// Nothing to manage on the free tier — render nothing so the caller
	// can open this modal speculatively without a crash when the user
	// happens to have just cancelled.
	if (!subscription) {
		return (
			<Modal open={open} onClose={onClose} title="Gérer l'abonnement">
				<p className="text-sm text-slate-500 dark:text-slate-400">
					Aucun abonnement actif.
				</p>
			</Modal>
		)
	}

	return (
		<Modal open={open} onClose={onClose} title="Gérer l'abonnement">
			<div className="space-y-5">
				<PlanSummary subscription={subscription} />
				<StatsPanel startedAt={subscription.started_at} />
				<AutoRenewSwitch subscription={subscription} />
				<ChangeCycleBlock subscription={subscription} />
				<PortalActions />
			</div>
		</Modal>
	)
}

function PlanSummary({ subscription }: { subscription: Subscription }) {
  const planLabel = subscription.plan === "agency" ? "Premium Agence" : "Premium Freelance"
  const price = priceOf(subscription.plan, subscription.billing_cycle)
  const cycle = subscription.billing_cycle === "annual" ? "Annuel" : "Mensuel"
  const nextCycleLabel = subscription.pending_billing_cycle === "monthly"
    ? "Mensuel"
    : subscription.pending_billing_cycle === "annual"
      ? "Annuel"
      : null
  return (
    <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-800/40">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-slate-900 dark:text-white">
            {planLabel}
          </p>
          <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
            {cycle} · {price} €
          </p>
        </div>
        <p className="text-right text-xs text-slate-500 dark:text-slate-400">
          Prochain renouvellement
          <br />
          <span className="font-medium text-slate-700 dark:text-slate-300">
            {formatDate(subscription.current_period_end)}
          </span>
        </p>
      </div>
      {nextCycleLabel && subscription.pending_cycle_effective_at ? (
        <p className="mt-3 rounded-lg border border-amber-200 bg-amber-50/60 px-3 py-2 text-xs text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300">
          Passage en {nextCycleLabel.toLowerCase()} prévu le{" "}
          <span className="font-semibold">
            {formatDate(subscription.pending_cycle_effective_at)}
          </span>
          . Tu gardes ton accès {cycle.toLowerCase()} jusqu&apos;à cette date.
        </p>
      ) : null}
    </div>
  )
}

function StatsPanel({ startedAt }: { startedAt: string }) {
  const { data: stats } = useSubscriptionStats()
  if (!stats) return null
  const saved = formatCurrency(stats.saved_fee_cents / 100)
  return (
    <div className="rounded-xl border border-rose-200 bg-gradient-to-br from-rose-50 to-white p-4 dark:border-rose-500/30 dark:from-rose-500/10 dark:to-slate-900/40">
      <p className="text-sm text-slate-700 dark:text-slate-200">
        Tu as économisé{" "}
        <span className="font-mono font-semibold text-rose-600 dark:text-rose-300">
          {saved}
        </span>{" "}
        depuis le{" "}
        <span className="font-medium">{formatDate(startedAt)}</span>
      </p>
    </div>
  )
}

function AutoRenewSwitch({ subscription }: { subscription: Subscription }) {
  const toggle = useToggleAutoRenew()
  const checked = !subscription.cancel_at_period_end
  return (
    <div className="flex items-center justify-between gap-3 rounded-xl border border-slate-200 bg-white px-4 py-3 dark:border-slate-700 dark:bg-slate-800/30">
      <div>
        <p className="text-sm font-medium text-slate-900 dark:text-slate-100">
          Renouvellement automatique
        </p>
        <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
          {checked
            ? "Tu seras facturé automatiquement à chaque échéance"
            : "Premium expirera à la fin de la période actuelle"}
        </p>
      </div>
      <SwitchToggle
        checked={checked}
        disabled={toggle.isPending}
        onChange={(v) => toggle.mutate(v)}
        label="Renouvellement automatique"
      />
    </div>
  )
}

function SwitchToggle({
  checked,
  disabled,
  onChange,
  label,
}: {
  checked: boolean
  disabled: boolean
  onChange: (v: boolean) => void
  label: string
}) {
  return (
    <Button variant="ghost" size="auto"
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative inline-flex h-6 w-11 flex-shrink-0 items-center rounded-full transition-colors duration-200",
        "focus:outline-none focus:ring-2 focus:ring-rose-500/40",
        "disabled:cursor-not-allowed disabled:opacity-60",
        checked ? "bg-rose-500" : "bg-slate-300 dark:bg-slate-600",
      )}
    >
      <span
        className={cn(
          "inline-block h-5 w-5 transform rounded-full bg-white shadow-md transition-transform duration-200",
          checked ? "translate-x-5" : "translate-x-0.5",
        )}
      />
    </Button>
  )
}

function ChangeCycleBlock({ subscription }: { subscription: Subscription }) {
  const [target, setTarget] = useState<BillingCycle | null>(null)
  const change = useChangeCycle()
  const preview = useCyclePreview(target)
  const nextTarget: BillingCycle =
    subscription.billing_cycle === "monthly" ? "annual" : "monthly"
  const label =
    nextTarget === "annual" ? "Passer à l'annuel (-21%)" : "Repasser en mensuel"

  const handleCancel = useCallback(() => setTarget(null), [])
  const handleConfirm = useCallback(() => {
    if (!target) return
    change.mutate(target, {
      onSuccess: () => setTarget(null),
    })
  }, [change, target])

  // If a schedule is already pending, swap the CTA to "Annuler le
  // passage programmé" — re-confirming the current cycle releases the
  // schedule cleanly. V1 keeps this implicit via the upgrade path
  // clearing pendings; a dedicated cancel button is a B.2 concern.
  const hasPending = Boolean(subscription.pending_billing_cycle)

  // Downgrades schedule a future transition via a Stripe Subscription
  // Schedule, which overrides cancel_at_period_end. If auto-renew is off
  // we would silently resume charging at the phase boundary — the server
  // now rejects this case and the UI mirrors it so the user understands
  // what to do before clicking.
  const isDowngradeTarget = nextTarget === "monthly"
  const downgradeBlocked = isDowngradeTarget && subscription.cancel_at_period_end
  const disabled = hasPending || downgradeBlocked
  const hint = hasPending
    ? "Changement déjà programmé"
    : downgradeBlocked
      ? "Active le renouvellement automatique avant de programmer un passage en mensuel."
      : null

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-800/30">
      <p className="text-xs text-slate-500 dark:text-slate-400">
        Cycle actuel :{" "}
        <span className="font-medium text-slate-700 dark:text-slate-300">
          {subscription.billing_cycle === "annual" ? "annuel" : "mensuel"}
        </span>
      </p>
      {target ? (
        <ConfirmCycleChange
          target={target}
          preview={preview.data ?? null}
          loading={preview.isLoading}
          error={preview.isError}
          pending={change.isPending}
          currentPeriodEnd={subscription.current_period_end}
          onCancel={handleCancel}
          onConfirm={handleConfirm}
        />
      ) : (
        <>
          <Button variant="ghost" size="auto"
            type="button"
            onClick={() => setTarget(nextTarget)}
            disabled={disabled}
            title={hint ?? undefined}
            className={cn(
              "mt-2 w-full rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-xs font-semibold text-rose-600",
              "transition-colors duration-200 hover:bg-rose-100",
              "dark:border-rose-500/40 dark:bg-rose-500/10 dark:text-rose-300 dark:hover:bg-rose-500/20",
              "disabled:cursor-not-allowed disabled:opacity-60",
            )}
          >
            {hasPending ? "Changement déjà programmé" : label}
          </Button>
          {downgradeBlocked ? (
            <p className="mt-2 text-[11px] leading-snug text-amber-700 dark:text-amber-300">
              Active le renouvellement automatique avant de programmer un passage en mensuel.
            </p>
          ) : null}
        </>
      )}
    </div>
  )
}

function ConfirmCycleChange({
  target,
  preview,
  loading,
  error,
  pending,
  currentPeriodEnd,
  onCancel,
  onConfirm,
}: {
  target: BillingCycle
  preview: CyclePreview | null
  loading: boolean
  error: boolean
  pending: boolean
  currentPeriodEnd: string
  onCancel: () => void
  onConfirm: () => void
}) {
  return (
    <div className="mt-3 space-y-2">
      <PreviewMessage
        target={target}
        preview={preview}
        loading={loading}
        error={error}
        currentPeriodEnd={currentPeriodEnd}
      />
      <div className="flex gap-2">
        <Button variant="ghost" size="auto"
          type="button"
          onClick={onCancel}
          disabled={pending}
          className="flex-1 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-medium text-slate-700 transition-colors hover:bg-slate-50 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
        >
          Annuler
        </Button>
        <Button variant="ghost" size="auto"
          type="button"
          onClick={onConfirm}
          disabled={pending || loading || error}
          className={cn(
            "flex-1 rounded-lg bg-rose-500 px-3 py-2 text-xs font-semibold text-white",
            "transition-all hover:bg-rose-600 active:scale-[0.98]",
            "disabled:cursor-not-allowed disabled:opacity-60",
          )}
        >
          {pending ? "…" : "Confirmer"}
        </Button>
      </div>
    </div>
  )
}

function PreviewMessage({
  target,
  preview,
  loading,
  error,
  currentPeriodEnd,
}: {
  target: BillingCycle
  preview: CyclePreview | null
  loading: boolean
  error: boolean
  currentPeriodEnd: string
}) {
  if (loading) {
    return (
      <p className="text-xs text-slate-500 dark:text-slate-400">
        Calcul du montant…
      </p>
    )
  }
  if (error || !preview) {
    return (
      <p className="text-xs text-red-600 dark:text-red-400">
        Impossible d&apos;afficher le montant. Réessaie plus tard.
      </p>
    )
  }
  if (preview.prorate_immediately) {
    // Upgrade — Stripe charges the delta today.
    return (
      <p className="text-xs text-slate-600 dark:text-slate-300">
        Tu seras facturé{" "}
        <span className="font-semibold text-slate-900 dark:text-white">
          {formatCurrency(preview.amount_due_cents / 100)}
        </span>{" "}
        aujourd&apos;hui. Ton cycle passe en{" "}
        {target === "annual" ? "annuel" : "mensuel"} immédiatement.
      </p>
    )
  }
  // Downgrade — scheduled, no charge today, user keeps current access
  // until the end of the CURRENT period. We intentionally ignore
  // preview.period_end here: on an interval change Stripe's preview
  // returns the next monthly invoice window (today + 1 month), which
  // would misleadingly shorten the access date displayed to the user.
  return (
    <p className="text-xs text-slate-600 dark:text-slate-300">
      Aucun débit aujourd&apos;hui. Tu gardes ton accès jusqu&apos;au{" "}
      <span className="font-semibold text-slate-900 dark:text-white">
        {formatDate(currentPeriodEnd)}
      </span>
      , puis tu passeras en {target === "annual" ? "annuel" : "mensuel"}.
    </p>
  )
}

function PortalActions() {
  const portal = usePortalURL()
  const handleOpen = useCallback(() => {
    portal.mutate(undefined, {
      onSuccess: (url) => window.open(url, "_blank", "noopener,noreferrer"),
    })
  }, [portal])

  return (
    <div className="flex flex-col gap-2 border-t border-slate-200 pt-4 dark:border-slate-700">
      <PortalButton
        label="Gérer mon paiement"
        onClick={handleOpen}
        disabled={portal.isPending}
      />
      <PortalButton
        label="Voir mes factures"
        onClick={handleOpen}
        disabled={portal.isPending}
      />
    </div>
  )
}

function PortalButton({
  label,
  onClick,
  disabled,
}: {
  label: string
  onClick: () => void
  disabled: boolean
}) {
  return (
    <Button variant="ghost" size="auto"
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "rounded-lg px-3 py-2 text-xs font-medium text-slate-600 transition-colors",
        "hover:bg-slate-100 hover:text-slate-900",
        "dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-slate-100",
        "disabled:cursor-not-allowed disabled:opacity-60",
      )}
    >
      {label}
    </Button>
  )
}

function priceOf(plan: Subscription["plan"], cycle: BillingCycle): number {
  if (plan === "agency") return cycle === "annual" ? 468 : 49
  return cycle === "annual" ? 180 : 19
}
