/**
 * breadcrumb-nav.tsx — Soleil v2 breadcrumb navigation rendered at
 * the top of public profile pages.
 *
 * Visual breadcrumbs paired with the JSON-LD `BreadcrumbList` payload
 * (emitted from the page component via `safeJsonLd(buildBreadcrumbList())`)
 * give Google the structured + visual signal it needs to display the
 * breadcrumb trail in the SERP, replacing the raw URL with a friendly
 * path.
 *
 * The component is a thin Server Component — pure presentational, no
 * client JS. The caller passes already-translated labels and locale-
 * aware hrefs (built via next-intl `Link` href props).
 */

import { Fragment } from "react"
import { ChevronRight } from "lucide-react"
import { Link } from "@i18n/navigation"

export interface BreadcrumbNavCrumb {
  label: string
  /** Path relative to the locale prefix. Omit for the current page. */
  href?: string
}

export interface BreadcrumbNavProps {
  /** Localized accessible label, e.g. "Fil d'Ariane". */
  ariaLabel: string
  crumbs: BreadcrumbNavCrumb[]
}

export function BreadcrumbNav({ ariaLabel, crumbs }: BreadcrumbNavProps) {
  return (
    <nav aria-label={ariaLabel} className="text-xs text-muted-foreground">
      <ol className="flex flex-wrap items-center gap-1">
        {crumbs.map((crumb, index) => {
          const isLast = index === crumbs.length - 1
          return (
            <Fragment key={`${crumb.label}-${index}`}>
              <li className="flex items-center">
                {crumb.href && !isLast ? (
                  <Link
                    href={crumb.href}
                    className="transition-colors hover:text-foreground"
                  >
                    {crumb.label}
                  </Link>
                ) : (
                  <span
                    className="font-medium text-foreground"
                    aria-current={isLast ? "page" : undefined}
                  >
                    {crumb.label}
                  </span>
                )}
              </li>
              {!isLast ? (
                <li aria-hidden="true" className="flex items-center">
                  <ChevronRight className="h-3 w-3" />
                </li>
              ) : null}
            </Fragment>
          )
        })}
      </ol>
    </nav>
  )
}
