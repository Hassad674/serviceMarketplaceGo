import { Handshake } from "lucide-react"

interface ReferralSystemMessageProps {
  // The message content as posted by the backend (e.g.
  // "🤝 Mise en relation activée"). Optional metadata is intentionally
  // ignored for V1 — the activation message is self-explanatory.
  content: string
}

// ReferralSystemMessage renders the single referral-related system message
// type that the backend posts in the provider↔client conversation when an
// intro is activated. Visually it sits between provider and client messages
// as a centred chip so the conversation timeline reads naturally.
//
// Note: this is the ONLY referral system message — commission events
// (paid / clawed back) are notification-only and never appear in the
// working conversation, by design (B2B confidentiality).
export function ReferralSystemMessage({ content }: ReferralSystemMessageProps) {
  return (
    <div className="my-4 flex items-center justify-center">
      <div className="inline-flex items-center gap-2 rounded-full border border-rose-200 bg-rose-50 px-4 py-1.5 text-xs font-medium text-rose-700">
        <Handshake className="h-3.5 w-3.5" aria-hidden="true" />
        {content || "Mise en relation activée"}
      </div>
    </div>
  )
}
