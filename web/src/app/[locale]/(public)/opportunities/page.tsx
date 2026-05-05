import { getTranslations } from "next-intl/server"
import { OpportunityList } from "@/features/job/components/opportunity-list"

// W-12 · Opportunités feed — Soleil v2
//
// Editorial hero with Fraunces serif title (italic corail accent on the
// last word) and ATELIER mono eyebrow, mirrors `SoleilOpportunities`
// in design/assets/sources/phase1/soleil-lotC.jsx (lines 165-171).
// The page itself stays a Server Component — only the list below needs
// client interactivity.
export default async function OpportunitiesPage() {
  const t = await getTranslations("opportunity")

  return (
    <div className="space-y-8">
      <header className="space-y-3">
        <p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
          {t("eyebrow")}
        </p>
        <h1 className="font-serif text-[34px] font-normal leading-[1.1] tracking-[-0.025em] text-foreground sm:text-[42px]">
          {t("heroTitlePart1")}{" "}
          <span className="italic text-primary">{t("heroTitleAccent")}</span>
        </h1>
        <p className="max-w-xl text-[15px] leading-relaxed text-muted-foreground">
          {t("heroSubtitle")}
        </p>
      </header>
      <OpportunityList />
    </div>
  )
}
