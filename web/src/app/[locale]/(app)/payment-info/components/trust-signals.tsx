"use client"

import { Lock, ShieldCheck, Sparkles } from "lucide-react"
import { useTranslations } from "next-intl"

const SIGNAL_KEYS = [
  { icon: Lock, labelKey: "tlsEncrypted", detailKey: "tlsDetail" },
  { icon: ShieldCheck, labelKey: "gdprCompliant", detailKey: "gdprDetail" },
  { icon: Sparkles, labelKey: "pciCertified", detailKey: "pciDetail" },
] as const

export function TrustSignals() {
  const t = useTranslations("paymentInfo")
  return (
    <ul className="grid grid-cols-1 gap-2 sm:grid-cols-3">
      {SIGNAL_KEYS.map((signal) => {
        const Icon = signal.icon
        return (
          <li
            key={signal.labelKey}
            className="flex items-start gap-2.5 rounded-xl border border-border bg-card px-3 py-2.5"
          >
            <span
              className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-primary-soft text-primary"
              aria-hidden
            >
              <Icon className="h-3.5 w-3.5" />
            </span>
            <div className="min-w-0">
              <div className="text-[12px] font-semibold text-foreground">
                {t(signal.labelKey)}
              </div>
              <div className="text-[11px] leading-snug text-muted-foreground">
                {t(signal.detailKey)}
              </div>
            </div>
          </li>
        )
      })}
    </ul>
  )
}
