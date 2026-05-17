// BrandLogo — DesignedTrust Services brand identity (Key Hole logo).
//
// Inline SVG (never a bitmap) so the mark stays razor-sharp at every
// size and can be scaled via `className` (set the height with a
// Tailwind `h-*` utility — the SVG keeps its aspect ratio).
//
// This is the canonical brand asset provided by the owner. Geometry and
// colours are reproduced verbatim from the supplied SVG — do not
// "improve" it.
//
// Props:
//   • variant — "full" (pictogram + wordmark, default) | "mark"
//     (Key Hole pictogram only, square).
//   • tone — "light" (default, for ivoire/light surfaces) | "dark"
//     (for dark-mode / dark backgrounds). Only the wordmark inks flip;
//     the brand orange and the white keyhole are identical in both.
//
// BRAND COLOURS are intentionally hardcoded here because this file IS
// the brand asset (a logo), not a themed UI surface. No design-system
// token applies to a brand mark; every other component must keep using
// semantic tokens.
//
// Server Component — pure render, no state, no browser API.

const BRAND_ORANGE = "#FF7A1F"
const BRAND_LABEL = "DesignedTrust Services"

// Wordmark inks that depend on the surface (everything else is fixed).
const TONE = {
  light: { designed: "#0E0E12", bar: "#DDDFE5", services: "#5C5C68" },
  dark: { designed: "#FFFBF5", bar: "#3A3A42", services: "#A9A9B4" },
} as const

interface BrandLogoProps {
  /** "mark" = pictogram only · "full" = pictogram + wordmark. Default "full". */
  variant?: "full" | "mark"
  /** "light" for ivoire surfaces (default) · "dark" for dark backgrounds. */
  tone?: "light" | "dark"
  /** Tailwind sizing (set height, e.g. `h-7`). Width auto-scales. */
  className?: string
}

// Pictogram — "D" quarter-round with a keyhole punched in white, plus
// the faint inner highlight stroke. Drawn in its own ~64×64 space
// (path spans x:8→56, y:6→58) so the mark variant crops tight.
function KeyHoleMark() {
  return (
    <g>
      <path
        d="M8 6 H30 A26 26 0 0 1 56 32 A26 26 0 0 1 30 58 H8 Z"
        fill={BRAND_ORANGE}
      />
      <circle cx="34" cy="26" r="6" fill="#fff" />
      <path d="M30 26 L30 44 L38 44 L38 26 Z" fill="#fff" />
      <path
        d="M10 12 L14 12 L14 22"
        stroke="#fff"
        strokeWidth="2"
        strokeOpacity="0.4"
        fill="none"
        strokeLinecap="round"
      />
    </g>
  )
}

export function BrandLogo({
  variant = "full",
  tone = "light",
  className,
}: BrandLogoProps) {
  if (variant === "mark") {
    return (
      <svg
        viewBox="2 0 60 64"
        className={className}
        role="img"
        aria-label={BRAND_LABEL}
        xmlns="http://www.w3.org/2000/svg"
      >
        <KeyHoleMark />
      </svg>
    )
  }

  const c = TONE[tone]
  const wordmarkFont = "var(--font-inter-tight), Inter, system-ui, sans-serif"

  // Full lockup — verbatim artwork from the supplied SVG (mark at
  // translate(220,100) scale(2), wordmark at translate(490,130)). The
  // source viewBox was 0 0 1080 320 but the artwork only spans
  // ~x:236→960, y:112→220 — ~70% of that box was empty padding, which
  // made the logo render tiny at a fixed h-* (most of the height was
  // whitespace). viewBox is cropped to the content bounds (with ~8px
  // margin) so the SAME h-7 now shows the logo ~3× larger. Artwork and
  // coordinates are unchanged — only the visible window is tightened.
  return (
    <svg
      viewBox="225 105 745 122"
      className={className}
      role="img"
      aria-label={BRAND_LABEL}
      xmlns="http://www.w3.org/2000/svg"
    >
      <g transform="translate(220 100) scale(2.0)">
        <KeyHoleMark />
      </g>

      <g transform="translate(490 130)">
        <text
          x="0"
          y="36"
          fontFamily={wordmarkFont}
          fontWeight="900"
          fontSize="62"
          letterSpacing="-2.6"
          fill={c.designed}
        >
          Designed
        </text>
        {/* Mini keyhole (brand orange) between the two words */}
        <g transform="translate(290 16)">
          <circle cx="11" cy="16" r="9" fill={BRAND_ORANGE} />
          <path d="M6 16 L6 40 L16 40 L16 16 Z" fill={BRAND_ORANGE} />
        </g>
        <text
          x="320"
          y="36"
          fontFamily={wordmarkFont}
          fontWeight="900"
          fontSize="62"
          letterSpacing="-2.6"
          fill={BRAND_ORANGE}
        >
          Trust
        </text>

        {/* Sub-brand "SERVICES" with its rule */}
        <g transform="translate(0 64)">
          <rect x="0" y="14" width="42" height="2" fill={c.bar} />
          <text
            x="58"
            y="22"
            fontFamily={wordmarkFont}
            fontWeight="600"
            fontSize="20"
            letterSpacing="2.4"
            fill={c.services}
          >
            SERVICES
          </text>
        </g>
      </g>
    </svg>
  )
}
