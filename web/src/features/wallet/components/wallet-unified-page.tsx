"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { toast } from "sonner"

import { ApiError } from "@/shared/lib/api-client"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { BillingProfileCompletionModal } from "@/shared/components/billing-profile/billing-profile-completion-modal"
import { useBillingProfileCompleteness } from "@/shared/hooks/billing-profile/use-billing-profile-completeness"
import type { MissingField } from "@/shared/types/billing-profile"

import {
  useWalletSummary,
  useWalletWithdraw,
} from "../hooks/use-wallet"
import type { WithdrawLegError } from "../api/wallet-api"
import {
  extractMessage,
  extractMissingFields,
} from "./wallet-payout-section"
import { CommissionKYCRequiredModal } from "./commission-kyc-required-modal"
import { WalletUnifiedHeader } from "./wallet-unified-header"
import { WalletUnifiedHistory } from "./wallet-unified-history"
import { WalletWithdrawResultModal } from "./wallet-withdraw-result-modal"

/**
 * WalletUnifiedPage — orchestrator for the WALLET-UNIFY Run C
 * refonte. Consumes the unified GET /wallet/summary and wires the
 * single POST /wallet/withdraw flow through the existing KYC +
 * billing-profile gating modals (D1+D2).
 *
 * Flow:
 *   1. Hero shows total / available / escrowed / transmitted (from
 *      breakdown summed by backend + duplicated at the top level).
 *   2. Click "Retirer" → useWalletWithdraw mutation.
 *   3. Branches on result:
 *      - 200, no errors → success toast.
 *      - 207, errors[] populated → partial-result modal opens with
 *        the per-leg failure detail.
 *      - 422 kyc_required ApiError → CommissionKYCRequiredModal with
 *        the onboarding URL pulled off the error body.
 *      - 403 billing_profile_incomplete ApiError →
 *        BillingProfileCompletionModal with the missing fields.
 *
 * The orchestration mirrors `WalletPayoutSection` (legacy mission
 * payout) but adapted to the unified envelope. Both flows can
 * coexist — this page wires the new one for the /wallet route.
 */
export function WalletUnifiedPage() {
  const t = useTranslations("walletUnified")
  const { data: summary, isLoading, isError } = useWalletSummary()

  const withdrawMutation = useWalletWithdraw()
  const canWithdraw = useHasPermission("wallet.withdraw")
  const completeness = useBillingProfileCompleteness()

  const [completionModalOpen, setCompletionModalOpen] = useState(false)
  const [serverMissingFields, setServerMissingFields] = useState<
    MissingField[] | null
  >(null)
  const [kycModalOpen, setKycModalOpen] = useState(false)
  const [kycOnboardingURL, setKycOnboardingURL] = useState<string | undefined>(
    undefined,
  )
  const [resultModalOpen, setResultModalOpen] = useState(false)
  const [resultErrors, setResultErrors] = useState<WithdrawLegError[]>([])
  const [resultBreakdown, setResultBreakdown] = useState<{
    drained: number
    missions: number
    commissions: number
  } | null>(null)

  if (isLoading) {
    return <WalletUnifiedSkeleton />
  }
  if (isError || !summary) {
    return (
      <div className="mx-auto max-w-[1100px] px-4 py-8 sm:px-6">
        <p className="text-center text-[14px] text-muted-foreground">
          {t("history.empty")}
        </p>
      </div>
    )
  }

  function handleWithdraw() {
    if (!canWithdraw) return
    // Billing-profile pre-flight (mirrors WalletPayoutSection).
    if (!completeness.isLoading && !completeness.isComplete) {
      setServerMissingFields(null)
      setCompletionModalOpen(true)
      return
    }
    withdrawMutation.mutate(undefined, {
      onSuccess: (result) => {
        // Defensive consumption — the backend envelope can omit
        // any of these fields. Never destructure `result.errors`
        // directly: when no leg failed the JSON omits the key
        // entirely (`omitempty` in `wallet_withdraw.go`), causing a
        // hard runtime crash on `undefined.length`. This was bug
        // `fix/wallet-ui-crash-and-aggregation`.
        const errors = result?.errors ?? []
        const drained = result?.drained_cents ?? 0
        const missions = result?.missions_cents ?? 0
        const commissions = result?.commissions_cents ?? 0
        if (errors.length > 0 && drained > 0) {
          // Partial success (207) — open the breakdown modal.
          setResultErrors(errors)
          setResultBreakdown({
            drained,
            missions,
            commissions,
          })
          setResultModalOpen(true)
          toast(t("toast.partial"))
        } else if (drained > 0) {
          toast(t("toast.success"))
        } else {
          // Unknown shape OR a backend that responded 200 with
          // zero drained AND no errors — both extremely unlikely,
          // but a defensive fallback keeps the user out of a dead
          // state. Surface a generic "retrait en cours" notice and
          // let the cache invalidation reflect the real state.
          toast(t("toast.unknown"))
        }
      },
      onError: (err) => {
        if (!(err instanceof ApiError)) return
        if (err.status === 422 && err.code === "kyc_required") {
          setKycOnboardingURL(extractOnboardingURL(err.body))
          setKycModalOpen(true)
          return
        }
        if (err.status === 403 && err.code === "billing_profile_incomplete") {
          const missing = extractMissingFields(err.body)
          setServerMissingFields(
            missing.length > 0 ? missing : completeness.missingFields,
          )
          setCompletionModalOpen(true)
          return
        }
        // Anything else surfaces a generic error toast — the user
        // can retry. Logging happens via the mutation cache.
        toast.error(extractMessage(err.body) ?? err.message)
      },
    })
  }

  const fieldsForModal = serverMissingFields ?? completeness.missingFields

  return (
    <div className="mx-auto w-full max-w-[1100px] space-y-6 px-4 py-8 sm:px-6">
      <WalletUnifiedHeader
        totalCents={summary.total_cents}
        escrowedCents={summary.escrowed_cents}
        availableCents={summary.available_cents}
        transmittedCents={summary.transmitted_cents}
        payoutPending={withdrawMutation.isPending}
        onWithdraw={handleWithdraw}
      />
      <WalletUnifiedHistory />
      <BillingProfileCompletionModal
        open={completionModalOpen}
        onClose={() => setCompletionModalOpen(false)}
        missingFields={fieldsForModal}
        returnTo="/wallet"
      />
      <CommissionKYCRequiredModal
        open={kycModalOpen}
        onClose={() => setKycModalOpen(false)}
        onboardingURL={kycOnboardingURL}
      />
      {resultBreakdown && (
        <WalletWithdrawResultModal
          open={resultModalOpen}
          onClose={() => setResultModalOpen(false)}
          drainedCents={resultBreakdown.drained}
          missionsCents={resultBreakdown.missions}
          commissionsCents={resultBreakdown.commissions}
          errors={resultErrors}
        />
      )}
    </div>
  )
}

// extractOnboardingURL reads the optional `onboarding_url` off the
// 422 envelope body. Falls back to undefined so the modal renders
// the in-app /payment-info link instead.
function extractOnboardingURL(body: unknown): string | undefined {
  if (!body || typeof body !== "object") return undefined
  const candidate = (body as { onboarding_url?: unknown }).onboarding_url
  if (typeof candidate === "string" && candidate.length > 0) return candidate
  return undefined
}

function WalletUnifiedSkeleton() {
  return (
    <div className="mx-auto w-full max-w-[1100px] space-y-6 px-4 py-8 sm:px-6">
      <div className="h-44 animate-shimmer rounded-2xl bg-border" />
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        {[0, 1, 2].map((i) => (
          <div
            key={i}
            className="h-24 animate-shimmer rounded-2xl bg-border"
          />
        ))}
      </div>
      <div className="h-64 animate-shimmer rounded-2xl bg-border" />
    </div>
  )
}
