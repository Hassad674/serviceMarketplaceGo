"use client"

import { useState } from "react"
import { Loader2 } from "lucide-react"

import { useRespondToReferral } from "../hooks/use-referrals"
import type { Referral, ReferralActorRole, RespondReferralInput } from "../types"

interface ReferralActionsProps {
  referral: Referral
  viewerRole: ReferralActorRole
}

// ReferralActions renders the action buttons available to the viewer based
// on the current referral status and their role. Centralised here so the
// detail page does not have to repeat the matrix in every viewer-specific
// view.
//
// State machine (mirrors the backend):
//   pending_provider — provider can accept/negotiate/reject; referrer can cancel
//   pending_referrer — referrer can accept/negotiate/reject; referrer can cancel
//   pending_client   — client can accept/reject; referrer can cancel
//   active           — referrer can terminate
//   terminal         — no actions
export function ReferralActions({ referral, viewerRole }: ReferralActionsProps) {
  const respond = useRespondToReferral(referral.id)
  const [showNegotiate, setShowNegotiate] = useState(false)
  const [counterRate, setCounterRate] = useState<number>(referral.rate_pct ?? 5)
  const [message, setMessage] = useState("")
  const [error, setError] = useState<string | null>(null)

  async function send(input: RespondReferralInput) {
    setError(null)
    try {
      await respond.mutateAsync(input)
      setShowNegotiate(false)
      setMessage("")
    } catch (e) {
      setError(e instanceof Error ? e.message : "Action impossible")
    }
  }

  // Compute available actions for this (status, viewerRole) pair.
  const actions = computeActions(referral.status, viewerRole)
  if (actions.length === 0) {
    return null
  }

  return (
    <div className="space-y-4">
      {error && (
        <div
          role="alert"
          className="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700"
        >
          {error}
        </div>
      )}

      {showNegotiate ? (
        <div className="space-y-3 rounded-2xl border border-slate-200 bg-white p-4">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-slate-700">
              Nouveau taux : {counterRate.toFixed(counterRate % 1 === 0 ? 0 : 1)} %
            </label>
            <input
              type="range"
              min={0}
              max={30}
              step={0.5}
              value={counterRate}
              onChange={(e) => setCounterRate(parseFloat(e.target.value))}
              className="w-full accent-rose-500"
            />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-slate-700">
              Message (optionnel)
            </label>
            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              rows={2}
              className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-100"
            />
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setShowNegotiate(false)}
              className="rounded-lg border border-slate-300 px-4 py-2 text-sm text-slate-700 hover:bg-slate-50"
            >
              Annuler
            </button>
            <button
              type="button"
              onClick={() =>
                send({ action: "negotiate", new_rate_pct: counterRate, message })
              }
              disabled={respond.isPending}
              className="rounded-lg bg-rose-500 px-4 py-2 text-sm font-medium text-white hover:bg-rose-600 disabled:opacity-50"
            >
              {respond.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                "Envoyer la contre-proposition"
              )}
            </button>
          </div>
        </div>
      ) : (
        <div className="flex flex-wrap gap-2">
          {actions.map((action) => (
            <ActionButton
              key={action.kind}
              action={action}
              loading={respond.isPending}
              onAccept={() => send({ action: "accept" })}
              onReject={() => send({ action: "reject", message })}
              onCancel={() => send({ action: "cancel" })}
              onTerminate={() => send({ action: "terminate" })}
              onNegotiate={() => setShowNegotiate(true)}
            />
          ))}
        </div>
      )}
    </div>
  )
}

type ActionKind = "accept" | "reject" | "negotiate" | "cancel" | "terminate"
interface UIAction {
  kind: ActionKind
  label: string
  variant: "primary" | "secondary" | "danger"
}

function computeActions(
  status: Referral["status"],
  role: ReferralActorRole,
): UIAction[] {
  switch (status) {
    case "pending_provider":
      if (role === "provider") {
        return [
          { kind: "accept", label: "Accepter", variant: "primary" },
          { kind: "negotiate", label: "Négocier", variant: "secondary" },
          { kind: "reject", label: "Refuser", variant: "danger" },
        ]
      }
      if (role === "referrer") {
        return [{ kind: "cancel", label: "Annuler l'intro", variant: "danger" }]
      }
      return []

    case "pending_referrer":
      if (role === "referrer") {
        return [
          { kind: "accept", label: "Accepter le contre", variant: "primary" },
          { kind: "negotiate", label: "Contre-contrer", variant: "secondary" },
          { kind: "reject", label: "Refuser", variant: "danger" },
        ]
      }
      return []

    case "pending_client":
      if (role === "client") {
        return [
          { kind: "accept", label: "Accepter la mise en relation", variant: "primary" },
          { kind: "reject", label: "Refuser", variant: "danger" },
        ]
      }
      if (role === "referrer") {
        return [{ kind: "cancel", label: "Annuler l'intro", variant: "danger" }]
      }
      return []

    case "active":
      if (role === "referrer") {
        return [{ kind: "terminate", label: "Terminer l'intro", variant: "danger" }]
      }
      return []

    default:
      return []
  }
}

interface ActionButtonProps {
  action: UIAction
  loading: boolean
  onAccept: () => void
  onReject: () => void
  onCancel: () => void
  onTerminate: () => void
  onNegotiate: () => void
}

function ActionButton({
  action,
  loading,
  onAccept,
  onReject,
  onCancel,
  onTerminate,
  onNegotiate,
}: ActionButtonProps) {
  const handlers: Record<ActionKind, () => void> = {
    accept: onAccept,
    reject: onReject,
    cancel: onCancel,
    terminate: onTerminate,
    negotiate: onNegotiate,
  }
  const variantClasses: Record<UIAction["variant"], string> = {
    primary: "bg-rose-500 text-white hover:bg-rose-600",
    secondary: "border border-slate-300 text-slate-700 hover:bg-slate-50",
    danger: "border border-rose-200 text-rose-700 hover:bg-rose-50",
  }
  return (
    <button
      type="button"
      onClick={handlers[action.kind]}
      disabled={loading}
      className={`rounded-lg px-4 py-2 text-sm font-medium transition disabled:opacity-50 ${variantClasses[action.variant]}`}
    >
      {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : action.label}
    </button>
  )
}
