import { test, expect } from "@playwright/test"
import path from "path"
import fs from "fs"
import os from "os"

import { registerProvider } from "../helpers/auth"

// ---------------------------------------------------------------------------
// SEC-09 + SEC-21 — Upload abuse via the freelance profile UI.
//
// Two attack vectors are exercised:
//   1. Direct SVG upload — must be refused (image/svg+xml is not in the
//      photo allowlist; SVG can carry inline <script>).
//   2. HTML payload disguised as PNG (.png filename + image/png MIME) —
//      must be refused by the magic-byte sniffer.
//
// We make the assertion at the API boundary rather than the UI: the
// front-end may render the error in many shapes, but the underlying
// /api/v1/upload/photo response is the contract we ship. We watch for
// the 4xx response and verify its body shape.
// ---------------------------------------------------------------------------

function tmpFile(name: string, contents: Buffer | string): string {
  const tmp = path.join(os.tmpdir(), `upload-abuse-${Date.now()}-${name}`)
  fs.writeFileSync(tmp, contents)
  return tmp
}

test.describe("SEC-09 / SEC-21 — upload abuse refused", () => {
  test("SVG upload via /api/v1/upload/photo is rejected with 415", async ({ page, request }) => {
    await registerProvider(page)

    // Pull the session cookie from the browser context.
    const cookies = await page.context().cookies()
    const sessionCookie = cookies.find((c) => c.name === "session_id")
    test.skip(!sessionCookie, "no session_id cookie — auth flow changed?")

    const svg = `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`
    const tmp = tmpFile("evil.svg", svg)

    const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"
    const resp = await request.post(`${apiBase}/api/v1/upload/photo`, {
      headers: {
        Cookie: `session_id=${sessionCookie!.value}`,
      },
      multipart: {
        file: {
          name: "evil.svg",
          mimeType: "image/svg+xml",
          buffer: fs.readFileSync(tmp),
        },
      },
    })

    expect(resp.status()).toBe(415)
    const body = await resp.json().catch(() => ({}))
    expect(body.error || body.code).toBeTruthy()
  })

  test("HTML disguised as PNG via /api/v1/upload/photo is rejected", async ({ page, request }) => {
    await registerProvider(page)

    const cookies = await page.context().cookies()
    const sessionCookie = cookies.find((c) => c.name === "session_id")
    test.skip(!sessionCookie, "no session_id cookie — auth flow changed?")

    const html = "<!DOCTYPE html><html><body><script>alert(1)</script></body></html>"
    const tmp = tmpFile("fake.png", html)

    const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080"
    const resp = await request.post(`${apiBase}/api/v1/upload/photo`, {
      headers: {
        Cookie: `session_id=${sessionCookie!.value}`,
      },
      multipart: {
        file: {
          name: "fake.png",
          mimeType: "image/png",
          buffer: fs.readFileSync(tmp),
        },
      },
    })

    expect(resp.status()).toBe(415)
  })
})
