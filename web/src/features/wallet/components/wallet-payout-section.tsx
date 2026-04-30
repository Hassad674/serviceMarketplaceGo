"use client"

import { useState } from "react"
import { ApiError } from "@/shared/lib/api-client"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { BillingProfileCompletionModal } from "@/features/invoicing/components/billing-profile-completion-modal"
import { useBillingProfileCompleteness } from "@/features/invoicing/hooks/use-billing-profile-completeness"
import type { MissingField } from "@/features/invoicing/types"
import { useRequestPayout } from "../hooks/use-wallet"
import { KYCIncompleteModal } from "./kyc-incomplete-modal"
import { WalletOverviewCard } from "./wallet-overview-card"

// Owns the imperative "Request payout" flow — wraps the hero card
// in modal-aware logic. Pre-flight order MIRRORS the backend:
//   1. KYC gate (payouts_enabled flag in cached snapshot, then
//      defensive 403 `kyc_incomplete` from the response envelope).
//   2. Billing-profile gate (cached completeness, then defensive
//      403 `billing_profile_incomplete`).
// The two 403 codes are mutually exclusive on the API contract,
// so the frontend gates are independent state machines.

export interface WalletPayoutSectionProps {
  totalEarned: number
  available: number
  stripeAccountId: string
  payoutsEnabled: boolean
}

export function WalletPayoutSection({
  totalEarned,
  available,
  stripeAccountId,
  payoutsEnabled,
}: WalletPayoutSectionProps) {
  const payoutMutation = useRequestPayout()
  const canWithdraw = useHasPermission("wallet.withdraw")
  const [payoutMessage, setPayoutMessage] = useState("")

  const completeness = useBillingProfileCompleteness()
  const [completionModalOpen, setCompletionModalOpen] = useState(false)
  const [serverMissingFields, setServerMissingFields] = useState<
    MissingField[] | null
  >(null)

  const [kycModalOpen, setKycModalOpen] = useState(false)
  const [kycModalMessage, setKycModalMessage] = useState<string | undefined>(
    undefined,
  )

  function handlePayout() {
    // KYC pre-flight: the wallet payload already exposes
    // `payouts_enabled`, so when we know KYC is not done we open the
    // KYC modal up front and skip the request entirely.
    if (!payoutsEnabled) {
      setKycModalMessage(undefined)
      setKycModalOpen(true)
      return
    }

    // Billing-profile pre-flight: open the completion modal directly
    // when we already know the profile is incomplete.
    if (!completeness.isLoading && !completeness.isComplete) {
      setServerMissingFields(null)
      setCompletionModalOpen(true)
      return
    }

    payoutMutation.mutate(undefined, {
      onSuccess: (result) => setPayoutMessage(result.message),
      onError: (err) => {
        if (err instanceof ApiError && err.status === 403) {
          if (err.code === "kyc_incomplete") {
            // Defensive path — the cached `payouts_enabled` flag may
            // be stale across tabs.
            setKycModalMessage(extractMessage(err.body))
            setKycModalOpen(true)
            return
          }
          if (err.code === "billing_profile_incomplete") {
            const missing = extractMissingFields(err.body)
            setServerMissingFields(
              missing.length > 0 ? missing : completeness.missingFields,
            )
            setCompletionModalOpen(true)
            return
          }
        }
      },
    })
  }

  const fieldsForModal = serverMissingFields ?? completeness.missingFields

  return (
    <>
      <WalletOverviewCard
        totalEarned={totalEarned}
        available={available}
        stripeAccountId={stripeAccountId}
        payoutsEnabled={payoutsEnabled}
        canWithdraw={canWithdraw}
        payoutPending={payoutMutation.isPending}
        payoutMessage={payoutMessage}
        payoutError={
          payoutMutation.isError ? (payoutMutation.error as Error) : null
        }
        onPayout={handlePayout}
      />
      <BillingProfileCompletionModal
        open={completionModalOpen}
        onClose={() => setCompletionModalOpen(false)}
        missingFields={fieldsForModal}
        returnTo="/wallet"
      />
      <KYCIncompleteModal
        open={kycModalOpen}
        onClose={() => setKycModalOpen(false)}
        message={kycModalMessage}
      />
    </>
  )
}

// Reads the `error.message` from a parsed error envelope. Returns
// undefined when the body is missing or malformed so the consuming
// modal can fall back to its own default copy.
export function extractMessage(body: unknown): string | undefined {
  if (!body || typeof body !== "object") return undefined
  const errField = (body as { error?: unknown }).error
  if (errField && typeof errField === "object") {
    const msg = (errField as { message?: unknown }).message
    if (typeof msg === "string" && msg.length > 0) return msg
  }
  const topLevel = (body as { message?: unknown }).message
  if (typeof topLevel === "string" && topLevel.length > 0) return topLevel
  return undefined
}

// Reads the `missing_fields` array off a parsed error envelope.
// Returns an empty list when the body is missing or malformed —
// the modal falls back to the cached snapshot in that case.
export function extractMissingFields(body: unknown): MissingField[] {
  if (!body || typeof body !== "object") return []
  const candidate = (body as { missing_fields?: unknown }).missing_fields
  if (!Array.isArray(candidate)) return []
  const out: MissingField[] = []
  for (const item of candidate) {
    if (
      item &&
      typeof item === "object" &&
      typeof (item as { field?: unknown }).field === "string" &&
      typeof (item as { reason?: unknown }).reason === "string"
    ) {
      out.push({
        field: (item as { field: string }).field,
        reason: (item as { reason: string }).reason,
      })
    }
  }
  return out
}
