"use client"

import { useTranslations } from "next-intl"
import { Flag } from "lucide-react"

/**
 * ReportButton — placeholder UI affordance for the DSA art. 16
 * reporting mechanism. Surfaces a `mailto:` link with a pre-filled
 * subject and body so visitors can flag content without having to
 * leave the current page.
 *
 * This component is intentionally minimal: it renders an anchor,
 * not a feature-specific modal. A future backend agent will wire
 * the underlying `/api/v1/dsa/report` endpoint and replace the
 * mailto fallback with a proper signed-in flow (with attachment
 * upload, deduplication and audit-log entry). Until then, the
 * mailto channel ensures the legal obligation (DSA art. 16) is
 * already discoverable from every profile, message and proposal
 * card.
 *
 * The component accepts a `resourceType` + `resourceId` pair so the
 * pre-filled email body identifies the reported resource without
 * leaking sensitive data. Both fields are surfaced as plain text;
 * no PII or auth token is interpolated.
 */
export type ReportableResourceType =
  | "profile"
  | "message"
  | "proposal"
  | "mission"
  | "review"

interface ReportButtonProps {
  resourceType: ReportableResourceType
  resourceId: string
  /** Optional extra context — e.g. message preview, mission title. */
  context?: string
  className?: string
}

const DEFAULT_CLASSES =
  "inline-flex items-center gap-1.5 rounded-full border border-border bg-card px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-accent"

export function ReportButton({
  resourceType,
  resourceId,
  context,
  className,
}: ReportButtonProps) {
  const t = useTranslations("legal.reportButton")

  const subject = encodeURIComponent(
    t("emailSubject", { resourceType, resourceId }),
  )
  const bodyPlain = [
    t("emailBodyIntro"),
    "",
    `${t("emailFieldResourceType")}: ${resourceType}`,
    `${t("emailFieldResourceId")}: ${resourceId}`,
    context ? `${t("emailFieldContext")}: ${context}` : null,
    "",
    t("emailFieldReason"),
    "",
    t("emailBodyOutro"),
  ]
    .filter(Boolean)
    .join("\n")
  const body = encodeURIComponent(bodyPlain)

  return (
    <a
      href={`mailto:trust@designedtrust.com?subject=${subject}&body=${body}`}
      className={className ? `${DEFAULT_CLASSES} ${className}` : DEFAULT_CLASSES}
      aria-label={t("ariaLabel", { resourceType })}
    >
      <Flag className="size-3.5" aria-hidden />
      <span>{t("label")}</span>
    </a>
  )
}
