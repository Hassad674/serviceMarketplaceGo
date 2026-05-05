"use client"

import { useTranslations } from "next-intl"
import { isExpertiseDomainKey } from "@/shared/lib/profile/expertise"

interface ExpertiseDisplayProps {
  domains: string[]
}

// ExpertiseDisplay renders the read-only pill list of an org's
// declared expertise domains. Used on both /profile (owner view) and
// the public profile routes. Hidden entirely when the list is empty
// so consumers don't have to guard with another null-check.
export function ExpertiseDisplay({ domains }: ExpertiseDisplayProps) {
  const t = useTranslations("profile.expertise")
  const tDomains = useTranslations("profile.expertise.domains")
  const valid = domains.filter(isExpertiseDomainKey)
  if (valid.length === 0) return null

  return (
    <section
      aria-labelledby="expertise-display-title"
      className="bg-card border border-border rounded-2xl p-7 shadow-[var(--shadow-card)]"
    >
      <h2
        id="expertise-display-title"
        className="font-serif text-xl font-medium tracking-[-0.005em] text-foreground mb-4"
      >
        {t("sectionTitle")}
      </h2>
      <ul className="flex flex-wrap gap-1.5" aria-label={t("sectionTitle")}>
        {valid.map((key) => (
          <li key={key}>
            <span className="inline-flex items-center rounded-full border border-primary bg-primary-soft px-3.5 py-1.5 text-[13px] font-semibold text-primary-deep">
              {tDomains(key)}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
