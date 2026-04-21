"use client"

import { useCallback, useEffect, useId, useRef, useState } from "react"
import { X } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { formatCurrency, formatDate } from "@/shared/lib/utils"
import { useSubscription } from "../hooks/use-subscription"
import { useSubscriptionStats } from "../hooks/use-subscription-stats"
import { useToggleAutoRenew } from "../hooks/use-toggle-auto-renew"
import { useChangeCycle } from "../hooks/use-change-cycle"
import { usePortalURL } from "../hooks/use-portal"
import type { BillingCycle, Subscription } from "../types"

type ManageModalProps = {
  open: boolean
  onClose: () => void
}

/**
 * Premium management panel. Reads the cached subscription +
 * stats, orchestrates the toggle / cycle-change / portal actions.
 * Each action has its own hook so failures are isolated — a
 * portal error never breaks the auto-renew toggle and vice versa.
 */
export function ManageModal({ open, onClose }: ManageModalProps) {
  const { data: subscription } = useSubscription()
  const titleId = useId()
  const triggerRef = useRef<HTMLElement | null>(null)
  const dialogRef = useRef<HTMLDivElement | null>(null)
  useModalKeyboard({ open, onClose, triggerRef, dialogRef })

  if (!open) return null
  if (!subscription) return null

  return (
    <ModalShell onDismiss={onClose} labelledBy={titleId} dialogRef={dialogRef}>
      <ManageHeader id={titleId} onClose={onClose} />
      <div className="space-y-5">
        <PlanSummary subscription={subscription} />
        <StatsPanel startedAt={subscription.started_at} />
        <AutoRenewSwitch subscription={subscription} />
        <ChangeCycleBlock subscription={subscription} />
        <PortalActions />
      </div>
    </ModalShell>
  )
}

function useModalKeyboard({
  open,
  onClose,
  triggerRef,
  dialogRef,
}: {
  open: boolean
  onClose: () => void
  triggerRef: React.MutableRefObject<HTMLElement | null>
  dialogRef: React.MutableRefObject<HTMLDivElement | null>
}) {
  useEffect(() => {
    if (!open) return
    // document.activeElement is typed Element | null. The trigger
    // that opens the modal is always a focusable HTMLElement (the
    // badge button), and we call focus() optionally on cleanup —
    // so the narrowing is safe.
    triggerRef.current = document.activeElement as HTMLElement | null
    const dialog = dialogRef.current
    const focusable = dialog?.querySelector<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
    )
    focusable?.focus()
    return () => {
      triggerRef.current?.focus?.()
    }
  }, [open, dialogRef, triggerRef])

  useEffect(() => {
    if (!open) return
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose()
      if (e.key === "Tab") trapFocus(e, dialogRef.current)
    }
    document.addEventListener("keydown", handleKey)
    return () => document.removeEventListener("keydown", handleKey)
  }, [open, onClose, dialogRef])
}

function trapFocus(e: KeyboardEvent, dialog: HTMLElement | null) {
  if (!dialog) return
  const nodes = dialog.querySelectorAll<HTMLElement>(
    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
  )
  if (nodes.length === 0) return
  const first = nodes[0]
  const last = nodes[nodes.length - 1]
  if (e.shiftKey && document.activeElement === first) {
    e.preventDefault()
    last.focus()
  } else if (!e.shiftKey && document.activeElement === last) {
    e.preventDefault()
    first.focus()
  }
}

function ModalShell({
  children,
  onDismiss,
  labelledBy,
  dialogRef,
}: {
  children: React.ReactNode
  onDismiss: () => void
  labelledBy: string
  dialogRef: React.MutableRefObject<HTMLDivElement | null>
}) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-sm"
      onClick={(e) => e.target === e.currentTarget && onDismiss()}
      role="presentation"
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={labelledBy}
        // max-h + overflow guards against short viewports (laptops around
        // 720px tall): content taller than the screen becomes internally
        // scrollable instead of overflowing off the top or bottom.
        className="flex max-h-[calc(100vh-2rem)] w-full max-w-md animate-scale-in flex-col overflow-y-auto rounded-2xl bg-white p-6 shadow-xl dark:bg-slate-900"
      >
        {children}
      </div>
    </div>
  )
}

function ManageHeader({ id, onClose }: { id: string; onClose: () => void }) {
  return (
    <div className="mb-5 flex items-start justify-between">
      <h2
        id={id}
        className="text-lg font-semibold text-slate-900 dark:text-white"
      >
        Gérer mon abonnement
      </h2>
      <button
        type="button"
        onClick={onClose}
        aria-label="Fermer"
        className="rounded-lg p-1.5 text-slate-500 transition-colors hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
      >
        <X className="h-5 w-5" aria-hidden="true" />
      </button>
    </div>
  )
}

function PlanSummary({ subscription }: { subscription: Subscription }) {
  const planLabel = subscription.plan === "agency" ? "Premium Agence" : "Premium Freelance"
  const price = priceOf(subscription.plan, subscription.billing_cycle)
  const cycle = subscription.billing_cycle === "annual" ? "Annuel" : "Mensuel"
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
    <button
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
    </button>
  )
}

function ChangeCycleBlock({ subscription }: { subscription: Subscription }) {
  const [confirming, setConfirming] = useState(false)
  const change = useChangeCycle()
  const target: BillingCycle = subscription.billing_cycle === "monthly" ? "annual" : "monthly"
  const label =
    target === "annual" ? "Passer à l'annuel (-21%)" : "Repasser en mensuel"

  const handleConfirm = useCallback(() => {
    change.mutate(target, {
      onSuccess: () => setConfirming(false),
    })
  }, [change, target])

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-800/30">
      <p className="text-xs text-slate-500 dark:text-slate-400">
        Cycle actuel :{" "}
        <span className="font-medium text-slate-700 dark:text-slate-300">
          {subscription.billing_cycle === "annual" ? "annuel" : "mensuel"}
        </span>
      </p>
      {confirming ? (
        <ConfirmCycleChange
          onCancel={() => setConfirming(false)}
          onConfirm={handleConfirm}
          pending={change.isPending}
        />
      ) : (
        <button
          type="button"
          onClick={() => setConfirming(true)}
          className={cn(
            "mt-2 w-full rounded-lg border border-rose-300 bg-rose-50 px-3 py-2 text-xs font-semibold text-rose-600",
            "transition-colors duration-200 hover:bg-rose-100",
            "dark:border-rose-500/40 dark:bg-rose-500/10 dark:text-rose-300 dark:hover:bg-rose-500/20",
          )}
        >
          {label}
        </button>
      )}
    </div>
  )
}

function ConfirmCycleChange({
  onCancel,
  onConfirm,
  pending,
}: {
  onCancel: () => void
  onConfirm: () => void
  pending: boolean
}) {
  return (
    <div className="mt-3 space-y-2">
      <p className="text-xs text-slate-600 dark:text-slate-300">
        En confirmant, Stripe applique une proration immédiate pour le montant restant.
      </p>
      <div className="flex gap-2">
        <button
          type="button"
          onClick={onCancel}
          disabled={pending}
          className="flex-1 rounded-lg border border-slate-300 bg-white px-3 py-2 text-xs font-medium text-slate-700 transition-colors hover:bg-slate-50 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700"
        >
          Annuler
        </button>
        <button
          type="button"
          onClick={onConfirm}
          disabled={pending}
          className={cn(
            "flex-1 rounded-lg bg-rose-500 px-3 py-2 text-xs font-semibold text-white",
            "transition-all hover:bg-rose-600 active:scale-[0.98]",
            "disabled:cursor-not-allowed disabled:opacity-60",
          )}
        >
          {pending ? "…" : "Confirmer"}
        </button>
      </div>
    </div>
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
    <button
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
    </button>
  )
}

function priceOf(plan: Subscription["plan"], cycle: BillingCycle): number {
  if (plan === "agency") return cycle === "annual" ? 468 : 49
  return cycle === "annual" ? 180 : 19
}
