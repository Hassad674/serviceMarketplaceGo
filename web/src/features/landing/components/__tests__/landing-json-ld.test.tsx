import { describe, it, expect } from "vitest"
import { render } from "@testing-library/react"
import { LandingJsonLd } from "../landing-json-ld"

/**
 * landing-json-ld.test.tsx asserts the structured-data payload
 * Googlebot ingests when crawling `/`. Two payloads are emitted:
 *  - Organization: brand identity (Atelier, Paris, founding year)
 *  - WebSite: SearchAction telling Google the in-site search box
 *    is at /freelancers?q={search_term_string}
 *
 * The test parses the rendered <script> blocks back to JSON and
 * checks the schema.org contract verbatim — Google penalises mismatches
 * with the public spec so any drift here would tank rich results.
 */

describe("LandingJsonLd", () => {
  it("renders Organization and WebSite JSON-LD scripts", () => {
    const { container } = render(<LandingJsonLd />)
    const scripts = container.querySelectorAll(
      'script[type="application/ld+json"]',
    )
    expect(scripts).toHaveLength(2)

    const org = JSON.parse(scripts[0].innerHTML)
    expect(org["@context"]).toBe("https://schema.org")
    expect(org["@type"]).toBe("Organization")
    expect(org.name).toBe("Atelier")
    expect(org.address.addressLocality).toBe("Paris")
    expect(org.address.addressCountry).toBe("FR")
    expect(typeof org.url).toBe("string")
    expect(org.url.length).toBeGreaterThan(0)

    const site = JSON.parse(scripts[1].innerHTML)
    expect(site["@type"]).toBe("WebSite")
    expect(site.potentialAction["@type"]).toBe("SearchAction")
    expect(site.potentialAction.target.urlTemplate).toContain(
      "/freelancers?q={search_term_string}",
    )
    // Google requires this string verbatim — locking it.
    expect(site.potentialAction["query-input"]).toBe(
      "required name=search_term_string",
    )
  })

  it("escapes break-out sequences inside the JSON-LD body", () => {
    const { container } = render(<LandingJsonLd />)
    const scripts = container.querySelectorAll(
      'script[type="application/ld+json"]',
    )
    for (const script of Array.from(scripts)) {
      // safeJsonLd substitutes "</" with "</" so a hostile
      // payload cannot terminate the script tag. Verify that no
      // raw "</script" survives the helper.
      expect(script.innerHTML).not.toContain("</script")
    }
  })
})
