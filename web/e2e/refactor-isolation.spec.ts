import { test, expect } from "@playwright/test"
import { execSync } from "node:child_process"
import { readFileSync, statSync } from "node:fs"
import path from "node:path"

// ---------------------------------------------------------------------------
// Refactor isolation guards — Phase 3, Agent I
//
// These tests assert two structural invariants the provider→shared
// extraction must preserve:
//
//   1. The four extracted modules live under shared/ at their canonical
//      target paths and are non-empty.
//   2. The four old paths under features/provider/ are gone.
//   3. No file under web/src/ still imports the four modules from the
//      old features/provider/ paths.
//
// They are runtime checks against the on-disk source tree — fast, no
// browser navigation needed. They protect against an accidental
// reintroduction of the old paths in a follow-up patch.
// ---------------------------------------------------------------------------

const REPO_WEB_SRC = path.resolve(__dirname, "..", "src")

const NEW_PATHS = [
  path.join(REPO_WEB_SRC, "shared/lib/upload-api.ts"),
  path.join(REPO_WEB_SRC, "shared/components/expertise/expertise-editor.tsx"),
  path.join(REPO_WEB_SRC, "shared/components/location/city-autocomplete.tsx"),
  path.join(REPO_WEB_SRC, "shared/lib/search/search-api.ts"),
  // companion modules moved alongside city-autocomplete
  path.join(REPO_WEB_SRC, "shared/lib/location/city-search.ts"),
] as const

const OLD_PATHS = [
  path.join(REPO_WEB_SRC, "features/provider/api/upload-api.ts"),
  path.join(REPO_WEB_SRC, "features/provider/components/expertise-editor.tsx"),
  path.join(REPO_WEB_SRC, "features/provider/components/city-autocomplete.tsx"),
  path.join(REPO_WEB_SRC, "features/provider/api/search-api.ts"),
  path.join(REPO_WEB_SRC, "features/provider/lib/city-search.ts"),
] as const

const FORBIDDEN_IMPORT_PATTERNS = [
  "@/features/provider/api/upload-api",
  "@/features/provider/components/expertise-editor",
  "@/features/provider/components/city-autocomplete",
  "@/features/provider/api/search-api",
  "@/features/provider/lib/city-search",
] as const

test.describe("Refactor isolation — provider → shared extraction", () => {
  test("all four new shared paths exist and are non-empty", () => {
    for (const target of NEW_PATHS) {
      const stats = statSync(target)
      expect(stats.isFile(), `expected ${target} to be a file`).toBe(true)
      const contents = readFileSync(target, "utf-8")
      expect(
        contents.length,
        `expected ${target} to be non-empty`,
      ).toBeGreaterThan(50)
    }
  })

  test("all four old provider paths are deleted", () => {
    for (const oldPath of OLD_PATHS) {
      let exists = true
      try {
        statSync(oldPath)
      } catch {
        exists = false
      }
      expect(exists, `${oldPath} should have been deleted`).toBe(false)
    }
  })

  test("no file under web/src imports any of the four modules from the old provider paths", () => {
    // grep returns exit code 1 when nothing matched; we wrap and treat
    // that as success.
    for (const pattern of FORBIDDEN_IMPORT_PATTERNS) {
      let output = ""
      try {
        output = execSync(
          `grep -rln --include='*.ts' --include='*.tsx' ${JSON.stringify(pattern)} ${JSON.stringify(REPO_WEB_SRC)} || true`,
          { encoding: "utf-8" },
        )
      } catch (e) {
        // grep errored for some other reason — surface it
        throw e
      }
      const matches = output.trim().split("\n").filter(Boolean)
      expect(
        matches,
        `at least one file still imports the old path "${pattern}"`,
      ).toEqual([])
    }
  })
})
