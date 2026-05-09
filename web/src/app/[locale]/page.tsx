import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LandingHeader } from "@/features/landing/components/landing-header"
import { LandingHero } from "@/features/landing/components/landing-hero"
import { LandingFeatures } from "@/features/landing/components/landing-features"
import { LandingPricing } from "@/features/landing/components/landing-pricing"
import { LandingCredits } from "@/features/landing/components/landing-credits"
import { LandingReferrers } from "@/features/landing/components/landing-referrers"
import { LandingAgencies } from "@/features/landing/components/landing-agencies"
import { LandingCta } from "@/features/landing/components/landing-cta"
import { LandingFooter } from "@/features/landing/components/landing-footer"
import { LandingJsonLd } from "@/features/landing/components/landing-json-ld"
import { buildAlternates, type SupportedLocale } from "@/shared/lib/seo/alternates"

// Public landing page — Soleil v2 direction.
//
// Pure Server Component: pulls i18n translations server-side, renders
// editorial HTML, and ships exactly ONE client island (the search
// bar inside the hero). LCP target < 2.5s on a 3G connection — zero
// JS for the above-the-fold prose, fonts swap-loaded.

type Props = {
  params: Promise<{ locale: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "landing" })
  const alternates = buildAlternates({
    locale: locale as SupportedLocale,
    path: "/",
  })

  return {
    title: t("metaTitle"),
    description: t("metaDescription"),
    alternates,
    openGraph: {
      type: "website",
      title: t("metaTitle"),
      description: t("metaDescription"),
      url: alternates.canonical,
      locale: locale === "fr" ? "fr_FR" : "en_US",
      siteName: "Atelier",
    },
    twitter: {
      card: "summary_large_image",
      title: t("metaTitle"),
      description: t("metaDescription"),
    },
  }
}

export default function HomePage() {
  return (
    <main className="flex min-h-screen flex-col bg-background">
      <LandingJsonLd />
      <LandingHeader />
      <LandingHero />
      <LandingFeatures />
      <LandingPricing />
      <LandingCredits />
      <LandingReferrers />
      <LandingAgencies />
      <LandingCta />
      <LandingFooter />
    </main>
  )
}
