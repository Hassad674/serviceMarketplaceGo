/**
 * feature-flag.test.ts pins the env-var → engine resolution.
 * The module reads process.env at import time, so vitest needs
 * to swap the env BEFORE the dynamic import runs.
 */

import { afterEach, describe, expect, it, vi } from "vitest"

async function load(env: string | undefined) {
  vi.resetModules()
  if (env === undefined) {
    delete process.env.NEXT_PUBLIC_SEARCH_ENGINE
  } else {
    process.env.NEXT_PUBLIC_SEARCH_ENGINE = env
  }
  return await import("../feature-flag")
}

afterEach(() => {
  delete process.env.NEXT_PUBLIC_SEARCH_ENGINE
})

describe("feature-flag", () => {
  it("defaults to sql when env is unset", async () => {
    const { searchEngine, isTypesenseEnabled } = await load(undefined)
    expect(searchEngine()).toBe("sql")
    expect(isTypesenseEnabled()).toBe(false)
  })

  it("returns typesense when env is 'typesense'", async () => {
    const { searchEngine, isTypesenseEnabled } = await load("typesense")
    expect(searchEngine()).toBe("typesense")
    expect(isTypesenseEnabled()).toBe(true)
  })

  it("normalises case", async () => {
    const { searchEngine } = await load("TYPESENSE")
    expect(searchEngine()).toBe("typesense")
  })

  it("falls back to sql for unknown values", async () => {
    const { searchEngine, isTypesenseEnabled } = await load("elasticsearch")
    expect(searchEngine()).toBe("sql")
    expect(isTypesenseEnabled()).toBe(false)
  })
})
