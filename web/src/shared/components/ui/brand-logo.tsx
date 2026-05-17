// BrandLogo — DesignedTrust Services brand identity (Key Hole logo).
//
// Inline SVG (never a bitmap) so the mark stays razor-sharp at every
// size and can be themed/scaled via `className` (set the height with a
// Tailwind `h-*` utility — the SVG keeps its aspect ratio).
//
// Variants:
//   • "mark" — the Key Hole pictogram only (square, "D" + keyhole).
//   • "full" — pictogram + "DesignedTrust Services" wordmark (default).
//
// Geometry is a faithful, minimal recreation of the canonical
// "DesignedTrust — Key Hole" middle (orange) variant: a quarter-round
// "D" with a keyhole punched out in negative (white), beside the
// wordmark "Designed" + "Trust" (Trust in brand orange) + "Services".
//
// BRAND COLOR: #FF7A1F is the DesignedTrust brand orange. It is the
// fixed visual identity of the company — intentionally hardcoded here
// because this file IS the brand asset (a logo), not a themed UI
// surface. No design-system token applies to a brand mark; every other
// component must keep using semantic tokens.
//
// Server Component — pure render, no state, no browser API.

const BRAND_ORANGE = "#FF7A1F"
const BRAND_LABEL = "DesignedTrust Services"

interface BrandLogoProps {
  /** "mark" = pictogram only · "full" = pictogram + wordmark. Default "full". */
  variant?: "full" | "mark"
  /** Tailwind sizing (set height, e.g. `h-7`). Width auto-scales. */
  className?: string
}

// Pictogram — quarter-round "D" with a keyhole in negative. Drawn in a
// 200×200 box (the D spans x:0→180, y:0→200) so the mark sits flush on
// its left edge.
function KeyHoleMark() {
  return (
    <g>
      <path
        d="M0 0 H80 A100 100 0 0 1 180 100 A100 100 0 0 1 80 200 H0 Z"
        fill={BRAND_ORANGE}
      />
      <circle cx="100" cy="80" r="22" fill="#fff" />
      <path d="M86 80 V148 H114 V80 Z" fill="#fff" />
    </g>
  )
}

export function BrandLogo({ variant = "full", className }: BrandLogoProps) {
  if (variant === "mark") {
    return (
      <svg
        viewBox="0 0 200 200"
        className={className}
        role="img"
        aria-label={BRAND_LABEL}
        xmlns="http://www.w3.org/2000/svg"
      >
        <KeyHoleMark />
      </svg>
    )
  }

  // Full lockup: mark (200 wide) + gap + wordmark, on a 760×200 canvas.
  return (
    <svg
      viewBox="0 0 760 200"
      className={className}
      role="img"
      aria-label={BRAND_LABEL}
      xmlns="http://www.w3.org/2000/svg"
    >
      <KeyHoleMark />
      <text
        x="232"
        y="118"
        fontFamily="var(--font-inter-tight), Inter, system-ui, sans-serif"
        fontWeight="800"
        fontSize="92"
        letterSpacing="-3"
        fill="currentColor"
      >
        Designed
        <tspan fill={BRAND_ORANGE}>Trust</tspan>
      </text>
      <text
        x="234"
        y="178"
        fontFamily="var(--font-inter-tight), Inter, system-ui, sans-serif"
        fontWeight="500"
        fontSize="42"
        letterSpacing="6"
        fill="currentColor"
        opacity="0.62"
      >
        SERVICES
      </text>
    </svg>
  )
}
