"use client"

import { useEffect } from "react"
import { useLocale, useTranslations } from "next-intl"
import * as CookieConsent from "vanilla-cookieconsent"

import "vanilla-cookieconsent/dist/cookieconsent.css"
import "@/styles/cookie-consent.css"

import { applyCustomConsent } from "@/shared/lib/posthog-consent"
import { legalHref } from "@i18n/routing"

/**
 * Mounts the vanilla-cookieconsent CMP and gates analytics opt-in on
 * the `analytics` category. Renders nothing — the CMP injects its own
 * modal/banner DOM into `document.body` on `run()`.
 *
 * Why this sits outside any feature folder: every public/auth/dash
 * page must surface the CMP, and the cookie banner straddles
 * analytics + legal concerns that don't belong to a single feature.
 *
 * Why we keep the legacy `applyCustomConsent()` glue: the GA4 provider
 * + the PostHog SDK glue still read the legacy localStorage flag (see
 * `shared/lib/posthog-consent.ts` header). Mirroring the CMP state into
 * that flag avoids touching every analytics consumer in this dispatch.
 */
export function CookieConsentProvider() {
  const t = useTranslations("cookieConsent")
  const locale = useLocale()

  useEffect(() => {
    // Run is idempotent — the library guards against double init via
    // an internal flag. We still wrap in a try/catch to never crash
    // the host app on a third-party hiccup.
    try {
      void CookieConsent.run({
        // RGPD-compliant default: nothing tracks before the user
        // makes a choice.
        mode: "opt-in",
        autoShow: true,
        // We surface our own logging via onChange/onFirstConsent —
        // the CMP cookie itself is sufficient persistence.
        revision: 1,
        // Disable the CMP's <script type="text/plain"> auto-runner.
        // Our analytics SDKs are ES module imports gated on the
        // category flag, not inline <script> tags, so this would be
        // a no-op anyway and can be safely turned off.
        manageScriptTags: false,
        guiOptions: {
          consentModal: {
            layout: "box",
            position: "bottom right",
            equalWeightButtons: false,
            flipButtons: false,
          },
          preferencesModal: {
            layout: "box",
            position: "right",
            equalWeightButtons: false,
            flipButtons: false,
          },
        },
        categories: {
          necessary: {
            enabled: true,
            readOnly: true,
          },
          analytics: {
            enabled: false,
            readOnly: false,
            autoClear: {
              cookies: [
                { name: /^_ga/ },
                { name: /^ph_/ },
                { name: "_gid" },
              ],
            },
          },
        },
        language: {
          default: locale,
          translations: {
            fr: buildTranslation(t, locale),
            en: buildTranslation(t, locale),
          },
        },
        onFirstConsent: ({ cookie }) => syncConsentToAnalytics(cookie.categories),
        onChange: ({ cookie }) => syncConsentToAnalytics(cookie.categories),
      })
    } catch {
      // best-effort — never block the app on a CMP boot failure
    }
    // Only re-init when the locale changes; the translations function is
    // recomputed alongside the locale via next-intl, so no need to add
    // `t` to the deps and trigger spurious reboots.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [locale])

  return null
}

/**
 * Translate the CMP modals from the same `cookieConsent.*` i18n
 * namespace used everywhere else in the app. Pulled out so the
 * (heavy) literal table is not inlined in the effect.
 *
 * The `footer` field is rendered as raw HTML by vanilla-cookieconsent,
 * so we use `t.markup()` (next-intl's HTML-string variant of `t.rich()`)
 * to inflate the `<privacy>`, `<cookies>`, `<notices>`, `<subprocessors>`
 * ICU tags into proper `<a>` anchors. Embedding raw `<a href="...">`
 * directly in the i18n string is rejected by next-intl 4.x (the ICU
 * parser raises `INVALID_TAG` on attribute-bearing tags).
 */
function buildTranslation(
  t: ReturnType<typeof useTranslations<"cookieConsent">>,
  locale: string,
): CookieConsent.Translation {
  return {
    consentModal: {
      title: t("banner.title"),
      description: t("banner.description"),
      acceptAllBtn: t("banner.acceptAll"),
      acceptNecessaryBtn: t("banner.refuseAll"),
      showPreferencesBtn: t("banner.preferences"),
      footer: buildFooterMarkup(t, locale),
    },
    preferencesModal: {
      title: t("preferences.title"),
      acceptAllBtn: t("banner.acceptAll"),
      acceptNecessaryBtn: t("banner.refuseAll"),
      savePreferencesBtn: t("preferences.save"),
      closeIconLabel: t("preferences.close"),
      sections: [
        {
          title: t("preferences.intro.title"),
          description: t("preferences.intro.description"),
        },
        {
          title: t("preferences.necessary.title"),
          description: t("preferences.necessary.description"),
          linkedCategory: "necessary",
        },
        {
          title: t("preferences.analytics.title"),
          description: t("preferences.analytics.description"),
          linkedCategory: "analytics",
        },
      ],
    },
  }
}

/**
 * Mirror the CMP's chosen categories into the legacy analytics
 * surface (PostHog opt-in flag + GA4 conditional mount + audit log
 * receipt). Keeps a single localStorage flag in sync with the CMP
 * cookie so legacy consumers don't need to learn the CMP API.
 */
function syncConsentToAnalytics(categories: string[]): void {
  const analyticsAccepted = categories.includes("analytics")
  applyCustomConsent(analyticsAccepted, categories)
}

/**
 * Canonical CMP footer link slots. Maps the ICU rich-text tag names
 * embedded in `cookieConsent.banner.footer` to the canonical (FR) path
 * key declared in `web/i18n/routing.ts`. The locale-aware URL is then
 * resolved by `legalHref()` so the rendered HTML always points to the
 * right localized segment (e.g. `/legal/privacy` on EN,
 * `/fr/legal/politique-confidentialite` on FR).
 *
 * Defined at module scope so the array is allocated exactly once — the
 * CMP boots on every page mount and re-allocating per call would be
 * gratuitous garbage.
 */
const FOOTER_LINK_SLOTS = [
  { tag: "privacy", path: "/legal/politique-confidentialite" },
  { tag: "cookies", path: "/cookies" },
  { tag: "notices", path: "/legal" },
  { tag: "subprocessors", path: "/sous-processeurs" },
] as const

/**
 * Translate the ICU rich-text `<privacy>…</privacy>` (etc.) tags in the
 * `cookieConsent.banner.footer` message into proper `<a>` HTML anchors
 * with the correct locale-aware hrefs. Returned as an HTML string so the
 * CMP can inject it directly into its modal DOM.
 *
 * `t.markup()` is the next-intl 4.x escape hatch that renders rich-text
 * tags to plain strings (instead of React nodes). Each tag function must
 * return a string — the result is concatenated and used verbatim by
 * vanilla-cookieconsent.
 */
function buildFooterMarkup(
  t: ReturnType<typeof useTranslations<"cookieConsent">>,
  locale: string,
): string {
  const values: Record<string, (chunks: string) => string> = {}
  for (const { tag, path } of FOOTER_LINK_SLOTS) {
    const href = legalHref(path, locale)
    values[tag] = (chunks) =>
      `<a href="${href}" class="cc__link">${chunks}</a>`
  }
  return t.markup("banner.footer", values)
}
