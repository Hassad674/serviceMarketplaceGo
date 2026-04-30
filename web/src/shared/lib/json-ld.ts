/**
 * Safely serialize a JSON-LD payload for embedding inside an inline
 * `<script type="application/ld+json">` tag rendered via React's
 * `dangerouslySetInnerHTML`.
 *
 * `JSON.stringify` does NOT escape sequences that can break out of
 * the surrounding `<script>` block. A user-controlled string containing
 * `</script><script>...` would terminate the JSON-LD tag and execute
 * attacker JS in the page origin — i.e. stored XSS.
 *
 * This helper escapes the four classic break-out sequences:
 *
 *  - `</`            -> `</`         (closing-tag terminator)
 *  - `-->`           -> `-->`        (HTML comment terminator)
 *  - U+2028 (LS)     -> ` `          (raw LS bytes are invalid in JS strings)
 *  - U+2029 (PS)     -> ` `          (same)
 *
 * The escaped string remains valid JSON: parsers reading it back recover
 * the original payload byte-for-byte.
 *
 * Reference: OWASP / React docs on inlining JSON in HTML.
 */
export function safeJsonLd(payload: unknown): string {
  return JSON.stringify(payload)
    .replace(/</g, "\\u003c")
    .replace(/-->/g, "--\\u003e")
    .replace(/\u2028/g, "\\u2028")
    .replace(/\u2029/g, "\\u2029")
}
