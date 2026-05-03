#!/usr/bin/env node
/**
 * sweep-api-paths.mjs
 *
 * F.3.2 Stage 3 codemod: rewrites every `apiClient<TypeArg>(...)` site so
 * the request path is anchored to the OpenAPI contract via the helpers
 * declared in `src/shared/lib/api-paths.ts`. The type argument becomes
 *
 *   apiClient<Get<"/api/v1/path"> & TypeArg>(path)
 *
 * (or Post / Put / Patch / Delete depending on the call's method). The
 * intersection preserves whatever local DTO the caller named so callers
 * downstream see the same shape they always saw — no behaviour change,
 * pure typing.
 *
 * Driven by:
 *   - web/scripts/api-paths.inventory.json            (call-site list)
 *   - backend/internal/handler/testdata/openapi.golden.json (path canonicaliser)
 *
 * The OpenAPI schema is consulted to translate the inventory's
 * `:param` placeholders into the actual parameter names declared in
 * the schema (e.g. `:param/applications/:param` → `/{id}/applications/{applicantId}`).
 *
 * Run from the web/ root:
 *   node scripts/sweep-api-paths.mjs
 *
 * Idempotent — re-running on a fully-swept tree is a no-op.
 */

import { readFileSync, writeFileSync } from "node:fs"
import { join, dirname } from "node:path"
import { fileURLToPath } from "node:url"

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const ROOT = join(__dirname, "..")
const INVENTORY = JSON.parse(
  readFileSync(join(__dirname, "api-paths.inventory.json"), "utf8"),
)
const OPENAPI = JSON.parse(
  readFileSync(
    join(__dirname, "..", "..", "backend", "internal", "handler", "testdata", "openapi.golden.json"),
    "utf8",
  ),
)

// ---------------------------------------------------------------------------
// Build a (method, segments) → canonicalPath index from the OpenAPI doc.
// segments is the path's slash-split form with `{whatever}` collapsed to
// the placeholder marker `*`. The codemod uses this to look up the right
// canonical path for an inventory entry whose static portion may use
// :param-anonymous placeholders.
// ---------------------------------------------------------------------------

const SCHEMA_INDEX = new Map()

for (const [path, ops] of Object.entries(OPENAPI.paths ?? {})) {
  for (const method of ["get", "post", "put", "patch", "delete"]) {
    if (!ops[method]) continue
    const segs = path.split("/").map((s) => (s.startsWith("{") && s.endsWith("}") ? "*" : s))
    const key = method.toUpperCase() + " " + segs.join("/")
    if (!SCHEMA_INDEX.has(key)) SCHEMA_INDEX.set(key, path)
  }
}

// inventoryToCanonical takes an inventory entry and returns the
// canonical OpenAPI path string, or null if no match. Strips the
// query string before matching — paths in the OpenAPI doc never
// carry one. Handles three normalisations:
//
//   1. Drop `?...` query suffix (the OpenAPI doc lives at the path level).
//   2. Drop a TRAILING `:param` placeholder when it sat in a position
//      adjacent to a query-string template (`/jobs/mine${query}` →
//      inventory shows `/jobs/mine:param`; the canonical path is
//      `/jobs/mine`).
//   3. Try with a trailing-slash flip (chi exposes both forms).
function inventoryToCanonical(entry) {
  let p = entry.path
  const qIdx = p.indexOf("?")
  if (qIdx >= 0) p = p.slice(0, qIdx)

  const tryMatch = (candidate) => {
    const segs = candidate.split("/").map((s) => (s === ":param" ? "*" : s))
    const key = entry.method + " " + segs.join("/")
    if (SCHEMA_INDEX.has(key)) return SCHEMA_INDEX.get(key)
    if (key.endsWith("/")) {
      const alt = key.slice(0, -1)
      if (SCHEMA_INDEX.has(alt)) return SCHEMA_INDEX.get(alt)
    } else {
      const alt = key + "/"
      if (SCHEMA_INDEX.has(alt)) return SCHEMA_INDEX.get(alt)
    }
    return null
  }

  let match = tryMatch(p)
  if (match) return match

  // Strip a trailing `:param` (typically a query-string template
  // literal) and retry — `/api/v1/jobs/mine:param` → `/api/v1/jobs/mine`.
  if (p.endsWith(":param")) {
    match = tryMatch(p.slice(0, -":param".length))
    if (match) return match
  }
  // Some inventory entries carry a trailing `:param` in the middle
  // (e.g. `/api/v1/jobs/{id}/applications:param`) — same fix applies
  // because the trailing placeholder is the merged query-string suffix.
  // We only strip the LAST `:param` token; deeper templates are
  // assumed to be real path params.
  return null
}

const METHOD_TO_HELPER = {
  GET: "Get",
  POST: "Post",
  PUT: "Put",
  PATCH: "Patch",
  DELETE: "Delete",
}

// Skip rules — leave call/livekit alone (OFF-LIMITS), leave
// dynamically-computed bases (`${BASE}/...`) alone — those need to
// be hand-edited because the inventory parser couldn't resolve the
// base path and the canonical lookup will miss.
//
// Also skip the api-paths.ts module itself — the inventory parser
// indexes the JSDoc example as a real call site; rewriting it would
// break the documentation contract.
function shouldSkip(entry) {
  if (entry.file.includes("features/call/")) return true
  if (entry.file.endsWith("shared/lib/api-paths.ts")) return true
  if (!entry.path.startsWith("/api/v1/")) return true
  return false
}

// ---------------------------------------------------------------------------
// File-level sweep
// ---------------------------------------------------------------------------

const byFile = new Map()
const skipped = []
for (const e of INVENTORY) {
  if (shouldSkip(e)) {
    skipped.push(e)
    continue
  }
  const canonical = inventoryToCanonical(e)
  if (!canonical) {
    skipped.push({ ...e, reason: "no-canonical-match" })
    continue
  }
  if (!byFile.has(e.file)) byFile.set(e.file, [])
  byFile.get(e.file).push({ ...e, canonical })
}

let totalSwept = 0
let totalFiles = 0

for (const [file, entries] of byFile) {
  // Sort entries by line desc so per-line edits don't shift later
  // positions. Each entry is matched to ITS OWN source line — when
  // multiple entries share the same `apiClient<void>(` text, we
  // disambiguate by line number rather than first-match.
  entries.sort((a, b) => b.line - a.line)
  const fullPath = join(ROOT, file)
  let text = readFileSync(fullPath, "utf8")
  const original = text
  let helpersUsed = new Set()

  for (const e of entries) {
    const helper = METHOD_TO_HELPER[e.method] ?? "Get"
    const oapiPath = e.canonical
    // The inventory captures missing type args as the literal string
    // "unknown". Build a regex that matches the no-arg form
    // (`apiClient(`) AND the explicit `apiClient<unknown>(` form,
    // OR for any other typeArg the explicit `apiClient<TypeName>(`.
    const typeArgEscaped = e.typeArg.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
    const lineRegex = e.typeArg === "unknown"
      ? /apiClient(?:<unknown>)?\s*\(/
      : new RegExp(`apiClient<${typeArgEscaped}>\\s*\\(`)
    // Idempotent guard: skip if this site is already swept.
    if (e.typeArg.startsWith(`${helper}<`) || e.typeArg.startsWith("Void<")) {
      continue
    }
    // Locate the exact line. e.line is 1-based.
    const lines = text.split("\n")
    const idx = e.line - 1
    if (idx < 0 || idx >= lines.length) continue
    const line = lines[idx]
    // When the local type is `void`, the caller's intent is "ignore
    // the response — the function returns Promise<void>". The Void<P>
    // helper validates the path against the OpenAPI contract while
    // resolving to plain `void`, preserving the callsite's
    // Promise<void> annotation.
    //
    // When the local type is "unknown" (the inventory's marker for a
    // missing type arg), use the helper directly — no intersection
    // since there's no caller-supplied type to preserve.
    //
    // For all other types, intersect with the OpenAPI-derived shape
    // so the path-validation runs and the local type stays the
    // ground truth where it's richer than the schema.
    const newType = (() => {
      if (e.typeArg === "void") {
        return `Void<"${oapiPath}">`
      }
      if (e.typeArg === "unknown") {
        return `${helper}<"${oapiPath}">`
      }
      return `${helper}<"${oapiPath}"> & ${e.typeArg}`
    })()
    const replacement = `apiClient<${newType}>(`
    const helperToImport = e.typeArg === "void" ? "Void" : helper
    if (!lineRegex.test(line)) {
      let found = false
      for (let off = -2; off <= 2; off++) {
        const j = idx + off
        if (j < 0 || j >= lines.length) continue
        if (lineRegex.test(lines[j])) {
          lines[j] = lines[j].replace(lineRegex, replacement)
          helpersUsed.add(helperToImport)
          totalSwept++
          found = true
          break
        }
      }
      if (!found) continue
      text = lines.join("\n")
      continue
    }
    lines[idx] = line.replace(lineRegex, replacement)
    text = lines.join("\n")
    helpersUsed.add(helperToImport)
    totalSwept++
  }

  if (text !== original && helpersUsed.size > 0) {
    const importLine = `import type { ${[...helpersUsed].sort().join(", ")} } from "@/shared/lib/api-paths"`
    if (!text.includes(`from "@/shared/lib/api-paths"`)) {
      const apiClientImport = /import\s+\{[^}]*apiClient[^}]*\}\s+from\s+["']@\/shared\/lib\/api-client["']\s*\n?/
      const m = text.match(apiClientImport)
      if (m) {
        const insertAt = (m.index ?? 0) + m[0].length
        text = text.slice(0, insertAt) + importLine + "\n" + text.slice(insertAt)
      } else {
        text = importLine + "\n" + text
      }
    } else {
      const existingImport = text.match(/import\s+type\s+\{([^}]*)\}\s+from\s+["']@\/shared\/lib\/api-paths["']/)
      if (existingImport) {
        const names = new Set(
          existingImport[1].split(",").map((s) => s.trim()).filter(Boolean),
        )
        for (const h of helpersUsed) names.add(h)
        const newImport = `import type { ${[...names].sort().join(", ")} } from "@/shared/lib/api-paths"`
        text = text.replace(existingImport[0], newImport)
      }
    }
  }

  if (text !== original) {
    writeFileSync(fullPath, text, "utf8")
    totalFiles++
  }
}

console.log(`swept ${totalSwept} call sites across ${totalFiles} files`)
if (skipped.length > 0) {
  const reasons = {}
  for (const s of skipped) {
    const r = s.reason ?? (s.file.includes("features/call/") ? "call-livekit" : "non-api-path")
    reasons[r] = (reasons[r] ?? 0) + 1
  }
  console.log(`skipped ${skipped.length}:`, reasons)
}
