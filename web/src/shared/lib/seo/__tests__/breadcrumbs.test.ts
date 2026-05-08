import { describe, it, expect } from "vitest"

import { buildBreadcrumbList } from "../breadcrumbs"

describe("buildBreadcrumbList", () => {
  it("builds a schema.org BreadcrumbList with positional items", () => {
    const out = buildBreadcrumbList([
      { name: "Home", item: "https://example.com" },
      { name: "Freelancers", item: "https://example.com/freelancers" },
      { name: "Alice" },
    ])
    expect(out["@context"]).toBe("https://schema.org")
    expect(out["@type"]).toBe("BreadcrumbList")
    const elements = out.itemListElement as Array<Record<string, unknown>>
    expect(elements).toHaveLength(3)
    expect(elements[0]).toMatchObject({
      "@type": "ListItem",
      position: 1,
      name: "Home",
      item: "https://example.com",
    })
    expect(elements[2]).toMatchObject({
      position: 3,
      name: "Alice",
    })
    // Final crumb (current page) must NOT have an `item` URL — that
    // is the schema.org convention for the trailing breadcrumb.
    expect(elements[2].item).toBeUndefined()
  })

  it("handles a single-item list", () => {
    const out = buildBreadcrumbList([{ name: "Home" }])
    const elements = out.itemListElement as Array<Record<string, unknown>>
    expect(elements).toHaveLength(1)
    expect(elements[0].position).toBe(1)
  })
})
