import { describe, it, expect } from "vitest"
import { buildCSP, type CSPEnv } from "../csp"

const PROD_ENV: CSPEnv = {
  NEXT_PUBLIC_API_URL: "https://api.example.app",
  NEXT_PUBLIC_WS_URL: "wss://api.example.app",
  NEXT_PUBLIC_LIVEKIT_URL: "wss://project.livekit.cloud",
  NEXT_PUBLIC_APP_URL: "https://app.example.app",
  NEXT_PUBLIC_STORAGE_URL: "https://pub-x.r2.dev",
}

function getDirective(csp: string, name: string): string {
  const directive = csp
    .split(";")
    .map((d) => d.trim())
    .find((d) => d.startsWith(`${name} `) || d === name)
  if (!directive) {
    throw new Error(`CSP directive ${name} not found in: ${csp}`)
  }
  return directive
}

describe("buildCSP — connect-src env-driven origins", () => {
  it("includes wss URLs from NEXT_PUBLIC_WS_URL and NEXT_PUBLIC_LIVEKIT_URL in prod", () => {
    const csp = buildCSP(
      {
        NEXT_PUBLIC_WS_URL: "wss://railway.app",
        NEXT_PUBLIC_LIVEKIT_URL: "wss://livekit.cloud",
      },
      true,
    )
    const connect = getDirective(csp, "connect-src")
    expect(connect).toContain("wss://railway.app")
    expect(connect).toContain("wss://livekit.cloud")
  })

  it("derives http+ws origin pair from NEXT_PUBLIC_API_URL in prod", () => {
    const csp = buildCSP(
      {
        NEXT_PUBLIC_API_URL: "https://api.x.app",
        NEXT_PUBLIC_WS_URL: "wss://api.x.app",
        NEXT_PUBLIC_LIVEKIT_URL: "wss://lk.example.app",
      },
      true,
    )
    const connect = getDirective(csp, "connect-src")
    expect(connect).toContain("https://api.x.app")
    expect(connect).toContain("wss://api.x.app")
  })

  it("falls back to dev localhost origins when not in production", () => {
    const csp = buildCSP({}, false)
    const connect = getDirective(csp, "connect-src")
    expect(connect).toContain("'self'")
    expect(connect).toContain("http://localhost:8083")
    expect(connect).toContain("ws://localhost:8083")
    expect(connect).toContain("wss://localhost:8083")
    expect(connect).toContain("http://localhost:9000")
  })

  it("always includes Stripe + R2 origins in connect-src", () => {
    const csp = buildCSP(PROD_ENV, true)
    const connect = getDirective(csp, "connect-src")
    expect(connect).toContain("https://*.stripe.com")
    expect(connect).toContain("https://api.stripe.com")
    expect(connect).toContain("https://*.r2.cloudflarestorage.com")
    expect(connect).toContain("https://*.r2.dev")
  })

  it("does not include dev fallback origins in production", () => {
    const csp = buildCSP(PROD_ENV, true)
    const connect = getDirective(csp, "connect-src")
    expect(connect).not.toContain("http://localhost")
    expect(connect).not.toContain("ws://localhost")
    expect(connect).not.toContain("192.168.1.156")
  })
})

describe("buildCSP — script-src environment awareness", () => {
  it("includes 'unsafe-eval' in development for HMR", () => {
    const csp = buildCSP({}, false)
    const scriptSrc = getDirective(csp, "script-src")
    expect(scriptSrc).toContain("'unsafe-eval'")
  })

  it("does NOT include 'unsafe-eval' in production", () => {
    const csp = buildCSP(PROD_ENV, true)
    const scriptSrc = getDirective(csp, "script-src")
    expect(scriptSrc).not.toContain("'unsafe-eval'")
  })

  it("includes Stripe script origins in both modes", () => {
    const dev = getDirective(buildCSP({}, false), "script-src")
    const prod = getDirective(buildCSP(PROD_ENV, true), "script-src")
    expect(dev).toContain("https://js.stripe.com")
    expect(prod).toContain("https://js.stripe.com")
  })
})

describe("buildCSP — fail-fast in production", () => {
  it("throws a clear error when NEXT_PUBLIC_WS_URL is missing in prod", () => {
    expect(() =>
      buildCSP(
        { NEXT_PUBLIC_LIVEKIT_URL: "wss://lk.example.app" },
        true,
      ),
    ).toThrow(/NEXT_PUBLIC_WS_URL is required in production/)
  })

  it("throws a clear error when NEXT_PUBLIC_LIVEKIT_URL is missing in prod", () => {
    expect(() =>
      buildCSP(
        { NEXT_PUBLIC_WS_URL: "wss://api.example.app" },
        true,
      ),
    ).toThrow(/NEXT_PUBLIC_LIVEKIT_URL is required in production/)
  })

  it("does NOT throw in development when those vars are missing", () => {
    expect(() => buildCSP({}, false)).not.toThrow()
  })

  it("throws on malformed NEXT_PUBLIC_WS_URL with the var name in the message", () => {
    expect(() =>
      buildCSP(
        {
          NEXT_PUBLIC_WS_URL: "not-a-url",
          NEXT_PUBLIC_LIVEKIT_URL: "wss://lk.example.app",
        },
        true,
      ),
    ).toThrow(/NEXT_PUBLIC_WS_URL is not a valid URL/)
  })

  it("throws on malformed NEXT_PUBLIC_API_URL with the var name in the message", () => {
    expect(() =>
      buildCSP(
        {
          NEXT_PUBLIC_API_URL: "://broken",
          NEXT_PUBLIC_WS_URL: "wss://api.example.app",
          NEXT_PUBLIC_LIVEKIT_URL: "wss://lk.example.app",
        },
        true,
      ),
    ).toThrow(/NEXT_PUBLIC_API_URL is not a valid URL/)
  })

  it("throws on malformed NEXT_PUBLIC_LIVEKIT_URL with the var name in the message", () => {
    expect(() =>
      buildCSP(
        {
          NEXT_PUBLIC_WS_URL: "wss://api.example.app",
          NEXT_PUBLIC_LIVEKIT_URL: "totally invalid",
        },
        true,
      ),
    ).toThrow(/NEXT_PUBLIC_LIVEKIT_URL is not a valid URL/)
  })

  it("throws on malformed NEXT_PUBLIC_STORAGE_URL", () => {
    expect(() =>
      buildCSP(
        {
          NEXT_PUBLIC_WS_URL: "wss://api.example.app",
          NEXT_PUBLIC_LIVEKIT_URL: "wss://lk.example.app",
          NEXT_PUBLIC_STORAGE_URL: "%%%",
        },
        true,
      ),
    ).toThrow(/NEXT_PUBLIC_STORAGE_URL is not a valid URL/)
  })
})

describe("buildCSP — media and image directives", () => {
  it("img-src always includes data: and blob:", () => {
    const csp = buildCSP(PROD_ENV, true)
    const imgSrc = getDirective(csp, "img-src")
    expect(imgSrc).toContain("data:")
    expect(imgSrc).toContain("blob:")
    expect(imgSrc).toContain("'self'")
  })

  it("media-src includes blob: and storage origin from NEXT_PUBLIC_STORAGE_URL", () => {
    const csp = buildCSP(PROD_ENV, true)
    const mediaSrc = getDirective(csp, "media-src")
    expect(mediaSrc).toContain("blob:")
    expect(mediaSrc).toContain("https://pub-x.r2.dev")
  })

  it("img-src and media-src include MinIO localhost in dev", () => {
    const csp = buildCSP({}, false)
    expect(getDirective(csp, "img-src")).toContain("http://localhost:9000")
    expect(getDirective(csp, "media-src")).toContain("http://localhost:9000")
  })

  it("media-src does not include MinIO localhost in prod", () => {
    const csp = buildCSP(PROD_ENV, true)
    expect(getDirective(csp, "media-src")).not.toContain("localhost")
  })
})

describe("buildCSP — structural invariants", () => {
  it("starts with default-src 'self' and ends with hardening directives", () => {
    const csp = buildCSP(PROD_ENV, true)
    expect(csp.startsWith("default-src 'self';")).toBe(true)
    expect(csp).toContain("frame-ancestors 'none'")
    expect(csp).toContain("object-src 'none'")
    expect(csp).toContain("base-uri 'self'")
    expect(csp).toContain("form-action 'self'")
  })

  it("frame-src whitelists Stripe (Embedded Components / hooks)", () => {
    const csp = buildCSP(PROD_ENV, true)
    const frameSrc = getDirective(csp, "frame-src")
    expect(frameSrc).toContain("https://js.stripe.com")
    expect(frameSrc).toContain("https://hooks.stripe.com")
    expect(frameSrc).toContain("https://*.stripe.com")
  })

  it("returns a single-line string with directives separated by ;", () => {
    const csp = buildCSP(PROD_ENV, true)
    expect(csp).not.toContain("\n")
    expect(csp.split(";").length).toBeGreaterThan(8)
  })

  it("connect-src whitelists *.posthog.com so the browser SDK can ship events", () => {
    const csp = buildCSP(PROD_ENV, true)
    const connect = getDirective(csp, "connect-src")
    expect(connect).toContain("https://*.posthog.com")
    expect(connect).toContain("https://*.i.posthog.com")
  })

  it("script-src whitelists PostHog so the SDK loader is not blocked", () => {
    const csp = buildCSP(PROD_ENV, true)
    const scriptSrc = getDirective(csp, "script-src")
    expect(scriptSrc).toContain("https://*.posthog.com")
  })

  it("connect-src includes the configured PostHog host in prod", () => {
    const csp = buildCSP(
      {
        ...PROD_ENV,
        NEXT_PUBLIC_POSTHOG_HOST: "https://eu.posthog.com",
      },
      true,
    )
    const connect = getDirective(csp, "connect-src")
    expect(connect).toContain("https://eu.posthog.com")
  })

  it("script-src whitelists Google Tag Manager so the GA4 loader is not blocked", () => {
    const csp = buildCSP(PROD_ENV, true)
    const scriptSrc = getDirective(csp, "script-src")
    expect(scriptSrc).toContain("https://www.googletagmanager.com")
  })

  it("connect-src whitelists Google Analytics endpoints", () => {
    const csp = buildCSP(PROD_ENV, true)
    const connect = getDirective(csp, "connect-src")
    expect(connect).toContain("https://www.google-analytics.com")
    expect(connect).toContain("https://*.analytics.google.com")
    expect(connect).toContain("https://*.googletagmanager.com")
  })

  it("img-src whitelists Google Analytics for the gtag pixel beacons", () => {
    const csp = buildCSP(PROD_ENV, true)
    const imgSrc = getDirective(csp, "img-src")
    expect(imgSrc).toContain("https://www.google-analytics.com")
    expect(imgSrc).toContain("https://*.analytics.google.com")
  })
})
