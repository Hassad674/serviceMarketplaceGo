import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { Link } from "@i18n/navigation"
import { LegalShell } from "@/shared/components/legal/legal-shell"

// /legal — D4 (GDPR Phase C). Sommaire des documents de conformité +
// mentions légales (LCEN art. 6 III). Le contenu "mentions" reste rendu
// ici pour ne pas rompre la route existante référencée depuis le
// LegalFooter; le sommaire des 6 documents D4 vient en complément.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "legal" })
  return {
    title: `${t("docs.indexTitle")} | Marketplace Service`,
    description: t("docs.indexIntro"),
    robots: { index: false, follow: false },
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
      lastUpdatedISO="2026-05-11"
    >
      <div className="space-y-2">
        <h2 className="font-display text-xl text-foreground">
          {t("mentions.editorTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">
          {t("mentions.editorPlaceholder")}
        </p>
      </div>
      <div className="space-y-2">
        <h2 className="font-display text-xl text-foreground">
          {t("mentions.hostingTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">
          {t("mentions.hostingDescription")}
        </p>
      </div>

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
    </LegalShell>
  )
}
