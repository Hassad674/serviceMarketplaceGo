"use client"

import { useTranslations } from "next-intl"
import {
  isExpertiseDomainKey,
  orgTypeSupportsExpertise,
} from "../constants/expertise"

interface ExpertiseDisplayProps {
  domains: string[] | undefined
  orgType: string | undefined
}

// Read-only pill list for public profiles (/freelancers/[id],
// /agencies/[id]). Hidden entirely when:
//   - the org type doesn't support expertise (enterprise), or
//   - there are no selected domains (nothing meaningful to show).
//
// Marked "use client" purely so it can consume the shared useTranslations
// hook — it has no state and could be rendered in a Server Component if
// we later rewire i18n access. The rendered output is fully static HTML.
export function ExpertiseDisplay({ domains, orgType }: ExpertiseDisplayProps) {
  const t = useTranslations("profile.expertise")
  const tDomains = useTranslations("profile.expertise.domains")

  if (!orgTypeSupportsExpertise(orgType)) return null

  const validDomains = (domains ?? []).filter(isExpertiseDomainKey)
  if (validDomains.length === 0) return null

  return (
    <section
      aria-labelledby="expertise-display-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <h2
        id="expertise-display-title"
        className="text-lg font-semibold text-foreground mb-3"
      >
        {t("sectionTitle")}
      </h2>
      <ul className="flex flex-wrap gap-2">
        {validDomains.map((key) => (
          <li key={key}>
            <span className="inline-flex items-center rounded-full bg-primary/10 text-primary px-3 py-1 text-sm font-medium border border-primary/20">
              {tDomains(key)}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
