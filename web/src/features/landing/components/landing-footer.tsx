import { Heart, Linkedin } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { legalHref, legalPathnames } from "@i18n/routing"

// Author signature — the maintainer's public LinkedIn. Hardcoded on
// purpose: it is a fixed personal profile URL, not an env-driven or
// tenant-specific value.
const AUTHOR_LINKEDIN_URL =
  "https://www.linkedin.com/in/hassad-s-24714929b/"

// LandingFooter — site footer.
//
// Desktop layout (mockup page 5): logo + 60-char tagline on the
// left, four columns (Produit / Comprendre / Entreprise / Légal),
// then a bottom row with copyright + social handles.
//
// Mobile layout (mockup page 4): logo + short tagline, single row
// of 4 inline links, then copyright. The dense column grid is hidden
// to keep the foot simple on small screens — content stays
// accessible via the dashboard once signed in.
//
// Server Component. Anchors point to in-page sections or sensible
// existing routes (the manifesto / contact pages may not exist —
// they are still labelled per the mockup; clicks land on `/` if the
// destination is missing, which the user can flag as out-of-scope).

const PRODUCT_LINKS = [
  { labelKey: "productEnterprises", href: "/register" },
  { labelKey: "productFreelances", href: "/freelancers" },
  { labelKey: "productReferrers", href: "/referrers" },
  { labelKey: "productAgencies", href: "/agencies" },
  { labelKey: "productPricing", href: "#pricing" },
] as const

const UNDERSTAND_LINKS = [
  { labelKey: "understandHowItWorks", href: "#how-it-works" },
  { labelKey: "understandManifesto", href: "#referrers" },
  { labelKey: "understandSecurity", href: "#pricing" },
] as const

const LEGAL_LINKS = [
  { labelKey: "legalTerms", href: "/legal/cgu" },
  { labelKey: "legalPrivacy", href: "/legal/politique-confidentialite" },
  { labelKey: "legalNotices", href: "/legal" },
  { labelKey: "legalCookies", href: "/cookies" },
] as const

const MOBILE_LINKS = [
  { labelKey: "understandHowItWorks", href: "#how-it-works" },
  { labelKey: "productPricing", href: "#pricing" },
  { labelKey: "legalHelp", href: "/legal" },
  { labelKey: "legalTerms", href: "/legal/cgu" },
  { labelKey: "legalPrivacy", href: "/legal/politique-confidentialite" },
] as const

export function LandingFooter() {
  return (
    <footer
      role="contentinfo"
      className="mx-auto w-full max-w-6xl px-4 pb-12 pt-8 sm:px-6 sm:pb-16"
    >
      <DesktopFooter />
      <MobileFooter />
    </footer>
  )
}

function DesktopFooter() {
  return (
    <div className="hidden sm:block">
      <div className="grid gap-10 border-t border-border pt-12 lg:grid-cols-[1.4fr_repeat(3,1fr)]">
        <FooterBrand />
        <FooterColumn labelKey="productLabel" links={PRODUCT_LINKS} />
        <FooterColumn labelKey="understandLabel" links={UNDERSTAND_LINKS} />
        <FooterColumn labelKey="legalLabel" links={LEGAL_LINKS} />
      </div>
      <FooterBottom />
    </div>
  )
}

function FooterBrand() {
  const t = useTranslations("landing.footer")
  return (
    <div>
      <Link
        href="/"
        className="inline-flex items-center font-serif text-2xl font-medium tracking-[-0.02em] text-foreground"
      >
        <span>Atelier</span>
        <span aria-hidden="true" className="ml-[2px] text-primary">
          .
        </span>
      </Link>
      <p className="mt-4 max-w-xs text-[13.5px] leading-[1.55] text-muted-foreground">
        {t("tagline")}
      </p>
    </div>
  )
}

function FooterBottom() {
  return (
    <div className="mt-10 flex items-center justify-between border-t border-border pt-6 text-[12.5px] text-muted-foreground">
      <AuthorSignature />
      <AuthorLinkedInLink />
    </div>
  )
}

// AuthorSignature — "DesignedTrust Services {year} — made with ❤️ by
// Hassad Smara". The heart is a real lucide glyph (Soleil corail
// accent) injected via t.rich so the copy stays fully i18n-keyed.
function AuthorSignature() {
  const t = useTranslations("landing.footer")
  const year = new Date().getFullYear()
  return (
    <p className="inline-flex flex-wrap items-center gap-x-1">
      {t.rich("madeBy", {
        year,
        heart: () => (
          <Heart
            className="inline size-3.5 fill-accent text-accent align-[-2px]"
            aria-hidden="true"
          />
        ),
      })}
    </p>
  )
}

function AuthorLinkedInLink() {
  const t = useTranslations("landing.footer")
  return (
    <a
      href={AUTHOR_LINKEDIN_URL}
      target="_blank"
      rel="noreferrer noopener"
      aria-label={t("authorLinkedInAria")}
      className="transition-colors hover:text-foreground"
    >
      <Linkedin className="size-4" aria-hidden="true" />
    </a>
  )
}

function MobileFooter() {
  const t = useTranslations("landing.footer")
  return (
    <div className="block border-t border-border pt-10 sm:hidden">
      <Link
        href="/"
        className="inline-flex items-center font-serif text-2xl font-medium tracking-[-0.02em] text-foreground"
      >
        <span>Atelier</span>
        <span aria-hidden="true" className="ml-[2px] text-primary">
          .
        </span>
      </Link>
      <p className="mt-3 max-w-xs text-[13px] leading-[1.55] text-muted-foreground">
        {t("mobileTagline")}
      </p>
      <ul className="mt-6 flex flex-wrap items-center gap-x-5 gap-y-2 text-[13.5px] text-foreground">
        {MOBILE_LINKS.map((link) => (
          <li key={link.labelKey}>
            <FooterLink link={link} />
          </li>
        ))}
      </ul>
      <div className="mt-6 flex items-center justify-between text-[12px] text-muted-foreground">
        <AuthorSignature />
        <AuthorLinkedInLink />
      </div>
    </div>
  )
}

interface FooterColumnProps {
  labelKey: string
  links: readonly { labelKey: string; href: string }[]
}

function FooterColumn({ labelKey, links }: FooterColumnProps) {
  const t = useTranslations("landing.footer")
  return (
    <div>
      <p className="text-[13.5px] font-semibold text-foreground">
        {t(labelKey)}
      </p>
      <ul className="mt-4 space-y-3 text-[13px] text-muted-foreground">
        {links.map((link) => (
          <li key={link.labelKey}>
            <FooterLink link={link} />
          </li>
        ))}
      </ul>
    </div>
  )
}

function FooterLink({
  link,
}: {
  link: { labelKey: string; href: string }
}) {
  const t = useTranslations("landing.footer")
  const locale = useLocale()
  if (link.href.startsWith("#")) {
    return (
      <a href={link.href} className="transition-colors hover:text-foreground">
        {t(link.labelKey)}
      </a>
    )
  }
  // Legal routes have locale-aware URL segments (FR slugs canonical,
  // EN-named on EN locale). We fall back to next-intl Link for every
  // other route so client-side prefetch + smart navigation still kick
  // in for product / understand / company links.
  if (link.href in legalPathnames) {
    return (
      <a
        href={legalHref(link.href, locale)}
        className="transition-colors hover:text-foreground"
      >
        {t(link.labelKey)}
      </a>
    )
  }
  return (
    <Link href={link.href} className="transition-colors hover:text-foreground">
      {t(link.labelKey)}
    </Link>
  )
}
