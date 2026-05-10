/**
 * Env-driven Content Security Policy builder.
 *
 * The CSP shipped with each Next.js response controls which origins the
 * browser is allowed to talk to (XHR/fetch, WebSocket, images, scripts,
 * frames…). Hardcoding the origins inside `next.config.ts` was the
 * source of a silent prod bug: when the backend domain rotated on
 * Railway the new WebSocket origin was not whitelisted and the browser
 * blocked it without surfacing a network error to the messaging UI
 * (online/offline broken, read-receipts dead, instant delivery
 * interrupted).
 *
 * This module derives every allow-listed origin from environment
 * variables so deployments only need to update Vercel/Railway env vars
 * to rotate domains. In production, missing required envs fail fast at
 * build/start time rather than at runtime in the browser.
 */

export interface CSPEnv {
  NEXT_PUBLIC_API_URL?: string
  NEXT_PUBLIC_WS_URL?: string
  NEXT_PUBLIC_APP_URL?: string
  NEXT_PUBLIC_LIVEKIT_URL?: string
  NEXT_PUBLIC_STORAGE_URL?: string
  /** PostHog SDK host (eu.posthog.com / us.posthog.com). Optional —
   * when missing the analytics CSP entry is omitted and the SDK is
   * inert anyway. */
  NEXT_PUBLIC_POSTHOG_HOST?: string
}

const STRIPE_ORIGINS = [
  "https://*.stripe.com",
  "https://api.stripe.com",
  "https://js.stripe.com",
  "https://hooks.stripe.com",
] as const

// PostHog browser SDK origins. Both EU and US hosts use a number of
// regional CDN endpoints — the wildcard captures the assets the SDK
// requests (config, recorder, decide). Connecting to either eu./us.
// is whitelisted via NEXT_PUBLIC_POSTHOG_HOST below; the static
// origin set here covers the vendor JS + the array endpoint that
// receives capture events.
const POSTHOG_ORIGINS = [
  "https://*.posthog.com",
  "https://*.i.posthog.com",
] as const

// Google Analytics 4 origins. The `gtag.js` loader lives on
// googletagmanager.com; events are POSTed to google-analytics.com and
// the regional analytics.google.com subdomains. The 1x1 pixel beacons
// fall on google-analytics.com and *.analytics.google.com under
// img-src.
const GA4_SCRIPT_ORIGINS = ["https://www.googletagmanager.com"] as const
const GA4_CONNECT_ORIGINS = [
  "https://www.google-analytics.com",
  "https://*.analytics.google.com",
  "https://*.googletagmanager.com",
] as const
const GA4_IMG_ORIGINS = [
  "https://www.google-analytics.com",
  "https://*.analytics.google.com",
] as const

const R2_ORIGINS = [
  "https://*.r2.cloudflarestorage.com",
  "https://*.r2.dev",
] as const

// City autocomplete uses two public geocoding APIs from the browser
// (no backend proxy). BAN — French national addresses
// (api-adresse.data.gouv.fr) — for FR cities, Photon
// (photon.komoot.io) — international fallback.
const CITY_AUTOCOMPLETE_ORIGINS = [
  "https://api-adresse.data.gouv.fr",
  "https://photon.komoot.io",
] as const

const DEV_HTTP_FALLBACKS = [
  "http://localhost:8083",
  "http://localhost:9000",
  "http://192.168.1.156:9000",
] as const

const DEV_WS_FALLBACKS = [
  "ws://localhost:8083",
  "wss://localhost:8083",
] as const

const DEV_MEDIA_FALLBACKS = [
  "http://localhost:9000",
  "http://192.168.1.156:9000",
] as const

/** Parse a value as an absolute URL or throw with a descriptive message. */
function parseEnvUrl(varName: string, value: string): URL {
  try {
    return new URL(value)
  } catch {
    throw new Error(
      `CSP: ${varName} is not a valid URL (got: "${value}"). ` +
        `Expected an absolute URL such as https://api.example.com.`,
    )
  }
}

/** Return `${protocol}//${host}` for a parsed URL. */
function originOf(url: URL): string {
  return `${url.protocol}//${url.host}`
}

/** Map an http(s) origin to its ws(s) counterpart. */
function toWsOrigin(httpOrigin: string): string {
  return httpOrigin.replace(/^http(s?):\/\//, "ws$1://")
}

/** Add `value` to `set` only when truthy and not already present. */
function addOrigin(set: Set<string>, value: string | undefined): void {
  if (!value) return
  set.add(value)
}

/**
 * Collect HTTP/WS origins for `connect-src` from env vars.
 * Throws in production if a required production-only origin is missing.
 */
function buildConnectOrigins(env: CSPEnv, isProduction: boolean): string[] {
  const origins = new Set<string>()
  STRIPE_ORIGINS.forEach((o) => origins.add(o))
  R2_ORIGINS.forEach((o) => origins.add(o))
  CITY_AUTOCOMPLETE_ORIGINS.forEach((o) => origins.add(o))
  POSTHOG_ORIGINS.forEach((o) => origins.add(o))
  GA4_CONNECT_ORIGINS.forEach((o) => origins.add(o))
  if (env.NEXT_PUBLIC_POSTHOG_HOST) {
    const hostUrl = parseEnvUrl(
      "NEXT_PUBLIC_POSTHOG_HOST",
      env.NEXT_PUBLIC_POSTHOG_HOST,
    )
    addOrigin(origins, originOf(hostUrl))
  }

  if (env.NEXT_PUBLIC_API_URL) {
    const apiUrl = parseEnvUrl("NEXT_PUBLIC_API_URL", env.NEXT_PUBLIC_API_URL)
    const httpOrigin = originOf(apiUrl)
    addOrigin(origins, httpOrigin)
    addOrigin(origins, toWsOrigin(httpOrigin))
  }

  if (isProduction && !env.NEXT_PUBLIC_WS_URL) {
    throw new Error(
      "CSP: NEXT_PUBLIC_WS_URL is required in production. " +
        "Set it to the backend WebSocket origin (e.g. wss://api.example.com).",
    )
  }
  if (env.NEXT_PUBLIC_WS_URL) {
    const wsUrl = parseEnvUrl("NEXT_PUBLIC_WS_URL", env.NEXT_PUBLIC_WS_URL)
    addOrigin(origins, originOf(wsUrl))
  }

  if (isProduction && !env.NEXT_PUBLIC_LIVEKIT_URL) {
    throw new Error(
      "CSP: NEXT_PUBLIC_LIVEKIT_URL is required in production. " +
        "Set it to the LiveKit wss origin (e.g. wss://project.livekit.cloud).",
    )
  }
  if (env.NEXT_PUBLIC_LIVEKIT_URL) {
    const livekitUrl = parseEnvUrl(
      "NEXT_PUBLIC_LIVEKIT_URL",
      env.NEXT_PUBLIC_LIVEKIT_URL,
    )
    addOrigin(origins, originOf(livekitUrl))
  }

  if (!isProduction) {
    DEV_HTTP_FALLBACKS.forEach((o) => origins.add(o))
    DEV_WS_FALLBACKS.forEach((o) => origins.add(o))
  }

  return Array.from(origins)
}

/** Collect origins for `img-src` / `media-src`. */
function buildMediaOrigins(env: CSPEnv, isProduction: boolean): string[] {
  const origins = new Set<string>()
  R2_ORIGINS.forEach((o) => origins.add(o))

  if (env.NEXT_PUBLIC_STORAGE_URL) {
    const storageUrl = parseEnvUrl(
      "NEXT_PUBLIC_STORAGE_URL",
      env.NEXT_PUBLIC_STORAGE_URL,
    )
    addOrigin(origins, originOf(storageUrl))
  }

  if (!isProduction) {
    DEV_MEDIA_FALLBACKS.forEach((o) => origins.add(o))
  }

  return Array.from(origins)
}

/**
 * Build the full CSP header value.
 *
 * @throws Error when an env var is malformed, or when a production-only
 *   required env var is missing.
 */
export function buildCSP(env: CSPEnv, isProduction: boolean): string {
  const connectOrigins = buildConnectOrigins(env, isProduction)
  const mediaOrigins = buildMediaOrigins(env, isProduction)

  // 'unsafe-eval' is required by Next/Turbopack HMR runtime in dev.
  // Production bundles ship only static code → drop it.
  const scriptOrigins = `${STRIPE_ORIGINS.join(" ")} ${POSTHOG_ORIGINS.join(" ")} ${GA4_SCRIPT_ORIGINS.join(" ")}`
  const scriptSrc = isProduction
    ? `script-src 'self' 'unsafe-inline' ${scriptOrigins}`
    : `script-src 'self' 'unsafe-inline' 'unsafe-eval' ${scriptOrigins}`

  const directives: string[] = [
    "default-src 'self'",
    scriptSrc,
    "style-src 'self' 'unsafe-inline'",
    `img-src 'self' data: blob: ${mediaOrigins.join(" ")} ${STRIPE_ORIGINS.join(" ")} ${GA4_IMG_ORIGINS.join(" ")}`,
    `media-src 'self' blob: ${mediaOrigins.join(" ")}`,
    "font-src 'self' data:",
    `connect-src 'self' ${connectOrigins.join(" ")}`,
    `frame-src ${STRIPE_ORIGINS.join(" ")}`,
    "frame-ancestors 'none'",
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
  ]

  return directives.join("; ")
}
