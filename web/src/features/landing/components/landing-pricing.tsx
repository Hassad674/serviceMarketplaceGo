import { useTranslations } from "next-intl"

// LandingPricing — two-column section. Left column tells the
// "0 % de commission, la même protection" story with a 3-step
// numbered list; right column shows the flat-fee table on a soft
// ivoire card with the corail accent on amount cells.
//
// Server Component.

const STEPS = [
  { titleKey: "step1Title", bodyKey: "step1Body" },
  { titleKey: "step2Title", bodyKey: "step2Body" },
  { titleKey: "step3Title", bodyKey: "step3Body" },
] as const

const ROWS = [
  {
    labelKey: "rowClient",
    detailKey: "rowClientDetail",
    amountKey: "rowClientAmount",
    accent: "success",
  },
  {
    labelKey: "rowReferrer",
    detailKey: "rowReferrerDetail",
    amountKey: "rowReferrerAmount",
    accent: "success",
  },
  {
    labelKey: "rowFreelance",
    detailKey: "rowFreelanceDetail",
    amountKey: "rowFreelanceAmount",
    accent: "primary",
  },
  {
    labelKey: "rowAgency",
    detailKey: "rowAgencyDetail",
    amountKey: "rowAgencyAmount",
    accent: "primary",
  },
] as const

export function LandingPricing() {
  return (
    <section
      id="pricing"
      aria-labelledby="landing-pricing-title"
      className="mx-auto w-full max-w-6xl px-4 py-16 sm:px-6 sm:py-24"
    >
      <div className="grid gap-10 lg:grid-cols-2 lg:gap-16">
        <PricingStory />
        <PricingTable />
      </div>
    </section>
  )
}

function PricingStory() {
  const t = useTranslations("landing.pricing")
  return (
    <div>
      <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary-deep">
        {t("eyebrow")}
      </p>
      <h2
        id="landing-pricing-title"
        className="mt-3 font-serif text-[34px] font-medium leading-[1.05] tracking-[-0.02em] text-foreground sm:text-[44px]"
      >
        <span className="block sm:inline">{t("titleLead")}</span>{" "}
        <span className="italic text-primary">{t("titleAccent")}</span>
      </h2>
      <p className="mt-5 max-w-xl text-[15px] leading-[1.55] text-muted-foreground">
        {t("intro")}
      </p>
      <ol className="mt-8 space-y-6">
        {STEPS.map((step, idx) => (
          <PricingStep key={step.titleKey} step={step} index={idx} />
        ))}
      </ol>
    </div>
  )
}

interface PricingStepProps {
  step: (typeof STEPS)[number]
  index: number
}

function PricingStep({ step, index }: PricingStepProps) {
  const t = useTranslations("landing.pricing")
  return (
    <li className="flex gap-5">
      <span
        aria-hidden="true"
        className="font-serif text-[22px] font-medium tracking-[-0.01em] text-primary"
      >
        {String(index + 1).padStart(2, "0")}
      </span>
      <div>
        <h3 className="font-serif text-[18px] font-medium text-foreground">
          {t(step.titleKey)}
        </h3>
        <p className="mt-1 max-w-md text-[14px] leading-[1.55] text-muted-foreground">
          {t(step.bodyKey)}
        </p>
      </div>
    </li>
  )
}

function PricingTable() {
  const t = useTranslations("landing.pricing")
  return (
    <aside
      aria-label={t("tableEyebrow")}
      className="rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)] sm:p-8"
    >
      <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-subtle-foreground">
        {t("tableEyebrow")}
      </p>
      <h3 className="mt-3 font-serif text-[28px] font-medium leading-[1.1] tracking-[-0.02em] text-foreground sm:text-[32px]">
        <span>{t("tableTitleLead")}</span>{" "}
        <span className="italic text-primary">{t("tableTitleAccent")}</span>{" "}
        <span>{t("tableTitleTrail")}</span>
      </h3>
      <ul className="mt-6 divide-y divide-border">
        {ROWS.map((row) => (
          <PricingRow key={row.labelKey} row={row} />
        ))}
      </ul>
      <p className="mt-6 rounded-xl bg-primary-soft px-4 py-3 text-[12.5px] leading-[1.55] text-primary-deep">
        {t("footnote")}
      </p>
    </aside>
  )
}

interface PricingRowProps {
  row: (typeof ROWS)[number]
}

function PricingRow({ row }: PricingRowProps) {
  const t = useTranslations("landing.pricing")
  return (
    <li className="flex items-start justify-between gap-4 py-4 first:pt-0 last:pb-0">
      <div className="min-w-0 flex-1">
        <p className="text-[14px] font-semibold text-foreground">
          {t(row.labelKey)}
        </p>
        <p className="mt-1 text-[12.5px] leading-[1.5] text-muted-foreground">
          {t(row.detailKey)}
        </p>
      </div>
      <p
        className={
          row.accent === "success"
            ? "shrink-0 font-mono text-[15px] font-medium text-success"
            : "shrink-0 font-mono text-[15px] font-medium text-primary-deep"
        }
      >
        {t(row.amountKey)}
      </p>
    </li>
  )
}
