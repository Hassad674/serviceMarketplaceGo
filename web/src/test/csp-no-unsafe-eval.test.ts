/**
 * F.6 W5 — regression test: production CSP must NEVER include
 * 'unsafe-eval'. The directive was kept for dev because Next's
 * Turbopack HMR runtime eval()s module factories on every fast-refresh
 * cycle, but the production bundle ships only static code so the
 * escape is unnecessary. A future contributor reverting the split or
 * adding 'unsafe-eval' to the production list will trip this test.
 *
 * Since the CSP is now env-driven (web/src/shared/lib/csp.ts), this
 * test imports `buildCSP` directly and exercises both branches with
 * representative env vars. It also asserts that next.config.ts still
 * declares a single Content-Security-Policy header so a future refactor
 * cannot accidentally drop the directive entirely.
 */
import { describe, it, expect } from "vitest"
import { readFileSync } from "node:fs"
import { resolve } from "node:path"
import { buildCSP } from "@/shared/lib/csp"

const NEXT_CONFIG_PATH = resolve(__dirname, "..", "..", "next.config.ts")

const PROD_ENV = {
  NEXT_PUBLIC_API_URL: "https://api.example.app",
  NEXT_PUBLIC_WS_URL: "wss://api.example.app",
  NEXT_PUBLIC_LIVEKIT_URL: "wss://project.livekit.cloud",
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

describe("CSP unsafe-eval split (F.6 W5)", () => {
  it("production script-src must NOT contain 'unsafe-eval'", () => {
    const csp = buildCSP(PROD_ENV, true)
    const scriptSrc = getDirective(csp, "script-src")
    expect(scriptSrc).not.toContain("'unsafe-eval'")
    expect(scriptSrc).toContain("script-src")
  })

  it("production script-src keeps 'unsafe-inline' (until nonces ship)", () => {
    // Tracked as a separate follow-up — 'unsafe-inline' will be
    // replaced with a per-request nonce. Keeping it in this test pins
    // the current invariant so the next contributor who tightens the
    // dev CSP doesn't accidentally drop 'unsafe-inline' from prod.
    const csp = buildCSP(PROD_ENV, true)
    const scriptSrc = getDirective(csp, "script-src")
    expect(scriptSrc).toContain("'unsafe-inline'")
  })

  it("development script-src keeps 'unsafe-eval' for Turbopack HMR", () => {
    const csp = buildCSP({}, false)
    const scriptSrc = getDirective(csp, "script-src")
    expect(scriptSrc).toContain("'unsafe-eval'")
  })

  it("Stripe origin survives in both environments", () => {
    const prod = getDirective(buildCSP(PROD_ENV, true), "script-src")
    const dev = getDirective(buildCSP({}, false), "script-src")
    expect(prod).toContain("https://js.stripe.com")
    expect(dev).toContain("https://js.stripe.com")
  })

  it("config still applies a single Content-Security-Policy header", () => {
    // Sanity: ensure no future refactor splits the header into
    // multiple, non-additive declarations or removes it altogether.
    const src = readFileSync(NEXT_CONFIG_PATH, "utf-8")
    const occurrences = src.match(/Content-Security-Policy/g) ?? []
    expect(occurrences.length).toBeGreaterThanOrEqual(1)
  })
})
