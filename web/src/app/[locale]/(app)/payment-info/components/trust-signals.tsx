"use client"

import { Lock, ShieldCheck, Sparkles } from "lucide-react"

const SIGNALS = [
  {
    icon: Lock,
    label: "Chiffrement TLS",
    detail: "Vos données transitent sur un canal chiffré",
  },
  {
    icon: ShieldCheck,
    label: "RGPD conforme",
    detail: "Hébergement Union Européenne",
  },
  {
    icon: Sparkles,
    label: "Certifié PCI-DSS",
    detail: "Niveau 1 — le plus haut standard",
  },
]

export function TrustSignals() {
  return (
    <ul className="grid grid-cols-1 gap-2 sm:grid-cols-3">
      {SIGNALS.map((signal) => {
        const Icon = signal.icon
        return (
          <li
            key={signal.label}
            className="flex items-start gap-2.5 rounded-lg border border-slate-100 bg-white px-3 py-2.5"
          >
            <span
              className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-rose-50 to-rose-100 text-rose-600"
              aria-hidden
            >
              <Icon className="h-3.5 w-3.5" />
            </span>
            <div className="min-w-0">
              <div className="text-[12px] font-semibold text-slate-900">{signal.label}</div>
              <div className="text-[11px] leading-tight text-slate-500">{signal.detail}</div>
            </div>
          </li>
        )
      })}
    </ul>
  )
}
