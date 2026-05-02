"use client"

import Link from "next/link"
import { ArrowRight, Handshake, Hourglass, MessageSquareWarning, XCircle } from "lucide-react"

import { useReferral } from "@/shared/hooks/referral/use-referral"
import {
  formatRatePct,
  type ReferralActorRole,
  type ReferralStatus,
} from "@/shared/types/referral"
import { ReferralActions } from "@/shared/components/referral/referral-actions"

// ReferralSystemMessageMetadata mirrors the JSON payload posted by the
// Go backend in service_messages.go. Fields may be absent when the
// message was created by an older version — components read defensively.
export interface ReferralSystemMessageMetadata {
  referral_id?: string
  new_status?: ReferralStatus
  prev_status?: ReferralStatus
  rate_pct?: number
  referrer_id?: string
  provider_id?: string
  client_id?: string
}

// messageTone defines the visual treatment per (type, new_status) pair.
// The headline + icon hint at the event; the action area below is driven
// by the live referral state to keep buttons in sync with the authoritative
// backend status (a stale rejected intro must not show an Accept button).
function messageTone(type: string, status?: ReferralStatus): {
  icon: typeof Handshake
  accent: string
  headline: string
} {
  if (type === "referral_intro_negotiated") {
    return {
      icon: MessageSquareWarning,
      accent: "border-amber-200 bg-amber-50 text-amber-800",
      headline: "Contre-proposition de taux",
    }
  }
  if (type === "referral_intro_activated" || status === "active") {
    return {
      icon: Handshake,
      accent: "border-emerald-200 bg-emerald-50 text-emerald-800",
      headline: "Mise en relation activée",
    }
  }
  if (type === "referral_intro_closed") {
    if (status === "expired") {
      return {
        icon: Hourglass,
        accent: "border-slate-200 bg-slate-50 text-slate-700",
        headline: "Mise en relation expirée",
      }
    }
    return {
      icon: XCircle,
      accent: "border-slate-200 bg-slate-50 text-slate-700",
      headline: "Mise en relation clôturée",
    }
  }
  return {
    icon: Handshake,
    accent: "border-rose-200 bg-rose-50 text-rose-800",
    headline: "Nouvelle proposition d'apport d'affaires",
  }
}

function viewerRoleFor(
  metadata: ReferralSystemMessageMetadata,
  currentUserId: string,
): ReferralActorRole | null {
  if (metadata.referrer_id === currentUserId) return "referrer"
  if (metadata.provider_id === currentUserId) return "provider"
  if (metadata.client_id === currentUserId) return "client"
  return null
}

interface ReferralSystemMessageProps {
  type: string
  metadata: ReferralSystemMessageMetadata
  content: string
  currentUserId: string
}

// ReferralSystemMessage renders an interactive card for every referral
// lifecycle event posted into a conversation. The live referral is
// fetched so the action buttons always reflect the authoritative status
// (no race with someone else accepting from another device).
//
// Fallback: when metadata is missing (e.g. old messages from before
// Phase B) the component degrades to a read-only chip matching the old
// UI — never breaks a legacy conversation timeline.
export function ReferralSystemMessage({
  type,
  metadata,
  content,
  currentUserId,
}: ReferralSystemMessageProps) {
  const referralId = metadata.referral_id
  const { data: referral } = useReferral(referralId)

  // Legacy fallback: no metadata → render the old chip.
  if (!referralId) {
    return (
      <div className="my-4 flex items-center justify-center">
        <div className="inline-flex items-center gap-2 rounded-full border border-rose-200 bg-rose-50 px-4 py-1.5 text-xs font-medium text-rose-700">
          <Handshake className="h-3.5 w-3.5" aria-hidden="true" />
          {content || "Mise en relation activée"}
        </div>
      </div>
    )
  }

  // Prefer live status (defensive against stale metadata after retries).
  const status: ReferralStatus | undefined =
    referral?.status ?? metadata.new_status
  const { icon: Icon, accent, headline } = messageTone(type, status)
  const viewerRole = viewerRoleFor(metadata, currentUserId)
  const canShowRate = metadata.rate_pct !== undefined && metadata.rate_pct !== null

  return (
    <div className="my-4 flex w-full justify-center">
      <div className={`w-full max-w-md rounded-2xl border p-4 shadow-sm ${accent}`}>
        <div className="flex items-start gap-3">
          <div className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-white/80">
            <Icon className="h-4 w-4" aria-hidden="true" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="text-sm font-semibold">{headline}</div>
            {content && (
              <p className="mt-0.5 text-xs opacity-80">{content}</p>
            )}
            {canShowRate && (
              <div className="mt-2 inline-flex items-center gap-1.5 rounded-full bg-white/80 px-2.5 py-0.5 text-[11px] font-medium">
                Commission {formatRatePct(metadata.rate_pct)}
              </div>
            )}
          </div>
        </div>

        {referral && viewerRole && (
          <div className="mt-3">
            <ReferralActions referral={referral} viewerRole={viewerRole} />
          </div>
        )}

        <div className="mt-3 flex items-center justify-end">
          <Link
            href={`/referrals/${referralId}`}
            className="inline-flex items-center gap-1 text-xs font-medium opacity-80 hover:opacity-100"
          >
            Voir le détail
            <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
          </Link>
        </div>
      </div>
    </div>
  )
}
