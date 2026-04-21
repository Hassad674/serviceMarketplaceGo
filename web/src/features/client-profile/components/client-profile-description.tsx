"use client"

import { useTranslations } from "next-intl"

interface ClientProfileDescriptionProps {
  description: string
}

// ClientProfileDescription is the read-only sibling of
// `ClientProfileEditor` — used on the public `/clients/[id]` page and
// whenever the current user lacks `org_client_profile.edit` on their
// own org. Empty descriptions render an inviting empty state rather
// than hiding the card entirely because visitors benefit from knowing
// the section exists even when unpopulated (social proof is thin
// otherwise).
export function ClientProfileDescription(props: ClientProfileDescriptionProps) {
  const { description } = props
  const t = useTranslations("clientProfile")

  return (
    <section
      className="bg-card border border-border rounded-2xl p-6 shadow-sm"
      aria-labelledby="client-profile-description-heading"
    >
      <h2
        id="client-profile-description-heading"
        className="text-lg font-semibold text-foreground"
      >
        {t("description")}
      </h2>
      {description.trim() === "" ? (
        <p className="mt-3 text-sm text-muted-foreground">
          {t("descriptionEmpty")}
        </p>
      ) : (
        <p className="mt-3 whitespace-pre-wrap break-words text-sm leading-relaxed text-foreground">
          {description}
        </p>
      )}
    </section>
  )
}
