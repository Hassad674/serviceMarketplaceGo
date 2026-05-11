import type { MetadataRoute } from "next"
import { siteConfig } from "@/config/site"

// PERF-W-04 — robots.txt rules. Allow the public marketplace
// surfaces (home + listings + public profiles + opportunities) and
// block every authenticated route + API + auth flow.
//
// `disallow` is a closed list of path PREFIXES — Googlebot treats
// each as "block this and everything below". The list mirrors the
// PROTECTED_PATHS from `src/middleware.ts` plus the auth flow URLs
// that should never appear in search results.
//
// PERF-B: `revalidate = 86400` lets Vercel cache the rendered
// robots.txt for a full day. The content only changes when we ship
// a new build; a longer cache window has no downside.
export const revalidate = 86400

export default function robots(): MetadataRoute.Robots {
  const base = siteConfig.url.replace(/\/$/, "")
  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        disallow: [
          "/api/",
          "/dashboard/",
          "/login",
          "/register",
          "/account",
          "/account/",
          "/billing",
          "/billing/",
          "/wallet",
          "/wallet/",
          "/messages",
          "/messages/",
          "/notifications",
          "/notifications/",
          "/profile",
          "/profile/",
          "/payment-info",
          "/payment-info/",
          "/team",
          "/team/",
          "/invoices",
          "/invoices/",
          "/referral",
          "/referral/",
        ],
      },
    ],
    sitemap: `${base}/sitemap.xml`,
  }
}
