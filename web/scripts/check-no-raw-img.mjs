#!/usr/bin/env node
/**
 * check-no-raw-img.mjs
 *
 * Fails if any source file under src/ contains a raw `<img` JSX tag
 * that is NOT immediately preceded by an `eslint-disable-next-line`
 * comment for `@next/next/no-img-element`.
 *
 * Why a custom script instead of (or alongside) ESLint:
 * - ESLint already enforces `@next/next/no-img-element` as `error`,
 *   but a future override could silently downgrade it back to `warn`.
 * - This script is a structural backup: it scans the source tree and
 *   refuses to count silenced sites unless they carry an explicit
 *   reason annotation, which makes the keep-as-img exceptions
 *   reviewable.
 *
 * Allowed exceptions are sites with one of the documented reasons:
 *   - `blob:` URL (URL.createObjectURL preview, no remote optimizer)
 *   - SVG inline asset (rare, kept for vector quality)
 *   - email template (email clients don't run JS)
 *
 * Run via:
 *   npm run check:no-raw-img
 *
 * Exit code: 0 on success, 1 on any unexplained `<img>` site.
 */
import { readFileSync, readdirSync, statSync } from "node:fs"
import { join, relative } from "node:path"
import { fileURLToPath } from "node:url"
import { dirname } from "node:path"

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const ROOT = join(__dirname, "..")
const SRC = join(ROOT, "src")

const EXTS = new Set([".tsx", ".jsx"])

// Skip test fixtures: vitest test files render `<img>` to assert the
// rendered DOM, which is the whole point of the test — not a perf
// regression we should chase through the production source tree.
const SKIP_BASENAMES = new Set(["__tests__", "__mocks__"])
const TEST_FILE_RE = /\.(test|spec)\.(tsx|jsx)$/

/**
 * Recursively walk the src tree and yield every .tsx/.jsx file path,
 * skipping `__tests__` directories and `*.test.tsx` / `*.spec.tsx`
 * test fixtures.
 */
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

/**
 * The only documented reasons for keeping a raw `<img>` instead of
 * migrating to `next/image`. A site is allowed only if its annotation
 * (or any nearby comment line above it) mentions one of these tokens.
 *
 * Why these and only these:
 *   - blob URL: `URL.createObjectURL` returns `blob:...` URIs that
 *     `next/image`'s loader cannot fetch through the optimizer; raw
 *     `<img>` is the documented Next.js workaround.
 *   - SVG inline: `next/image` defaults reject SVGs (rasterising
 *     vectors loses quality and can introduce CLS); allowed for local
 *     vector assets.
 *   - email template: components rendered server-side into emails
 *     never run JS; `next/image` would emit no useful markup.
 */
const ALLOWED_REASON_PATTERNS = [/blob:/i, /\bSVG\b/, /email template/i]

/**
 * Returns true if the previous non-empty comment lines contain an
 * `eslint-disable-next-line @next/next/no-img-element` annotation
 * AND a documented reason token (blob/SVG/email). A bare disable is
 * NOT enough — we want every kept-as-img site to carry a clear,
 * grep-friendly justification.
 */
function hasDisableAbove(lines, idx) {
  let sawDisable = false
  let sawReason = false
  // Scan back through contiguous comment lines (and the comment that
  // wraps the disable annotation). Stop at the first non-comment,
  // non-empty line.
  for (let i = idx - 1; i >= 0; i--) {
    const line = lines[i].trim()
    if (line === "") continue
    const isComment =
      line.startsWith("//") ||
      line.startsWith("/*") ||
      line.startsWith("*") ||
      line.startsWith("{/*")
    if (!isComment) break
    if (line.includes("eslint-disable-next-line") && line.includes("@next/next/no-img-element")) {
      sawDisable = true
    }
    for (const re of ALLOWED_REASON_PATTERNS) {
      if (re.test(line)) {
        sawReason = true
        break
      }
    }
    if (sawDisable && sawReason) return true
  }
  return sawDisable && sawReason
}

const violations = []
const annotated = []

for (const file of walk(SRC)) {
  const text = readFileSync(file, "utf8")
  const lines = text.split(/\r?\n/)
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    // Match the start of a JSX `<img` opening tag. We deliberately
    // avoid `<img>` literal in code comments by trimming and checking
    // the line starts with `<img` (modulo whitespace) — comment-only
    // mentions like `// Plain <img> rather than ...` start with `//`.
    const trimmed = line.trim()
    if (!trimmed.startsWith("<img")) continue
    const rel = relative(ROOT, file)
    if (hasDisableAbove(lines, i)) {
      annotated.push(`${rel}:${i + 1}`)
    } else {
      violations.push(`${rel}:${i + 1}`)
    }
  }
}

if (violations.length > 0) {
  console.error(
    `check-no-raw-img: ${violations.length} unannotated <img> site(s) found:`,
  )
  for (const v of violations) console.error(`  - ${v}`)
  console.error(
    "\nMigrate to `next/image` or annotate with `// eslint-disable-next-line @next/next/no-img-element -- <reason>` and document the reason.",
  )
  process.exit(1)
}

console.log(
  `check-no-raw-img: 0 violations (annotated exceptions: ${annotated.length})`,
)
for (const a of annotated) console.log(`  · ${a}`)
process.exit(0)
