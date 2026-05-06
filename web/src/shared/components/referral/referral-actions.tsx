"use client"

import { useState } from "react"
import { Loader2 } from "lucide-react"

import { useRespondToReferral } from "@/shared/hooks/referral/use-referral"
import type {
  Referral,
  ReferralActorRole,
  RespondReferralInput,
} from "@/shared/types/referral"
import { Button } from "@/shared/components/ui/button"

import { Input } from "@/shared/components/ui/input"
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
          className="rounded-lg border border-primary/30 bg-primary-soft px-4 py-3 text-sm text-primary-deep"
        >
          {error}
        </div>
      )}

      {showNegotiate ? (
        <div className="space-y-3 rounded-2xl border border-border bg-card p-4">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">
              Nouveau taux : {counterRate.toFixed(counterRate % 1 === 0 ? 0 : 1)} %
            </label>
            <Input
              type="range"
              min={0}
              max={30}
              step={0.5}
              value={counterRate}
              onChange={(e) => setCounterRate(parseFloat(e.target.value))}
              className="w-full accent-primary"
            />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">
              Message (optionnel)
            </label>
            <textarea
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              rows={2}
              className="w-full rounded-lg border border-border-strong px-3 py-2 text-sm focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
            />
          </div>
          <div className="flex gap-2">
            <Button variant="ghost" size="auto"
              type="button"
              onClick={() => setShowNegotiate(false)}
              className="rounded-lg border border-border-strong px-4 py-2 text-sm text-foreground hover:bg-muted"
            >
              Annuler
            </Button>
            <Button variant="ghost" size="auto"
              type="button"
              onClick={() =>
                send({ action: "negotiate", new_rate_pct: counterRate, message })
              }
              disabled={respond.isPending}
              className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:bg-primary-deep disabled:opacity-50"
            >
              {respond.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                "Envoyer la contre-proposition"
              )}
            </Button>
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
    primary: "bg-primary text-white hover:bg-primary-deep",
    secondary: "border border-border-strong text-foreground hover:bg-muted",
    danger: "border border-primary/30 text-primary-deep hover:bg-primary-soft",
  }
  return (
    <Button variant="ghost" size="auto"
      type="button"
      onClick={handlers[action.kind]}
      disabled={loading}
      className={`rounded-lg px-4 py-2 text-sm font-medium transition disabled:opacity-50 ${variantClasses[action.variant]}`}
    >
      {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : action.label}
    </Button>
  )
}
