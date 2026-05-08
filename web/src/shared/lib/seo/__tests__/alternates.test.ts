import { describe, it, expect } from "vitest"

import { buildAlternates, absoluteUrl, localizedPath } from "../alternates"

describe("buildAlternates", () => {
  it("emits canonical + every supported locale + x-default", () => {
    const out = buildAlternates({ locale: "fr", path: "/freelancers/abc" })
    // canonical mirrors the locale being rendered.
    expect(out.canonical).toMatch(/\/fr\/freelancers\/abc$/)
    expect(out.languages.fr).toMatch(/\/fr\/freelancers\/abc$/)
    expect(out.languages.en).toMatch(/\/en\/freelancers\/abc$/)
    expect(out.languages["x-default"]).toMatch(/\/fr\/freelancers\/abc$/)
  })

  it("uses absolute URLs (origin matches NEXT_PUBLIC_APP_URL)", () => {
    const out = buildAlternates({ locale: "en", path: "/agencies/xyz" })
    expect(out.canonical).toMatch(/^https?:\/\//)
    expect(out.languages.fr).toMatch(/^https?:\/\//)
  })

  it("normalizes leading slash on the path", () => {
    const out = buildAlternates({ locale: "en", path: "agencies/xyz" })
    expect(out.canonical).toMatch(/\/en\/agencies\/xyz$/)
  })
})

describe("absoluteUrl", () => {
  it("prepends the site origin to a relative path", () => {
    const url = absoluteUrl("/freelancers/abc")
    expect(url).toMatch(/^https?:\/\/.+\/freelancers\/abc$/)
  })

  it("returns the input unchanged for already-absolute URLs", () => {
    expect(absoluteUrl("https://r2.example/photo.jpg")).toBe(
      "https://r2.example/photo.jpg",
    )
    expect(absoluteUrl("http://localhost/x")).toBe("http://localhost/x")
  })

  it("adds a leading slash when missing", () => {
    expect(absoluteUrl("freelancers/abc")).toMatch(/\/freelancers\/abc$/)
  })
})

describe("localizedPath", () => {
  it("returns /<locale>/<path> with leading slash normalized", () => {
    expect(localizedPath("fr", "/agencies")).toBe("/fr/agencies")
    expect(localizedPath("en", "agencies/abc")).toBe("/en/agencies/abc")
  })
})
