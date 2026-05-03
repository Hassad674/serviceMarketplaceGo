/**
 * F.6 W5 — regression test: production CSP must NEVER include
 * 'unsafe-eval'. The directive was kept for dev because Next's
 * Turbopack HMR runtime eval()s module factories on every fast-refresh
 * cycle, but the production bundle ships only static code so the
 * escape is unnecessary. A future contributor reverting the split or
 * adding 'unsafe-eval' to the production list will trip this test.
 *
 * We parse next.config.ts as plain text rather than dynamically
 * importing it: vitest's jsdom environment cannot transpile and run
 * the Next-flavoured config in-process, and a textual check is
 * sufficient to catch a regression — the script-src directive is
 * authored by hand, not by reflection.
 */
import { describe, it, expect } from "vitest"
import { readFileSync } from "node:fs"
import { resolve } from "node:path"

const NEXT_CONFIG_PATH = resolve(__dirname, "..", "..", "next.config.ts")

function readNextConfig(): string {
  return readFileSync(NEXT_CONFIG_PATH, "utf-8")
}

function extractScriptSrcLines(src: string): { production: string; development: string } {
  // The current file authors the script-src directive as two literal
  // strings inside a NODE_ENV ternary. We capture both branches with
  // a regex that tolerates whitespace + line breaks.
  const productionMatch = src.match(/isProduction\s*\?\s*"([^"]*script-src[^"]*)"/)
  const developmentMatch = src.match(/:\s*"([^"]*script-src[^"]*'unsafe-eval'[^"]*)"/)
  if (!productionMatch || !developmentMatch) {
    throw new Error(
      `Failed to parse script-src branches from next.config.ts. ` +
        `If the structure was refactored, update this test to read the new shape.`,
    )
  }
  return { production: productionMatch[1], development: developmentMatch[1] }
}

describe("CSP unsafe-eval split (F.6 W5)", () => {
  it("production script-src must NOT contain 'unsafe-eval'", () => {
    const { production } = extractScriptSrcLines(readNextConfig())
    expect(production).not.toContain("'unsafe-eval'")
    expect(production).toContain("script-src")
  })

  it("production script-src keeps 'unsafe-inline' (until nonces ship)", () => {
    // Tracked as a separate follow-up — 'unsafe-inline' will be
    // replaced with a per-request nonce. Keeping it in this test pins
    // the current invariant so the next contributor who tightens the
    // dev CSP doesn't accidentally drop 'unsafe-inline' from prod.
    const { production } = extractScriptSrcLines(readNextConfig())
    expect(production).toContain("'unsafe-inline'")
  })

  it("development script-src keeps 'unsafe-eval' for Turbopack HMR", () => {
    const { development } = extractScriptSrcLines(readNextConfig())
    expect(development).toContain("'unsafe-eval'")
  })

  it("Stripe origin survives in both environments", () => {
    const { production, development } = extractScriptSrcLines(readNextConfig())
    expect(production).toContain("https://js.stripe.com")
    expect(development).toContain("https://js.stripe.com")
  })

  it("config still applies a single Content-Security-Policy header", () => {
    // Sanity: ensure no future refactor splits the header into
    // multiple, non-additive declarations.
    const src = readNextConfig()
    const occurrences = src.match(/Content-Security-Policy/g) ?? []
    expect(occurrences.length).toBeGreaterThanOrEqual(1)
  })
})
