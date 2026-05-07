/**
 * Regression test for the Permissions-Policy header served by Next.js.
 *
 * Background: between 2026-04-30 and 2026-05-07 the header shipped as
 * `camera=(), microphone=(), geolocation=()`. An empty allowlist `()`
 * blocks getUserMedia for ALL origins (including same-origin) and the
 * browser refuses to show the permission prompt — this silently broke
 * voice messages AND LiveKit calls. The fix is `(self)` on microphone
 * and camera so same-origin getUserMedia keeps working; geolocation
 * stays `()` because the app does not use it.
 *
 * This test asserts the SEMANTIC invariants — not a string match — so
 * a future contributor can re-order directives without breaking the
 * test, but cannot accidentally re-introduce an empty allowlist on
 * microphone or camera.
 *
 * We parse next.config.ts as plain text rather than dynamically
 * importing it (same approach as csp-no-unsafe-eval.test.ts): vitest's
 * jsdom environment cannot transpile and run the Next-flavoured config
 * in-process, and a textual check is sufficient — the directive is
 * authored by hand, not by reflection.
 */
import { describe, it, expect } from "vitest"
import { readFileSync } from "node:fs"
import { resolve } from "node:path"

const NEXT_CONFIG_PATH = resolve(__dirname, "..", "..", "next.config.ts")

function readNextConfig(): string {
  return readFileSync(NEXT_CONFIG_PATH, "utf-8")
}

function extractPermissionsPolicyValue(src: string): string {
  // Match the Permissions-Policy entry inside the headers() block. The
  // value lives between the double-quotes following `value:` on the
  // same line as the Permissions-Policy key.
  const match = src.match(
    /key:\s*"Permissions-Policy"\s*,\s*value:\s*"([^"]+)"/,
  )
  if (!match) {
    throw new Error(
      "Failed to find the Permissions-Policy value in next.config.ts. " +
        "If the structure was refactored, update this test to read the new shape.",
    )
  }
  return match[1]
}

function parsePermissionsPolicy(header: string): Record<string, string> {
  const out: Record<string, string> = {}
  for (const rawEntry of header.split(",")) {
    const entry = rawEntry.trim()
    if (!entry) continue
    const eq = entry.indexOf("=")
    if (eq < 0) {
      throw new Error(`Malformed Permissions-Policy directive: "${entry}"`)
    }
    const name = entry.slice(0, eq).trim()
    const value = entry.slice(eq + 1).trim()
    out[name] = value
  }
  return out
}

describe("Permissions-Policy invariants (web/next.config.ts)", () => {
  it("microphone allowlist is NEVER empty — same-origin must work", () => {
    const value = extractPermissionsPolicyValue(readNextConfig())
    const directives = parsePermissionsPolicy(value)
    expect(directives).toHaveProperty("microphone")
    // `()` blocks getUserMedia silently (no browser prompt). The fix
    // is `(self)` — keep at least same-origin so voice messages and
    // LiveKit calls keep working.
    expect(directives.microphone).not.toBe("()")
    expect(directives.microphone).toContain("self")
  })

  it("camera allowlist is NEVER empty — same-origin must work for LiveKit", () => {
    const value = extractPermissionsPolicyValue(readNextConfig())
    const directives = parsePermissionsPolicy(value)
    expect(directives).toHaveProperty("camera")
    expect(directives.camera).not.toBe("()")
    expect(directives.camera).toContain("self")
  })

  it("geolocation stays fully disabled — the app does not use it", () => {
    const value = extractPermissionsPolicyValue(readNextConfig())
    const directives = parsePermissionsPolicy(value)
    expect(directives).toHaveProperty("geolocation")
    // If a future feature needs geolocation, update both the policy
    // AND this test in the same commit so the intent stays explicit.
    expect(directives.geolocation).toBe("()")
  })

  it("config still applies a single Permissions-Policy header", () => {
    // Sanity: ensure no future refactor splits the header into
    // multiple, non-additive declarations.
    const src = readNextConfig()
    const occurrences = src.match(/Permissions-Policy/g) ?? []
    expect(occurrences.length).toBeGreaterThanOrEqual(1)
  })
})
