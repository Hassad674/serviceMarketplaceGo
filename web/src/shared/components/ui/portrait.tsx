// Portrait — Soleil v2 stylized SVG avatar.
//
// 6 deterministic palettes selected by `id % 6`. Replaces ALL initials,
// emojis, or generic gray placeholders across the marketplace. The SVG
// renders inline so the avatar always paints with zero network overhead.
//
// Reference implementation: design/assets/sources/phase1/soleil.jsx
// lines 27-52. Keep the geometry in lock-step with that file when
// updating — the design language depends on the exact silhouette.

type PortraitRounded = "full" | "sm" | "md" | "lg" | "xl" | "2xl" | number

interface PortraitProps {
  /** Deterministic seed for palette selection (0-5 picked via id % 6). */
  id: number
  /** Pixel size of the avatar — width and height. Default 48. */
  size?: number
  /** Border radius. Default "full" (round). Pass a number for arbitrary px. */
  rounded?: PortraitRounded
  /** Extra class names to compose with the wrapper. */
  className?: string
  /** Optional accessible label. Defaults to a generic French label. */
  alt?: string
}

const PALETTES = [
  { bg: "#fde9e3", skin: "#e8a890", hair: "#3d2618", shirt: "#c43a26" }, // 0 — corail
  { bg: "#e8f2eb", skin: "#d4a584", hair: "#5a3a1f", shirt: "#5a9670" }, // 1 — vert olive
  { bg: "#fde6ed", skin: "#d49a82", hair: "#1a1a1a", shirt: "#c84d72" }, // 2 — rose
  { bg: "#fbf0dc", skin: "#c4926e", hair: "#8b4a1f", shirt: "#d4924a" }, // 3 — ambre
  { bg: "#e8e4f4", skin: "#d8a890", hair: "#2a1f3a", shirt: "#6b5b9a" }, // 4 — lilas
  { bg: "#dfecef", skin: "#c89478", hair: "#3d2818", shirt: "#3a6b7a" }, // 5 — bleu
] as const

const RADIUS_MAP: Record<Exclude<PortraitRounded, number>, string> = {
  full: "9999px",
  sm: "var(--radius-sm)",
  md: "var(--radius-md)",
  lg: "var(--radius-lg)",
  xl: "var(--radius-xl)",
  "2xl": "var(--radius-2xl)",
}

function resolveRadius(rounded: PortraitRounded): string {
  return typeof rounded === "number" ? `${rounded}px` : RADIUS_MAP[rounded]
}

export function Portrait({
  id,
  size = 48,
  rounded = "full",
  className = "",
  alt = "Portrait",
}: PortraitProps) {
  const palette = PALETTES[((id % PALETTES.length) + PALETTES.length) % PALETTES.length]
  const borderRadius = resolveRadius(rounded)

  return (
    <div
      className={className}
      role="img"
      aria-label={alt}
      style={{
        width: size,
        height: size,
        borderRadius,
        background: palette.bg,
        position: "relative",
        overflow: "hidden",
        flexShrink: 0,
      }}
    >
      <svg
        viewBox="0 0 60 60"
        width={size}
        height={size}
        aria-hidden="true"
        style={{ display: "block" }}
      >
        {/* Cou */}
        <rect x="24" y="38" width="12" height="10" fill={palette.skin} />
        {/* Épaules / haut */}
        <path d="M8 60 Q8 46 30 44 Q52 46 52 60 Z" fill={palette.shirt} />
        {/* Tête */}
        <ellipse cx="30" cy="28" rx="11" ry="13" fill={palette.skin} />
        {/* Cheveux */}
        <path
          d="M19 24 Q19 13 30 13 Q41 13 41 24 Q41 21 36 19 Q30 17 24 19 Q19 21 19 28 Z"
          fill={palette.hair}
        />
      </svg>
    </div>
  )
}

/** Number of distinct palettes — exposed for tests and consumers needing the count. */
export const PORTRAIT_PALETTE_COUNT = PALETTES.length
