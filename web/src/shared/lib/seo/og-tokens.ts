/**
 * og-tokens.ts — Soleil v2 design tokens duplicated for the OG image
 * generator.
 *
 * Why duplicate? `next/og` runs in the Edge runtime where Tailwind and
 * `@theme` CSS variables are unavailable — the renderer accepts a
 * subset of inline CSS only. Keeping the hex values in one source of
 * truth here (mirroring `design/DESIGN_SYSTEM.md`) means a palette
 * change still requires a deliberate edit, and the values stay
 * synchronized through code review rather than runtime CSS resolution.
 */

export const OG_PALETTE = {
  /** ivoire — page background */
  bg: "#fffbf5",
  /** blanc pur — card surface */
  surface: "#ffffff",
  /** sable clair — soft border */
  border: "#f0e6d8",
  /** encre — primary text */
  text: "#2a1f15",
  /** tabac — secondary text */
  textMute: "#7a6850",
  /** corail — accent / CTA */
  accent: "#e85d4a",
  /** rose pâle — soft background */
  accentSoft: "#fde9e3",
  /** corail foncé — gradient stop */
  accentDeep: "#c43a26",
  /** sapin — success */
  green: "#5a9670",
} as const

/** OG image canonical dimensions for OpenGraph + Twitter cards. */
export const OG_DIMENSIONS = { width: 1200, height: 630 } as const

/**
 * Soleil v2 signature gradient used as the OG image background. Goes
 * from corail-deep (top-left) through corail (mid) to ivoire (bottom-
 * right) — the same gradient direction used on the homepage hero.
 */
export const OG_GRADIENT = `linear-gradient(135deg, ${OG_PALETTE.accentDeep} 0%, ${OG_PALETTE.accent} 38%, ${OG_PALETTE.accentSoft} 78%, ${OG_PALETTE.bg} 100%)`
