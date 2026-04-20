"use client"

import { Receipt } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { useFeePreview } from "../hooks/use-fee-preview"
import type { FeePreview, FeePreviewTier } from "../types"

type Milestone = {
  /** Stable key for list rendering (e.g. milestone index). */
  key: string
  /** Milestone label shown on the breakdown row. */
  label: string
  /** Amount in centimes. Non-positive values are rendered as "—". */
  amountCents: number
}

type FeePreviewProps = {
  /**
   * One-time mode: a single milestone with the full proposal amount.
   * Milestone mode: one entry per milestone; the component renders a
   * breakdown and cumulates the fees across all milestones.
   */
  milestones: Milestone[]
  /** Controls whether to render the milestone-by-milestone breakdown. */
  mode: "one_time" | "milestone"
  /**
   * Optional recipient UUID. When set, the backend resolves roles on
   * the (caller, recipient) pair; the returned `viewer_is_provider`
   * flag determines whether this component renders anything at all.
   * When omitted, the backend assumes the caller is the provider —
   * appropriate for the received-proposal page where isProvider is
   * already confirmed client-side before this component mounts.
   */
  recipientId?: string
  /**
   * Overrides the heading text. Defaults to "Frais plateforme" for
   * the create-proposal form; the received-proposal view uses a more
   * explicit wording so the prestataire understands it's a preview
   * of what they'll be charged if they accept.
   */
  heading?: string
}

export function FeePreview({ milestones, mode, recipientId, heading }: FeePreviewProps) {
  // When any milestone is empty we still want to surface the tier grid
  // without hammering the backend. We pick the first positive amount
  // as the "reference" that drives the active-tier highlight — this
  // matches what a prestataire would expect (tier visible as soon as
  // they type any one amount).
  const referenceAmount = pickReferenceAmount(milestones)
  const query = useFeePreview(referenceAmount, recipientId)

  // Fail-closed: once the backend has answered, hide the entire
  // section from client-side viewers (enterprise, agency paired
  // with a provider, invalid combos). Loading / error states fire
  // before we know the flag — callers accept that a client may
  // briefly see a skeleton, never fee data.
  if (query.data?.viewer_is_provider === false) return null

  return (
    <section
      aria-labelledby="fee-preview-heading"
      className={cn(
        "rounded-2xl border border-slate-100 bg-white p-5 shadow-sm",
        "dark:border-slate-700 dark:bg-slate-800/80",
      )}
    >
      <FeePreviewHeader heading={heading} />
      <FeePreviewBody
        query={query}
        milestones={milestones}
        mode={mode}
      />
    </section>
  )
}

function FeePreviewHeader({ heading }: { heading?: string }) {
  return (
    <div className="mb-4 flex items-start gap-3">
      <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
        <Receipt className="h-4 w-4 text-rose-600 dark:text-rose-400" aria-hidden="true" />
      </div>
      <div>
        <h3
          id="fee-preview-heading"
          className="text-sm font-semibold text-slate-900 dark:text-white"
        >
          {heading ?? "Frais plateforme"}
        </h3>
        <p className="text-xs text-slate-500 dark:text-slate-400">
          Frais fixes par jalon, selon votre grille tarifaire
        </p>
      </div>
    </div>
  )
}

function FeePreviewBody({
  query,
  milestones,
  mode,
}: {
  query: ReturnType<typeof useFeePreview>
  milestones: Milestone[]
  mode: "one_time" | "milestone"
}) {
  if (query.isLoading) return <FeePreviewSkeleton />
  if (query.isError) {
    return (
      <p
        role="alert"
        className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-400"
      >
        Impossible d&apos;afficher les frais
      </p>
    )
  }
  if (!query.data) return <FeePreviewEmpty />

  return (
    <div className="space-y-4">
      <TierGrid tiers={query.data.tiers} activeIndex={query.data.active_tier_index} />
      {mode === "one_time" ? (
        <OneTimeSummary preview={query.data} />
      ) : (
        <MilestoneBreakdown milestones={milestones} tiers={query.data.tiers} />
      )}
    </div>
  )
}

function FeePreviewEmpty() {
  return (
    <p className="text-xs text-slate-500 dark:text-slate-400">
      Renseignez un montant pour simuler les frais plateforme.
    </p>
  )
}

function TierGrid({ tiers, activeIndex }: { tiers: FeePreviewTier[]; activeIndex: number }) {
  return (
    <ul
      aria-live="polite"
      className="divide-y divide-slate-100 overflow-hidden rounded-xl border border-slate-100 dark:divide-slate-700 dark:border-slate-700"
    >
      {tiers.map((tier, idx) => (
        <TierRow
          key={tier.label}
          tier={tier}
          isActive={idx === activeIndex}
        />
      ))}
    </ul>
  )
}

function TierRow({ tier, isActive }: { tier: FeePreviewTier; isActive: boolean }) {
  return (
    <li
      className={cn(
        "flex items-center justify-between px-4 py-2.5 text-sm transition-colors duration-200",
        isActive
          ? "border-l-4 border-l-rose-500 bg-rose-50 dark:border-l-rose-400 dark:bg-rose-500/10"
          : "border-l-4 border-l-transparent",
      )}
      aria-current={isActive ? "true" : undefined}
    >
      <span
        className={cn(
          "text-slate-700 dark:text-slate-300",
          isActive && "font-medium text-rose-700 dark:text-rose-300",
        )}
      >
        {tier.label}
      </span>
      <span
        className={cn(
          "font-mono text-sm font-semibold text-slate-900 dark:text-white",
          isActive && "text-rose-700 dark:text-rose-200",
        )}
      >
        {formatEur(tier.fee_cents)}
      </span>
    </li>
  )
}

function OneTimeSummary({ preview }: { preview: FeePreview }) {
  return (
    <div className="rounded-xl bg-slate-50 px-4 py-3 dark:bg-slate-900/40">
      <p className="text-sm text-slate-700 dark:text-slate-200">
        Tu encaisses{" "}
        <span className="font-mono font-semibold text-slate-900 dark:text-white">
          {formatEur(preview.net_cents)}
        </span>{" "}
        <span className="text-slate-500 dark:text-slate-400">
          (frais {formatEur(preview.fee_cents)})
        </span>
      </p>
    </div>
  )
}

function MilestoneBreakdown({
  milestones,
  tiers,
}: {
  milestones: Milestone[]
  tiers: FeePreviewTier[]
}) {
  // The backend charges a fee per milestone based on its own amount, so
  // milestones spanning different tiers (e.g. 300 € + 1 500 €) pay
  // different fees. The tier grid is identical for a given user role,
  // so one preview query gives us everything we need — we compute each
  // milestone's fee locally by matching its amount against the grid.
  // Correct total = sum over milestones, never referenceFee × count.
  const totalFees = milestones.reduce((acc, m) => acc + feeForAmount(m.amountCents, tiers), 0)

  return (
    <div className="space-y-3">
      <MilestoneRows milestones={milestones} tiers={tiers} />
      <MilestoneTotals totalFees={totalFees} />
    </div>
  )
}

function MilestoneRows({
  milestones,
  tiers,
}: {
  milestones: Milestone[]
  tiers: FeePreviewTier[]
}) {
  if (milestones.length === 0) {
    return (
      <p className="text-xs text-slate-500 dark:text-slate-400">
        Ajoutez au moins un jalon pour simuler les frais.
      </p>
    )
  }
  return (
    <ul className="space-y-1.5">
      {milestones.map((m) => (
        <MilestoneRow key={m.key} milestone={m} fee={feeForAmount(m.amountCents, tiers)} />
      ))}
    </ul>
  )
}

// feeForAmount mirrors the backend billing.Calculate rule: walk the tiers
// and return the fee of the first tier whose upper bound is greater than
// the amount (the last tier has a null max_cents — open-ended). Returns
// zero for non-positive amounts so the UI shows "—" for empty rows.
function feeForAmount(amountCents: number, tiers: FeePreviewTier[]): number {
  if (amountCents <= 0) return 0
  for (const tier of tiers) {
    if (tier.max_cents === null || amountCents < tier.max_cents) {
      return tier.fee_cents
    }
  }
  return 0
}

function MilestoneRow({ milestone, fee }: { milestone: Milestone; fee: number }) {
  const hasAmount = milestone.amountCents > 0
  return (
    <li className="flex items-center justify-between text-xs text-slate-600 dark:text-slate-300">
      <span className="truncate pr-3">{milestone.label}</span>
      <span className="font-mono">
        {hasAmount ? (
          <>
            {formatEur(milestone.amountCents)}{" "}
            <span className="text-slate-400">
              (-{formatEur(fee)})
            </span>
          </>
        ) : (
          <span className="text-slate-400">&mdash;</span>
        )}
      </span>
    </li>
  )
}

function MilestoneTotals({ totalFees }: { totalFees: number }) {
  return (
    <div className="flex items-center justify-between border-t border-slate-100 pt-3 text-sm dark:border-slate-700">
      <span className="font-medium text-slate-700 dark:text-slate-300">
        Total frais plateforme
      </span>
      <span className="font-mono font-semibold text-slate-900 dark:text-white">
        {formatEur(totalFees)}
      </span>
    </div>
  )
}

function FeePreviewSkeleton() {
  return (
    <div className="space-y-3" aria-hidden="true">
      <div className="h-10 animate-shimmer rounded-xl bg-slate-200 dark:bg-slate-700" />
      <div className="h-10 animate-shimmer rounded-xl bg-slate-200 dark:bg-slate-700" />
      <div className="h-10 animate-shimmer rounded-xl bg-slate-200 dark:bg-slate-700" />
      <div className="h-8 animate-shimmer rounded-lg bg-slate-200 dark:bg-slate-700" />
    </div>
  )
}

function pickReferenceAmount(milestones: Milestone[]): number {
  for (const m of milestones) {
    if (m.amountCents > 0) return m.amountCents
  }
  return 0
}

function formatEur(centimes: number): string {
  return new Intl.NumberFormat("fr-FR", {
    style: "currency",
    currency: "EUR",
  }).format(centimes / 100)
}
