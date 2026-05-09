import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"

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
  { labelKey: "understandCredits", href: "#" },
] as const

const COMPANY_LINKS = [
  { labelKey: "companyAbout", href: "/" },
  { labelKey: "companyContact", href: "/" },
] as const

const LEGAL_LINKS = [
  { labelKey: "legalTerms", href: "/" },
  { labelKey: "legalPrivacy", href: "/" },
  { labelKey: "legalNotices", href: "/" },
  { labelKey: "legalCookies", href: "/" },
] as const

const MOBILE_LINKS = [
  { labelKey: "understandHowItWorks", href: "#how-it-works" },
  { labelKey: "productPricing", href: "#pricing" },
  { labelKey: "legalHelp", href: "/" },
  { labelKey: "legalTerms", href: "/" },
  { labelKey: "legalPrivacy", href: "/" },
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
      <div className="grid gap-10 border-t border-border pt-12 lg:grid-cols-[1.4fr_repeat(4,1fr)]">
        <FooterBrand />
        <FooterColumn labelKey="productLabel" links={PRODUCT_LINKS} />
        <FooterColumn labelKey="understandLabel" links={UNDERSTAND_LINKS} />
        <FooterColumn labelKey="companyLabel" links={COMPANY_LINKS} />
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
  const t = useTranslations("landing.footer")
  const year = new Date().getFullYear()
  return (
    <div className="mt-10 flex items-center justify-between border-t border-border pt-6 text-[12.5px] text-muted-foreground">
      <p>{t("copyright", { year })}</p>
      <ul className="flex items-center gap-6">
        <li>
          <SocialLink href="https://www.linkedin.com" labelKey="social.linkedin" />
        </li>
        <li>
          <SocialLink href="https://x.com" labelKey="social.x" />
        </li>
        <li>
          <SocialLink
            href="https://www.instagram.com"
            labelKey="social.instagram"
          />
        </li>
      </ul>
    </div>
  )
}

function SocialLink({ href, labelKey }: { href: string; labelKey: string }) {
  const t = useTranslations("landing.footer")
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer noopener"
      className="transition-colors hover:text-foreground"
    >
      {t(labelKey)}
    </a>
  )
}

function MobileFooter() {
  const t = useTranslations("landing.footer")
  const year = new Date().getFullYear()
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
      <p className="mt-6 text-[12px] text-muted-foreground">
        {t("copyright", { year })}
      </p>
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
  if (link.href.startsWith("#")) {
    return (
      <a href={link.href} className="transition-colors hover:text-foreground">
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
