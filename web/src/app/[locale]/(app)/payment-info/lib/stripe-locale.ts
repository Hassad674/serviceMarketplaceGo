/**
 * Maps the app's locale (fr, en, de, es, it, pt, nl, etc.) to a Stripe-supported
 * BCP-47 locale string. Stripe Connect Embedded Components accept 40+ locales.
 *
 * Reference: https://docs.stripe.com/connect/supported-embedded-component-locales
 */

const APP_TO_STRIPE: Record<string, string> = {
  fr: "fr-FR",
  en: "en-US",
  de: "de-DE",
  es: "es-ES",
  it: "it-IT",
  pt: "pt-PT",
  "pt-br": "pt-BR",
  nl: "nl-NL",
  pl: "pl-PL",
  sv: "sv-SE",
  no: "nb-NO",
  da: "da-DK",
  fi: "fi-FI",
  ja: "ja-JP",
  ko: "ko-KR",
  zh: "zh-CN",
  "zh-tw": "zh-TW",
  "zh-hk": "zh-HK",
  ar: "ar",
  he: "he-IL",
  tr: "tr-TR",
  ru: "ru-RU",
  cs: "cs-CZ",
  sk: "sk-SK",
  hu: "hu-HU",
  ro: "ro-RO",
  bg: "bg-BG",
  el: "el-GR",
  hr: "hr-HR",
  sl: "sl-SI",
  et: "et-EE",
  lv: "lv-LV",
  lt: "lt-LT",
  vi: "vi-VN",
  th: "th-TH",
  id: "id-ID",
  ms: "ms-MY",
  fil: "fil-PH",
  mt: "mt-MT",
}

export function mapAppLocaleToStripe(appLocale: string): string {
  const normalized = appLocale.toLowerCase().trim()
  return APP_TO_STRIPE[normalized] ?? "auto"
}
