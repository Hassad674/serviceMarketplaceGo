import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { LegalShell } from "@/shared/components/legal/legal-shell"
import { CookieConsentManageButton } from "@/shared/components/analytics/cookie-consent-manage-button"

// /legal — Mentions légales (LCEN art. 6-III) + sommaire des documents
// de conformité RGPD D4. Indexable (Stripe Restricted Businesses + DSA
// art. 14: les CG, mentions, politique doivent être publiquement
// accessibles aux crawlers).
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal" })
  // No `robots` override — Stripe + DSA + LCEN demand the legal notice
  // is publicly indexable. The earlier `noindex` was a CNIL/Stripe red
  // flag.
  return {
    title: `${t("docs.indexTitle")} | Marketplace Service`,
    description: t("docs.indexIntro"),
    alternates: { canonical: "/legal" },
  }
}

const DOC_LINKS: ReadonlyArray<{ slug: string; key: string }> = [
  { slug: "registre", key: "registre" },
  { slug: "aipd", key: "aipd" },
  { slug: "dpa-template", key: "dpaTemplate" },
  {
    slug: "politique-confidentialite",
    key: "politiqueConfidentialite",
  },
  { slug: "cgu", key: "cgu" },
  { slug: "cgv", key: "cgv" },
  { slug: "code-de-conduite", key: "codeOfConduct" },
]

export default async function LegalIndexPage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal" })

  return (
    <LegalShell
      titleKey="mentions.title"
      introKey="mentions.intro"
      lastUpdatedISO="2026-05-12"
    >
      <EditorBlock />
      <DpoBlock />
      <HostingBlock />
      <PspBlock />
      <ContactBlock />
      <MediationBlock />
      <IpBlock />

      <section className="space-y-4 border-t border-border pt-6">
        <h2 className="font-display text-2xl text-foreground">
          {t("docs.sectionDocs")}
        </h2>
        <p className="text-sm text-muted-foreground">{t("docs.indexIntro")}</p>
        <ul className="grid gap-3 sm:grid-cols-2">
          {DOC_LINKS.map(({ slug, key }) => (
            <li
              key={slug}
              className="rounded-2xl border border-border bg-card p-4"
            >
              <Link
                href={`/legal/${slug}`}
                className="block space-y-2"
                aria-label={t(`docs.items.${key}.title`)}
              >
                <h3 className="font-display text-lg text-foreground">
                  {t(`docs.items.${key}.title`)}
                </h3>
                <p className="text-xs text-muted-foreground">
                  {t(`docs.items.${key}.summary`)}
                </p>
                <p className="text-[11px] uppercase tracking-wide text-accent">
                  {t("docs.labels.reference")} —{" "}
                  {t(`docs.items.${key}.reference`)}
                </p>
              </Link>
            </li>
          ))}
        </ul>
      </section>

      <section className="space-y-2 border-t border-border pt-6">
        <h2 className="font-display text-xl text-foreground">
          {t("mentions.cookieManageTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">
          {t("mentions.cookieManageDescription")}
        </p>
        <CookieConsentManageButton variant="inline" />
      </section>
    </LegalShell>
  )
}

function EditorBlock() {
  const t = useTranslations("legal.mentions")
  return (
    <div className="space-y-2">
      <h2 className="font-display text-xl text-foreground">
        {t("editorTitle")}
      </h2>
      <dl className="grid gap-2 text-sm text-muted-foreground sm:grid-cols-[auto,1fr] sm:gap-x-6">
        <dt className="font-medium text-foreground">
          {t("editorCompanyLabel")}
        </dt>
        <dd>{t("editorCompany")}</dd>
        <dt className="font-medium text-foreground">{t("editorFormLabel")}</dt>
        <dd>{t("editorForm")}</dd>
        <dt className="font-medium text-foreground">
          {t("editorCapitalLabel")}
        </dt>
        <dd>{t("editorCapital")}</dd>
        <dt className="font-medium text-foreground">{t("editorRcsLabel")}</dt>
        <dd>{t("editorRcs")}</dd>
        <dt className="font-medium text-foreground">{t("editorVatLabel")}</dt>
        <dd>{t("editorVat")}</dd>
        <dt className="font-medium text-foreground">
          {t("editorAddressLabel")}
        </dt>
        <dd>{t("editorAddress")}</dd>
        <dt className="font-medium text-foreground">
          {t("editorDirectorLabel")}
        </dt>
        <dd>{t("editorDirector")}</dd>
        <dt className="font-medium text-foreground">
          {t("editorContactLabel")}
        </dt>
        <dd>
          <a
            href="mailto:contact@designedtrust.com"
            className="text-accent underline-offset-4 hover:underline"
          >
            contact@designedtrust.com
          </a>
        </dd>
      </dl>
      <p className="text-xs text-muted-foreground italic">
        {t("editorFallback")}
      </p>
    </div>
  )
}

function DpoBlock() {
  const t = useTranslations("legal.mentions")
  return (
    <div className="space-y-2">
      <h2 className="font-display text-xl text-foreground">{t("dpoTitle")}</h2>
      <p className="text-sm text-muted-foreground">
        {t.rich("dpoBody", {
          email: () => (
            <a
              href="mailto:dpo@designedtrust.com"
              className="text-accent underline-offset-4 hover:underline"
            >
              dpo@designedtrust.com
            </a>
          ),
          cnil: () => (
            <a
              href="https://www.cnil.fr/fr/plaintes"
              target="_blank"
              rel="noopener noreferrer"
              className="text-accent underline-offset-4 hover:underline"
            >
              cnil.fr/fr/plaintes
            </a>
          ),
        })}
      </p>
    </div>
  )
}

function HostingBlock() {
  const t = useTranslations("legal.mentions")
  return (
    <div className="space-y-2">
      <h2 className="font-display text-xl text-foreground">
        {t("hostingTitle")}
      </h2>
      <p className="text-sm text-muted-foreground">{t("hostingFrontend")}</p>
      <p className="text-sm text-muted-foreground">{t("hostingBackend")}</p>
      <p className="text-sm text-muted-foreground">{t("hostingDatabase")}</p>
      <p className="text-sm text-muted-foreground">{t("hostingStorage")}</p>
    </div>
  )
}

function PspBlock() {
  const t = useTranslations("legal.mentions")
  return (
    <div className="space-y-2">
      <h2 className="font-display text-xl text-foreground">{t("pspTitle")}</h2>
      <p className="text-sm text-muted-foreground">{t("pspBody")}</p>
    </div>
  )
}

function ContactBlock() {
  const t = useTranslations("legal.mentions")
  return (
    <div className="space-y-2">
      <h2 className="font-display text-xl text-foreground">
        {t("contactTitle")}
      </h2>
      <p className="text-sm text-muted-foreground">
        {t.rich("contactBody", {
          email: () => (
            <a
              href="mailto:support@designedtrust.com"
              className="text-accent underline-offset-4 hover:underline"
            >
              support@designedtrust.com
            </a>
          ),
        })}
      </p>
    </div>
  )
}

function MediationBlock() {
  const t = useTranslations("legal.mentions")
  return (
    <div className="space-y-2">
      <h2 className="font-display text-xl text-foreground">
        {t("mediationTitle")}
      </h2>
      <p className="text-sm text-muted-foreground">{t("mediationBody")}</p>
    </div>
  )
}

function IpBlock() {
  const t = useTranslations("legal.mentions")
  return (
    <div className="space-y-2">
      <h2 className="font-display text-xl text-foreground">{t("ipTitle")}</h2>
      <p className="text-sm text-muted-foreground">{t("ipBody")}</p>
    </div>
  )
}
