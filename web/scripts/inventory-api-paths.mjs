#!/usr/bin/env node
/**
 * inventory-api-paths.mjs
 *
 * Walks the web source tree and collects every `apiClient<T>(literal)` call
 * site so the F.3.2 typing sweep has a deterministic worklist AND so the
 * api-client integration tests can dynamically generate a per-path table.
 *
 * Output: web/scripts/api-paths.inventory.json
 *
 * Each entry has the shape:
 *   {
 *     file: string         // path relative to repo web/ root
 *     line: number         // 1-based line number of the call site
 *     method: string       // GET | POST | PUT | PATCH | DELETE
 *     path: string         // the literal request path (template chars
 *                          // collapsed to ":param" for grouping)
 *     rawPath: string      // exact path as written (template literal kept)
 *     typeArg: string      // the T inside apiClient<T>(...)
 *     hasBody: boolean     // whether a body was passed in options
 *   }
 *
 * Run via:
 *   node web/scripts/inventory-api-paths.mjs
 *
 * Notes:
 *   - We deliberately limit the parser to the patterns actually used in
 *     the codebase. apiClient is invoked with either a string literal or
 *     a template literal, never a runtime-built path. The parser is a
 *     deterministic regex pass — we do NOT spin up a TS AST so the
 *     script runs in a few hundred ms with no install.
 *   - Inventory is committed to the repo so reviewers can diff the
 *     surface between commits and the F.3.2 sweep can use it as the
 *     source of truth.
 */
import { readFileSync, readdirSync, statSync, writeFileSync } from "node:fs"
import { join, relative } from "node:path"
import { fileURLToPath } from "node:url"
import { dirname } from "node:path"

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const ROOT = join(__dirname, "..")
const SRC = join(ROOT, "src")
const OUTPUT = join(__dirname, "api-paths.inventory.json")

// File extensions to scan. Only include source modules; tests are skipped
// because the inventory is meant to drive PRODUCTION call sites, not
// fixture mocks.
const EXTS = new Set([".ts", ".tsx"])
const SKIP_BASENAMES = new Set(["__tests__", "__mocks__", "node_modules"])
const TEST_FILE_RE = /\.(test|spec)\.(ts|tsx)$/

function* walk(dir) {
  for (const entry of readdirSync(dir)) {
    if (SKIP_BASENAMES.has(entry)) continue
    const full = join(dir, entry)
    const st = statSync(full)
    if (st.isDirectory()) {
      yield* walk(full)
    } else if (st.isFile()) {
      if (TEST_FILE_RE.test(entry)) continue
      const dotIdx = entry.lastIndexOf(".")
      if (dotIdx >= 0 && EXTS.has(entry.slice(dotIdx))) {
        yield full
      }
    }
  }
}

// Match `apiClient<TypeArg>(<arg>)` and `apiClient(<arg>)` at the start of
// a call expression. The TypeArg is captured non-greedily so nested
// generics (e.g. `apiClient<{ data: Foo[] }>`) do not break the parser.
const CALL_RE = /apiClient(?:<([^>]+(?:<[^>]+>[^>]*)*)>)?\s*\(/g

// Method literal scan inside a call's options object. We tolerate single
// or double quotes and arbitrary whitespace; we do NOT lift dynamic
// expressions because no production site computes the method at runtime.
const METHOD_RE = /method\s*:\s*["']([A-Z]+)["']/

// Body scan — a body present anywhere inside the options literal flips
// the hasBody bit. Used downstream to verify Content-Type negotiation
// in the integration tests.
const BODY_RE = /\bbody\s*:/

function findClosingParen(text, openIdx) {
  // openIdx points to '(' — return the index of its matching ')'.
  let depth = 1
  let inSingle = false
  let inDouble = false
  let inTemplate = false
  let inLineComment = false
  let inBlockComment = false
  for (let i = openIdx + 1; i < text.length; i++) {
    const ch = text[i]
    const next = text[i + 1]
    if (inLineComment) {
      if (ch === "\n") inLineComment = false
      continue
    }
    if (inBlockComment) {
      if (ch === "*" && next === "/") {
        inBlockComment = false
        i++
      }
      continue
    }
    if (inSingle) {
      if (ch === "\\") {
        i++
        continue
      }
      if (ch === "'") inSingle = false
      continue
    }
    if (inDouble) {
      if (ch === "\\") {
        i++
        continue
      }
      if (ch === '"') inDouble = false
      continue
    }
    if (inTemplate) {
      if (ch === "\\") {
        i++
        continue
      }
      if (ch === "`") inTemplate = false
      continue
    }
    if (ch === "/" && next === "/") {
      inLineComment = true
      i++
      continue
    }
    if (ch === "/" && next === "*") {
      inBlockComment = true
      i++
      continue
    }
    if (ch === "'") {
      inSingle = true
      continue
    }
    if (ch === '"') {
      inDouble = true
      continue
    }
    if (ch === "`") {
      inTemplate = true
      continue
    }
    if (ch === "(") depth++
    if (ch === ")") {
      depth--
      if (depth === 0) return i
    }
  }
  return -1
}

function extractFirstStringArg(callBody) {
  // The first argument is either a string literal ("..."), a template
  // literal (`...${x}...`), or — rarely — an identifier (we resolve to
  // null in that case so the inventory only carries deterministic
  // paths). Returns { rawPath, normalizedPath } or null.
  const text = callBody.trimStart()
  if (text.startsWith('"') || text.startsWith("'")) {
    const quote = text[0]
    let end = -1
    for (let i = 1; i < text.length; i++) {
      const ch = text[i]
      if (ch === "\\") {
        i++
        continue
      }
      if (ch === quote) {
        end = i
        break
      }
    }
    if (end < 0) return null
    const raw = text.slice(1, end)
    return { rawPath: raw, normalizedPath: raw }
  }
  if (text.startsWith("`")) {
    // Walk character-by-character so nested braces inside `${...}`
    // chunks are matched correctly (e.g. `${query ? "?x" : ""}`).
    let end = -1
    let inExpr = 0
    for (let i = 1; i < text.length; i++) {
      const ch = text[i]
      if (ch === "\\") {
        i++
        continue
      }
      if (inExpr > 0) {
        if (ch === "{") inExpr++
        else if (ch === "}") inExpr--
        continue
      }
      if (ch === "$" && text[i + 1] === "{") {
        inExpr++
        i++
        continue
      }
      if (ch === "`") {
        end = i
        break
      }
    }
    if (end < 0) return null
    const raw = text.slice(1, end)
    // Replace `${ ... }` chunks with `:param` for grouping (depth-aware).
    let normalized = ""
    let i = 0
    while (i < raw.length) {
      if (raw[i] === "$" && raw[i + 1] === "{") {
        let depth = 1
        let j = i + 2
        while (j < raw.length && depth > 0) {
          if (raw[j] === "{") depth++
          else if (raw[j] === "}") depth--
          j++
        }
        normalized += ":param"
        i = j
      } else {
        normalized += raw[i]
        i++
      }
    }
    return { rawPath: raw, normalizedPath: normalized }
  }
  // Identifier or computed expression — skip.
  return null
}

function lineOfIndex(text, idx) {
  let line = 1
  for (let i = 0; i < idx; i++) {
    if (text[i] === "\n") line++
  }
  return line
}

const inventory = []

for (const file of walk(SRC)) {
  // Skip the api-client module itself — its single occurrence is the
  // export, not a call site.
  if (file.endsWith("api-client.ts")) continue

  const text = readFileSync(file, "utf8")
  CALL_RE.lastIndex = 0
  let match
  while ((match = CALL_RE.exec(text)) !== null) {
    const openIdx = match.index + match[0].length - 1
    const closeIdx = findClosingParen(text, openIdx)
    if (closeIdx < 0) continue
    const callBody = text.slice(openIdx + 1, closeIdx)
    const firstArg = extractFirstStringArg(callBody)
    if (!firstArg) continue
    const restText = callBody.slice(callBody.indexOf(firstArg.rawPath) + firstArg.rawPath.length + 1)
    const methodMatch = METHOD_RE.exec(restText)
    const method = methodMatch ? methodMatch[1] : "GET"
    const hasBody = BODY_RE.test(restText)
    inventory.push({
      file: relative(ROOT, file),
      line: lineOfIndex(text, match.index),
      method,
      path: firstArg.normalizedPath,
      rawPath: firstArg.rawPath,
      typeArg: match[1] ?? "unknown",
      hasBody,
    })
  }
}

// Stable sort: by file then line so diffs are reviewable.
inventory.sort((a, b) => {
  if (a.file !== b.file) return a.file < b.file ? -1 : 1
  return a.line - b.line
})

writeFileSync(OUTPUT, JSON.stringify(inventory, null, 2) + "\n", "utf8")

const byMethod = inventory.reduce((acc, entry) => {
  acc[entry.method] = (acc[entry.method] ?? 0) + 1
  return acc
}, {})

const summary = {
  totalSites: inventory.length,
  uniquePaths: new Set(inventory.map((e) => e.path)).size,
  byMethod,
  filesScanned: new Set(inventory.map((e) => e.file)).size,
}

console.log("api-paths.inventory.json written")
console.log(JSON.stringify(summary, null, 2))
