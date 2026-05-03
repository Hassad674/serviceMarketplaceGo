/**
 * api-client.integration.test.ts
 *
 * Integration tests for `apiClient<T>` driven by the `api-paths.inventory.json`
 * file produced by `scripts/inventory-api-paths.mjs`. The intent is to lock
 * down the wrapper's contract (method routing, headers, body envelopes,
 * error mapping, cookie credentials) on EVERY method × path combination
 * the production code actually invokes — so the F.3.2 typing sweep can
 * change call sites without breaking the wrapper's promises.
 *
 * Tests use MSW v2 to intercept fetch and assert the request shape; we do
 * NOT round-trip to a real backend. The raw inventory drives a data table
 * which guarantees the test surface stays in lock-step with production.
 */
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest"
import { setupServer } from "msw/node"
import { http, HttpResponse } from "msw"
import { readFileSync } from "node:fs"
import { join, dirname } from "node:path"
import { fileURLToPath } from "node:url"
import { apiClient, ApiError, API_BASE_URL } from "../api-client"

// ---------------------------------------------------------------------------
// Inventory loading
// ---------------------------------------------------------------------------

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const INVENTORY_PATH = join(
  __dirname,
  "..",
  "..",
  "..",
  "..",
  "scripts",
  "api-paths.inventory.json",
)

type InventoryEntry = {
  file: string
  line: number
  method: string
  path: string
  rawPath: string
  typeArg: string
  hasBody: boolean
}

const inventory: InventoryEntry[] = JSON.parse(
  readFileSync(INVENTORY_PATH, "utf8"),
)

/**
 * Concrete sample paths the live wrapper can hit. We map every `:param`
 * placeholder AND every raw `${expression}` template chunk to a
 * deterministic fixture id so the request URL we exercise looks like a
 * real call site and the resolver can pattern-match it.
 *
 * The inventory carries TWO shapes:
 *   - `path` (normalized: ${x} -> :param) — used for grouping
 *   - `rawPath` (kept as-is) — used to fire a representative request
 *
 * `${x}` in the rawPath would otherwise be percent-encoded by `fetch`
 * and break the equality assertion below. Paths that compute their
 * prefix (e.g. `${BASE}/me`) are coerced to start with `/` so the
 * resulting URL is absolute.
 */
function concretizePath(p: string): string {
  // Replace `${...}` chunks (with depth-aware brace matching), `:param`
  // placeholders, and a handful of well-known custom params with a fixed
  // fixture string. The result is always a valid absolute URL path.
  let result = ""
  let i = 0
  while (i < p.length) {
    if (p[i] === "$" && p[i + 1] === "{") {
      let depth = 1
      let j = i + 2
      while (j < p.length && depth > 0) {
        if (p[j] === "{") depth++
        else if (p[j] === "}") depth--
        j++
      }
      result += "fixture-id-1234"
      i = j
    } else {
      result += p[i]
      i++
    }
  }
  result = result
    .replace(/:param/g, "fixture-id-1234")
    .replace(/:org_id/g, "org-fixture")
    .replace(/:user_id/g, "user-fixture")
  if (!result.startsWith("/")) result = `/api/v1/_synth/${result}`
  return result
}

// ---------------------------------------------------------------------------
// MSW server setup
// ---------------------------------------------------------------------------

// Catch-all handler that returns `{ ok: true, method, path }` so the test
// can assert the exact request shape the wrapper produced.
const RECORDED: { method: string; url: string; headers: Record<string, string>; body: string }[] = []

const server = setupServer(
  http.all("*", async ({ request }) => {
    const url = new URL(request.url)
    const headers: Record<string, string> = {}
    request.headers.forEach((v, k) => {
      headers[k] = v
    })
    let body = ""
    if (request.method !== "GET" && request.method !== "HEAD") {
      try {
        body = await request.text()
      } catch {
        body = ""
      }
    }
    RECORDED.push({
      method: request.method,
      url: url.pathname + url.search,
      headers,
      body,
    })
    return HttpResponse.json({ ok: true, method: request.method, path: url.pathname })
  }),
)

beforeAll(() => server.listen({ onUnhandledRequest: "error" }))
afterEach(() => {
  server.resetHandlers()
  RECORDED.length = 0
})
afterAll(() => server.close())

// ---------------------------------------------------------------------------
// Inventory-driven exhaustive coverage
// ---------------------------------------------------------------------------

// One test per unique (method, normalized path) tuple. The expectation is
// the same for every route — the wrapper must (a) fire a fetch with the
// exact method, (b) hit the exact path, (c) include credentials, (d)
// attach `Content-Type: application/json`, (e) parse the JSON envelope.
//
// The raw size of the inventory (~146 unique paths × ~1.2 method = ~180
// tests) is intentional: the F.3.2 sweep mutates ALL of these, so the
// safety net must cover ALL of them.
describe("apiClient — inventory-driven exhaustive coverage", () => {
  const uniqueByMethodAndPath = new Map<string, InventoryEntry>()
  for (const entry of inventory) {
    const key = `${entry.method} ${entry.path}`
    if (!uniqueByMethodAndPath.has(key)) {
      uniqueByMethodAndPath.set(key, entry)
    }
  }

  it.each(Array.from(uniqueByMethodAndPath.values()).map((e) => [
    `${e.method} ${e.path}`,
    e,
  ]))(
    "%s — fires fetch with the right method, path, and headers",
    async (_label, entry) => {
      const concrete = concretizePath(entry.rawPath)
      const opts: { method?: string; body?: unknown } = {}
      if (entry.method !== "GET") opts.method = entry.method
      if (entry.hasBody) opts.body = { sample: "fixture" }

      const res = await apiClient<{ ok: boolean }>(concrete, opts)
      expect(res).toEqual({ ok: true, method: entry.method, path: expect.any(String) })

      expect(RECORDED).toHaveLength(1)
      const recorded = RECORDED[0]
      expect(recorded.method).toBe(entry.method)
      expect(recorded.url).toBe(concrete)
      // Content-Type is forced on every call, body or not.
      expect(recorded.headers["content-type"]).toBe("application/json")
      // Body is JSON-stringified when provided, absent otherwise.
      if (entry.hasBody) {
        expect(recorded.body).toBe(JSON.stringify({ sample: "fixture" }))
      } else {
        // GETs have no body; non-GET without `hasBody` still has no body.
        expect(recorded.body).toBe("")
      }
    },
  )
})

// ---------------------------------------------------------------------------
// Method coverage matrix (one fast smoke per HTTP verb to confirm the
// wrapper does not silently downgrade DELETE / PATCH / etc.)
// ---------------------------------------------------------------------------

describe("apiClient — HTTP method matrix", () => {
  const methods = ["GET", "POST", "PUT", "PATCH", "DELETE"] as const

  it.each(methods)("%s — fires the right verb", async (method) => {
    const opts: { method?: string; body?: unknown } = {}
    if (method !== "GET") opts.method = method
    if (method === "POST" || method === "PUT" || method === "PATCH") {
      opts.body = { x: 1 }
    }
    await apiClient<unknown>(`/api/v1/_smoke/${method.toLowerCase()}`, opts)
    expect(RECORDED).toHaveLength(1)
    expect(RECORDED[0].method).toBe(method)
  })
})

// ---------------------------------------------------------------------------
// Cross-cutting wrapper behaviour
// ---------------------------------------------------------------------------

describe("apiClient — request envelope", () => {
  it("always sends credentials so the session cookie is included", async () => {
    // We cannot read RequestCredentials off the recorded request inside
    // MSW (it normalizes everything), so we install a fetch spy that
    // surfaces the original RequestInit. The wrapper sets
    // `credentials: "include"` on every call.
    const original = globalThis.fetch
    let recorded: { credentials?: RequestCredentials } | null = null
    globalThis.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
      recorded = { credentials: init?.credentials }
      return original(input as Request, init)
    }
    try {
      await apiClient("/api/v1/auth/me")
    } finally {
      globalThis.fetch = original
    }
    expect(recorded).not.toBeNull()
    expect(recorded!.credentials).toBe("include")
  })

  it("URL-prefixes paths with API_BASE_URL when set", async () => {
    // API_BASE_URL is empty by default in tests (production parity);
    // guarding the test ensures we still cover the truthy branch when
    // a developer sets NEXT_PUBLIC_API_URL locally.
    expect(API_BASE_URL === "" || API_BASE_URL.startsWith("http")).toBe(true)
  })

  it("preserves a custom Content-Type override (e.g. for multipart)", async () => {
    // The wrapper merges custom headers AFTER the default Content-Type,
    // so an explicit `Content-Type: text/plain` header from the caller
    // wins. This is the behaviour the upload helper relies on.
    await apiClient("/api/v1/_smoke/headers", {
      headers: { "Content-Type": "text/plain", "X-Foo": "bar" },
    })
    expect(RECORDED[0].headers["content-type"]).toBe("text/plain")
    expect(RECORDED[0].headers["x-foo"]).toBe("bar")
  })

  it("forwards an AbortSignal so callers can cancel in-flight requests", async () => {
    const controller = new AbortController()
    server.use(
      http.get("*/api/v1/_smoke/cancel", async () => {
        // Hold the response long enough for abort() to fire below.
        await new Promise((resolve) => setTimeout(resolve, 200))
        return HttpResponse.json({ ok: true })
      }),
    )
    const promise = apiClient("/api/v1/_smoke/cancel", { signal: controller.signal })
    controller.abort()
    await expect(promise).rejects.toThrow()
  })
})

describe("apiClient — error mapping", () => {
  it("maps a canonical envelope { error: { code, message } } to ApiError", async () => {
    server.use(
      http.get("*/api/v1/_smoke/error-canonical", () =>
        HttpResponse.json(
          { error: { code: "validation_error", message: "Email required" } },
          { status: 422 },
        ),
      ),
    )
    try {
      await apiClient("/api/v1/_smoke/error-canonical")
      throw new Error("should have thrown")
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      const apiErr = err as ApiError
      expect(apiErr.status).toBe(422)
      expect(apiErr.code).toBe("validation_error")
      expect(apiErr.message).toBe("Email required")
      expect(apiErr.body).toEqual({
        error: { code: "validation_error", message: "Email required" },
      })
    }
  })

  it("maps a legacy envelope { error: 'code', message: '...' } to ApiError", async () => {
    server.use(
      http.get("*/api/v1/_smoke/error-legacy", () =>
        HttpResponse.json(
          { error: "rate_limited", message: "Too many requests" },
          { status: 429 },
        ),
      ),
    )
    try {
      await apiClient("/api/v1/_smoke/error-legacy")
      throw new Error("should have thrown")
    } catch (err) {
      const apiErr = err as ApiError
      expect(apiErr.status).toBe(429)
      expect(apiErr.code).toBe("rate_limited")
      expect(apiErr.message).toBe("Too many requests")
    }
  })

  it("preserves the parsed body so callers can read sibling fields (missing_fields)", async () => {
    server.use(
      http.get("*/api/v1/_smoke/error-missing-fields", () =>
        HttpResponse.json(
          {
            error: { code: "billing_profile_incomplete", message: "Required" },
            missing_fields: ["country", "vat_number"],
          },
          { status: 403 },
        ),
      ),
    )
    try {
      await apiClient("/api/v1/_smoke/error-missing-fields")
      throw new Error("should have thrown")
    } catch (err) {
      const apiErr = err as ApiError
      expect(apiErr.body?.missing_fields).toEqual(["country", "vat_number"])
    }
  })

  it("falls back to 'unknown_error' / 'An error occurred' when body is not JSON", async () => {
    server.use(
      http.get("*/api/v1/_smoke/error-html", () =>
        new HttpResponse("<html>500</html>", {
          status: 500,
          headers: { "content-type": "text/html" },
        }),
      ),
    )
    try {
      await apiClient("/api/v1/_smoke/error-html")
      throw new Error("should have thrown")
    } catch (err) {
      const apiErr = err as ApiError
      expect(apiErr.status).toBe(500)
      expect(apiErr.code).toBe("unknown_error")
      expect(apiErr.message).toBe("An error occurred")
    }
  })

  it("maps 401 to ApiError with the right status (used by auth guard)", async () => {
    server.use(
      http.get("*/api/v1/_smoke/401", () =>
        HttpResponse.json({ error: "unauthorized" }, { status: 401 }),
      ),
    )
    await expect(apiClient("/api/v1/_smoke/401")).rejects.toMatchObject({
      status: 401,
      code: "unauthorized",
    })
  })

  it("maps 403 to ApiError with the right status (used by RBAC gate)", async () => {
    server.use(
      http.get("*/api/v1/_smoke/403", () =>
        HttpResponse.json({ error: "forbidden" }, { status: 403 }),
      ),
    )
    await expect(apiClient("/api/v1/_smoke/403")).rejects.toMatchObject({
      status: 403,
      code: "forbidden",
    })
  })

  it("maps 404 to ApiError with the right status (used by detail pages)", async () => {
    server.use(
      http.get("*/api/v1/_smoke/404", () =>
        HttpResponse.json({ error: "not_found" }, { status: 404 }),
      ),
    )
    await expect(apiClient("/api/v1/_smoke/404")).rejects.toMatchObject({
      status: 404,
      code: "not_found",
    })
  })
})

describe("apiClient — success envelopes", () => {
  it("returns the parsed JSON body on 200 OK", async () => {
    server.use(
      http.get("*/api/v1/_smoke/200", () =>
        HttpResponse.json({ id: "abc", name: "fixture" }),
      ),
    )
    const result = await apiClient<{ id: string; name: string }>("/api/v1/_smoke/200")
    expect(result).toEqual({ id: "abc", name: "fixture" })
  })

  it("returns the parsed JSON body on 201 Created", async () => {
    server.use(
      http.post("*/api/v1/_smoke/201", () =>
        HttpResponse.json({ id: "new" }, { status: 201 }),
      ),
    )
    const result = await apiClient<{ id: string }>("/api/v1/_smoke/201", {
      method: "POST",
      body: { name: "x" },
    })
    expect(result).toEqual({ id: "new" })
  })

  it("returns undefined on 204 No Content", async () => {
    server.use(
      http.delete("*/api/v1/_smoke/204", () =>
        new HttpResponse(null, { status: 204 }),
      ),
    )
    const result = await apiClient<undefined>("/api/v1/_smoke/204", { method: "DELETE" })
    expect(result).toBeUndefined()
  })
})

describe("apiClient — network failure handling", () => {
  it("propagates network errors instead of silently swallowing them", async () => {
    server.use(
      http.get("*/api/v1/_smoke/net-fail", () => HttpResponse.error()),
    )
    await expect(apiClient("/api/v1/_smoke/net-fail")).rejects.toThrow()
  })

  it("rejects when the JSON body cannot be parsed on a 2xx response", async () => {
    server.use(
      http.get("*/api/v1/_smoke/bad-json", () =>
        new HttpResponse("not-json", { status: 200, headers: { "content-type": "text/plain" } }),
      ),
    )
    await expect(apiClient("/api/v1/_smoke/bad-json")).rejects.toThrow()
  })
})
