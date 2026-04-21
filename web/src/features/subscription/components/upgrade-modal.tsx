"use client"

import { useCallback, useEffect, useId, useRef, useState } from "react"
import { X } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { useSubscribe } from "../hooks/use-subscribe"
import type { BillingCycle, Plan } from "../types"

type UpgradeModalProps = {
  open: boolean
  role: "freelance" | "agency"
  onClose: () => void
}

/**
 * Upgrade flow for free users. The modal carries all of the
 * session state (cycle, auto-renew) and fires a single mutation
 * that delegates to Stripe Checkout. The submit button is kept
 * sharp: the price label updates with the cycle so the user
 * always sees the amount they'll be charged on the next screen.
 */
export function UpgradeModal({ open, role, onClose }: UpgradeModalProps) {
  const [cycle, setCycle] = useState<BillingCycle>("monthly")
  const [autoRenew, setAutoRenew] = useState(false)
  const subscribe = useSubscribe()
  const titleId = useId()
  const triggerRef = useRef<HTMLElement | null>(null)
  const dialogRef = useRef<HTMLDivElement | null>(null)
  useModalKeyboard({ open, onClose, triggerRef, dialogRef })

  const handleClose = useCallback(() => {
    subscribe.reset()
    onClose()
  }, [subscribe, onClose])

  if (!open) return null

  const plan: Plan = role === "agency" ? "agency" : "freelance"
  const { monthlyAmount, annualAmount, annualPerMonth } = pricing(role)
  const ctaAmount = cycle === "monthly" ? monthlyAmount : annualAmount

  function handleSubmit() {
    subscribe.mutate({ plan, billing_cycle: cycle, auto_renew: autoRenew })
  }

  return (
    <ModalShell onDismiss={handleClose} labelledBy={titleId} dialogRef={dialogRef}>
      <UpgradeHeader id={titleId} onClose={handleClose} />
      <div className="space-y-5">
        <CycleToggle cycle={cycle} onChange={setCycle} />
        <PlanCard
          role={role}
          cycle={cycle}
          monthlyAmount={monthlyAmount}
          annualAmount={annualAmount}
          annualPerMonth={annualPerMonth}
        />
        <AutoRenewCheckbox checked={autoRenew} onChange={setAutoRenew} />
        {subscribe.isError ? (
          <p
            role="alert"
            className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300"
          >
            Impossible de démarrer le paiement. Réessaie.
          </p>
        ) : null}
        <SubmitRow
          amountEuros={ctaAmount}
          pending={subscribe.isPending}
          onSubmit={handleSubmit}
        />
        <p className="text-center text-xs text-slate-500 dark:text-slate-400">
          Tu peux annuler à tout moment
        </p>
      </div>
    </ModalShell>
  )
}

function pricing(role: "freelance" | "agency") {
  if (role === "agency") {
    return { monthlyAmount: 49, annualAmount: 468, annualPerMonth: 39 }
  }
  return { monthlyAmount: 19, annualAmount: 180, annualPerMonth: 15 }
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

function UpgradeHeader({ id, onClose }: { id: string; onClose: () => void }) {
  return (
    <div className="mb-5 flex items-start justify-between">
      <h2
        id={id}
        className="text-lg font-semibold text-slate-900 dark:text-white"
      >
        Passer Premium · 0% de frais
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

function CycleToggle({
  cycle,
  onChange,
}: {
  cycle: BillingCycle
  onChange: (c: BillingCycle) => void
}) {
  return (
    <div
      role="tablist"
      aria-label="Periode de facturation"
      className="flex items-center gap-1 rounded-full border border-slate-200 bg-slate-50 p-1 dark:border-slate-700 dark:bg-slate-800/50"
    >
      <CycleTab active={cycle === "monthly"} onClick={() => onChange("monthly")}>
        Mensuel
      </CycleTab>
      <CycleTab active={cycle === "annual"} onClick={() => onChange("annual")}>
        <span className="flex items-center gap-1.5">
          Annuel
          <span className="inline-flex items-center rounded-full bg-rose-500 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wider text-white">
            -21%
          </span>
        </span>
      </CycleTab>
    </div>
  )
}

function CycleTab({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={onClick}
      className={cn(
        "flex-1 rounded-full px-3 py-1.5 text-xs font-semibold transition-all duration-200",
        active
          ? "bg-white text-slate-900 shadow-sm dark:bg-slate-700 dark:text-white"
          : "text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200",
      )}
    >
      {children}
    </button>
  )
}

function PlanCard({
  role,
  cycle,
  monthlyAmount,
  annualAmount,
  annualPerMonth,
}: {
  role: "freelance" | "agency"
  cycle: BillingCycle
  monthlyAmount: number
  annualAmount: number
  annualPerMonth: number
}) {
  const title = role === "agency" ? "Premium Agence" : "Premium Freelance"
  return (
    <div className="rounded-xl border border-slate-200 bg-gradient-to-br from-rose-50 to-white p-5 dark:border-slate-700 dark:from-rose-500/10 dark:to-slate-900/40">
      <p className="text-sm font-semibold text-slate-900 dark:text-white">{title}</p>
      {cycle === "monthly" ? (
        <p className="mt-2 text-3xl font-bold text-slate-900 dark:text-white">
          {monthlyAmount} €
          <span className="ml-1 text-sm font-medium text-slate-500 dark:text-slate-400">
            /mois
          </span>
        </p>
      ) : (
        <div className="mt-2">
          <p className="text-3xl font-bold text-slate-900 dark:text-white">
            {annualPerMonth} €
            <span className="ml-1 text-sm font-medium text-slate-500 dark:text-slate-400">
              /mois
            </span>
          </p>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
            Facturé {annualAmount} €/an
          </p>
        </div>
      )}
    </div>
  )
}

function AutoRenewCheckbox({
  checked,
  onChange,
}: {
  checked: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <label className="flex cursor-pointer items-start gap-3 rounded-xl border border-slate-200 bg-slate-50 px-3 py-3 transition-colors hover:bg-slate-100 dark:border-slate-700 dark:bg-slate-800/40 dark:hover:bg-slate-800">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="mt-0.5 h-4 w-4 flex-shrink-0 cursor-pointer rounded border-slate-300 text-rose-600 focus:ring-2 focus:ring-rose-500/30"
      />
      <span className="space-y-1 text-sm">
        <span className="block font-medium text-slate-900 dark:text-slate-100">
          Activer le renouvellement automatique
        </span>
        <span className="block text-xs text-slate-500 dark:text-slate-400">
          Si désactivé, tu paies une fois puis Premium expire naturellement.
        </span>
      </span>
    </label>
  )
}

function SubmitRow({
  amountEuros,
  pending,
  onSubmit,
}: {
  amountEuros: number
  pending: boolean
  onSubmit: () => void
}) {
  return (
    <button
      type="button"
      onClick={onSubmit}
      disabled={pending}
      className={cn(
        "w-full rounded-full bg-gradient-to-r from-rose-500 to-rose-600 px-4 py-3 text-sm font-semibold text-white",
        "transition-all duration-200 hover:shadow-glow active:scale-[0.98]",
        "disabled:cursor-not-allowed disabled:opacity-60 disabled:hover:shadow-none",
        "focus:outline-none focus:ring-2 focus:ring-rose-500/40",
      )}
    >
      {pending ? "Redirection…" : `Payer ${amountEuros} €`}
    </button>
  )
}
