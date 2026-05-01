/**
 * Smoke test for `web/scripts/check-no-raw-img.mjs`.
 *
 * The script is the structural backup that prevents an unannotated
 * `<img>` from sneaking into `src/`. We exercise it as a child
 * process against a temporary fixture tree so the test is hermetic
 * and doesn't accidentally pass/fail because of changes to the real
 * `web/src/` content.
 */
import { describe, it, expect, beforeAll, afterAll } from "vitest"
import { spawnSync } from "node:child_process"
import { mkdtempSync, mkdirSync, writeFileSync, rmSync, cpSync } from "node:fs"
import { tmpdir } from "node:os"
import { join, resolve } from "node:path"

const REAL_SCRIPT = resolve(__dirname, "..", "..", "scripts", "check-no-raw-img.mjs")

interface FakeWeb {
  root: string
  scripts: string
  src: string
}

function makeFakeWeb(): FakeWeb {
  const root = mkdtempSync(join(tmpdir(), "p4-check-"))
  const scripts = join(root, "scripts")
  const src = join(root, "src")
  mkdirSync(scripts, { recursive: true })
  mkdirSync(src, { recursive: true })
  cpSync(REAL_SCRIPT, join(scripts, "check-no-raw-img.mjs"))
  return { root, scripts, src }
}

function runCheck(scripts: string) {
  return spawnSync("node", [join(scripts, "check-no-raw-img.mjs")], {
    encoding: "utf8",
  })
}

describe("check-no-raw-img script", () => {
  let fake: FakeWeb

  beforeAll(() => {
    fake = makeFakeWeb()
  })

  afterAll(() => {
    if (fake) rmSync(fake.root, { recursive: true, force: true })
  })

  it("passes when src/ has no <img> tags", () => {
    writeFileSync(
      join(fake.src, "clean.tsx"),
      `export function Clean() { return <div>no images</div> }\n`,
    )
    const res = runCheck(fake.scripts)
    expect(res.status).toBe(0)
    expect(res.stdout).toMatch(/0 violations/)
    rmSync(join(fake.src, "clean.tsx"))
  })

  it("fails on a bare <img> with no annotation", () => {
    writeFileSync(
      join(fake.src, "bare.tsx"),
      `export function Bare() {\n  return (\n    <img src="/foo.png" alt="" />\n  )\n}\n`,
    )
    const res = runCheck(fake.scripts)
    expect(res.status).toBe(1)
    expect(res.stderr).toMatch(/unannotated/)
    expect(res.stderr).toMatch(/bare\.tsx/)
    rmSync(join(fake.src, "bare.tsx"))
  })

  it("accepts an <img> annotated with a blob: reason", () => {
    writeFileSync(
      join(fake.src, "blob.tsx"),
      `export function Blob({ url }: { url: string }) {
  return (
    // eslint-disable-next-line @next/next/no-img-element -- url is a blob: URL from URL.createObjectURL
    <img src={url} alt="" />
  )
}
`,
    )
    const res = runCheck(fake.scripts)
    expect(res.status).toBe(0)
    expect(res.stdout).toMatch(/annotated exceptions: 1/)
    rmSync(join(fake.src, "blob.tsx"))
  })

  it("accepts an <img> with multi-line SVG reason comment block", () => {
    writeFileSync(
      join(fake.src, "svg.tsx"),
      `export function Svg() {
  return (
    // Inline SVG asset kept as raw <img> for vector quality.
    // eslint-disable-next-line @next/next/no-img-element
    <img src="/logo.svg" alt="" />
  )
}
`,
    )
    const res = runCheck(fake.scripts)
    expect(res.status).toBe(0)
    expect(res.stdout).toMatch(/annotated exceptions: 1/)
    rmSync(join(fake.src, "svg.tsx"))
  })

  it("rejects a bare disable comment without a reason token", () => {
    writeFileSync(
      join(fake.src, "noreason.tsx"),
      `export function NoReason({ url }: { url: string }) {
  return (
    // eslint-disable-next-line @next/next/no-img-element
    <img src={url} alt="" />
  )
}
`,
    )
    const res = runCheck(fake.scripts)
    expect(res.status).toBe(1)
    expect(res.stderr).toMatch(/noreason\.tsx/)
    rmSync(join(fake.src, "noreason.tsx"))
  })

  it("ignores files inside __tests__ directories", () => {
    const testDir = join(fake.src, "__tests__")
    mkdirSync(testDir, { recursive: true })
    writeFileSync(
      join(testDir, "thing.test.tsx"),
      `export function FakeImg() { return <img src="/x.png" alt="" /> }\n`,
    )
    const res = runCheck(fake.scripts)
    expect(res.status).toBe(0)
    rmSync(testDir, { recursive: true, force: true })
  })

  it("ignores *.test.tsx files outside __tests__", () => {
    writeFileSync(
      join(fake.src, "foo.test.tsx"),
      `export function Foo() { return <img src="/x.png" alt="" /> }\n`,
    )
    const res = runCheck(fake.scripts)
    expect(res.status).toBe(0)
    rmSync(join(fake.src, "foo.test.tsx"))
  })
})
