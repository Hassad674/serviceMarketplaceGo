/**
 * og-profile.tsx — branded OpenGraph image renderer for the public
 * profile pages (freelancers, agencies, referrers).
 *
 * Uses `next/og` `ImageResponse` which runs in the Edge runtime. The
 * renderer accepts a constrained subset of CSS (flex layout, basic
 * typography, no @media, no Tailwind tokens) — see Vercel docs on
 * the `next/og` capabilities matrix.
 *
 * Layout (1200x630, Soleil v2):
 *   ┌────────────────────────────────────────────────┐
 *   │ Marketplace Service                            │
 *   │                                                │
 *   │ ┌──────┐  Display name (2 lines max)           │
 *   │ │ pic  │  Title — City                         │
 *   │ │      │  ★ 4.9 (12 avis) — Pro freelance      │
 *   │ └──────┘                                       │
 *   │                                                │
 *   │ marketplace-service.com                        │
 *   └────────────────────────────────────────────────┘
 *
 * The photo is rendered as an `<img>` so user uploads (R2/MinIO) work
 * without re-encoding. Falls back to a stylized initial bubble when
 * no photo is configured.
 */

import { ImageResponse } from "next/og"
import { OG_PALETTE, OG_DIMENSIONS, OG_GRADIENT } from "./og-tokens"

export type OgProfileRole = "freelance" | "agency" | "referrer"

export interface OgProfileInput {
  displayName: string
  title?: string
  city?: string
  /** Absolute photo URL (https). When omitted we render the initial bubble. */
  photoUrl?: string
  /** Localized rating line, e.g. "★ 4.9 (12 avis)". Omit when no rating. */
  ratingLine?: string
  /** Localized role tag, e.g. "Freelance Pro" / "Agence Pro". */
  roleTag: string
  /** Localized site name displayed at the top. */
  siteName: string
  /** Localized footer URL display, e.g. "marketplace-service.com". */
  footerLabel: string
}

/**
 * renderProfileOgImage returns the `next/og` ImageResponse for the
 * profile OG card. Caller is responsible for bridging the page-level
 * data (profile + average rating + locale) into this view-model.
 */
export function renderProfileOgImage(input: OgProfileInput): ImageResponse {
  const { width, height } = OG_DIMENSIONS
  const initial = (input.displayName.trim()[0] ?? "?").toUpperCase()

  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          background: OG_GRADIENT,
          padding: "64px 80px",
          fontFamily: "sans-serif",
          color: OG_PALETTE.text,
          position: "relative",
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            fontSize: 26,
            fontWeight: 600,
            color: OG_PALETTE.surface,
            letterSpacing: "0.01em",
          }}
        >
          <span style={{ display: "flex" }}>{input.siteName}</span>
          <span
            style={{
              display: "flex",
              padding: "8px 18px",
              borderRadius: 999,
              background: "rgba(255, 255, 255, 0.18)",
              fontSize: 22,
              fontWeight: 500,
            }}
          >
            {input.roleTag}
          </span>
        </div>

        <div
          style={{
            flex: 1,
            display: "flex",
            alignItems: "center",
            gap: 56,
            marginTop: 44,
            background: OG_PALETTE.surface,
            borderRadius: 32,
            padding: "56px 60px",
            boxShadow: "0 30px 60px rgba(42, 31, 21, 0.18)",
          }}
        >
          {input.photoUrl ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={input.photoUrl}
              alt=""
              width={220}
              height={220}
              style={{
                width: 220,
                height: 220,
                borderRadius: "50%",
                objectFit: "cover",
                border: `6px solid ${OG_PALETTE.accentSoft}`,
                flexShrink: 0,
              }}
            />
          ) : (
            <div
              style={{
                width: 220,
                height: 220,
                borderRadius: "50%",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                background: OG_PALETTE.accentSoft,
                color: OG_PALETTE.accentDeep,
                fontSize: 110,
                fontWeight: 600,
                flexShrink: 0,
              }}
            >
              {initial}
            </div>
          )}

          <div
            style={{
              display: "flex",
              flexDirection: "column",
              gap: 18,
              flex: 1,
              minWidth: 0,
            }}
          >
            <div
              style={{
                display: "flex",
                fontSize: 64,
                fontWeight: 700,
                lineHeight: 1.05,
                color: OG_PALETTE.text,
                letterSpacing: "-0.01em",
              }}
            >
              {truncate(input.displayName, 60)}
            </div>
            {input.title ? (
              <div
                style={{
                  display: "flex",
                  fontSize: 32,
                  color: OG_PALETTE.textMute,
                  lineHeight: 1.3,
                }}
              >
                {[truncate(input.title, 70), input.city]
                  .filter(Boolean)
                  .join(" — ")}
              </div>
            ) : input.city ? (
              <div
                style={{
                  display: "flex",
                  fontSize: 32,
                  color: OG_PALETTE.textMute,
                }}
              >
                {input.city}
              </div>
            ) : null}
            {input.ratingLine ? (
              <div
                style={{
                  display: "flex",
                  fontSize: 28,
                  color: OG_PALETTE.accent,
                  fontWeight: 600,
                  marginTop: 6,
                }}
              >
                {input.ratingLine}
              </div>
            ) : null}
          </div>
        </div>

        <div
          style={{
            display: "flex",
            justifyContent: "flex-end",
            color: OG_PALETTE.surface,
            fontSize: 22,
            opacity: 0.85,
            marginTop: 32,
          }}
        >
          {input.footerLabel}
        </div>
      </div>
    ),
    {
      width,
      height,
    },
  )
}

function truncate(value: string, max: number): string {
  if (value.length <= max) return value
  return `${value.slice(0, max - 1).trimEnd()}…`
}
