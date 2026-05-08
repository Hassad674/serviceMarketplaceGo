/**
 * breadcrumbs.ts — JSON-LD `BreadcrumbList` builder for public pages.
 *
 * Google uses BreadcrumbList structured data to display the breadcrumb
 * trail in the SERP, replacing the raw URL. This dramatically improves
 * CTR on B2B-style queries where the path encodes useful context
 * ("Marketplace > Freelances > Alice Martin").
 *
 * Reference: https://schema.org/BreadcrumbList
 *
 * The helper accepts a list of crumbs already built by the caller —
 * the caller knows the localized labels (via `useTranslations`) and
 * the absolute URLs (via `absoluteUrl`). This separation keeps the
 * helper i18n-agnostic and trivially testable.
 */

export interface BreadcrumbCrumb {
  /** Human-readable label, already translated. */
  name: string
  /** Absolute URL. Omit for the final crumb (current page). */
  item?: string
}

/**
 * buildBreadcrumbList returns a JSON-LD `BreadcrumbList` payload.
 * Pass the result through `safeJsonLd()` before embedding in the DOM.
 */
export function buildBreadcrumbList(
  crumbs: BreadcrumbCrumb[],
): Record<string, unknown> {
  return {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: crumbs.map((crumb, index) => {
      const element: Record<string, unknown> = {
        "@type": "ListItem",
        position: index + 1,
        name: crumb.name,
      }
      if (crumb.item) {
        element.item = crumb.item
      }
      return element
    }),
  }
}
