/**
 * alternates.ts — hreflang + canonical URL helpers for the public
 * profile and listing pages (PERF-W-08, top-tier SEO).
 *
 * The web app uses next-intl with `localePrefix: 'as-needed'`. As a
 * result, English (default locale) URLs are served WITHOUT the `/en`
 * prefix — `/freelancers/123` not `/en/freelancers/123`. French URLs
 * keep their `/fr` prefix.
 *
 * To stay perfectly explicit for Googlebot the `alternates.languages`
 * map declares both `en` and `fr` URLs (and `x-default` pointing at
 * the French canonical, which is the version of the app most users
 * land on first). The `canonical` is locale-specific so each indexed
 * page maps 1:1 to a single canonical URL.
 *
 * Why duplicate the URL with `/en` instead of leaving English bare?
 * `localePrefix: 'as-needed'` accepts BOTH `/freelancers/x` and
 * `/en/freelancers/x` — Next-intl rewrites the second to the first.
 * Declaring `/en/...` in hreflang is the explicit form Google
 * recommends, and avoids ambiguity for crawlers parsing the alternates
 * map (Search Console flags missing locale prefixes).
 */

import { siteConfig } from "@/config/site"

export type SupportedLocale = "fr" | "en"

export const SUPPORTED_LOCALES: SupportedLocale[] = ["fr", "en"]
export const DEFAULT_LOCALE: SupportedLocale = "fr"

/**
 * Resolve the absolute base URL once. `siteConfig.url` is env-driven
 * (NEXT_PUBLIC_APP_URL) so production hits the real origin and dev
 * falls back to localhost — matching the pattern set by sitemap.ts.
 */
function baseUrl(): string {
  return siteConfig.url.replace(/\/$/, "")
}

/**
 * Build the locale-prefixed path for a given locale + relative path.
 * Locale-aware so that `/freelancers/abc` becomes `/fr/freelancers/abc`
 * for French and `/en/freelancers/abc` for English (explicit, even
 * though Next-intl serves the un-prefixed English URL too).
 */
export function localizedPath(locale: SupportedLocale, path: string): string {
  const cleaned = path.startsWith("/") ? path : `/${path}`
  return `/${locale}${cleaned}`
}

export interface AlternatesInput {
  /** The current locale being rendered. */
  locale: SupportedLocale
  /** The path relative to the locale prefix, e.g. `/freelancers/abc`. */
  path: string
}

/**
 * buildAlternates returns the `alternates` block consumed by Next.js
 * `generateMetadata`. Includes the canonical URL plus a complete
 * `languages` map covering every supported locale and `x-default`.
 */
export function buildAlternates(input: AlternatesInput): {
  canonical: string
  languages: Record<string, string>
} {
  const base = baseUrl()
  const languages: Record<string, string> = {}
  for (const lang of SUPPORTED_LOCALES) {
    languages[lang] = `${base}${localizedPath(lang, input.path)}`
  }
  // x-default points at the French version since the marketplace's
  // primary audience is French. Crawlers serving users without a
  // strong locale preference (e.g. some Google indexes, Bing) fall
  // back to this URL.
  languages["x-default"] = `${base}${localizedPath(DEFAULT_LOCALE, input.path)}`

  return {
    canonical: `${base}${localizedPath(input.locale, input.path)}`,
    languages,
  }
}

/**
 * absoluteUrl converts a relative path into a fully-qualified URL on
 * the configured site origin. Useful for OG image URLs, JSON-LD `@id`
 * fields, and any other surface where Google requires absolute URLs.
 */
export function absoluteUrl(path: string): string {
  if (/^https?:\/\//i.test(path)) return path
  const base = baseUrl()
  return `${base}${path.startsWith("/") ? path : `/${path}`}`
}
