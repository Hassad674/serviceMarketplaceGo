import { useTranslations } from "next-intl"

// LandingCredits — "Postuler ne s'achète pas. Ça se mérite."
//
// Two-column: editorial copy on the left explaining the credit
// rules (no purchase, weekly free reset, performance bonus on
// signed missions); a sample credit card on the right with a corail
// progress bar maxed at 14/20 and three line items.
//
// Server Component.

const ROWS = [
  {
    labelKey: "cardRowReset",
    detailKey: "cardRowResetDetail",
    valueKey: "cardRowResetValue",
    tone: "positive",
  },
  {
    labelKey: "cardRowMission1",
    detailKey: "cardRowMissionDetail",
    valueKey: "cardRowMission1Value",
    tone: "positive",
  },
  {
    labelKey: "cardRowMission2",
    detailKey: "cardRowMissionDetail",
    valueKey: "cardRowMission2Value",
    tone: "positive",
  },
  {
    labelKey: "cardRowApplication",
    detailKey: "cardRowApplicationDetail",
    valueKey: "cardRowApplicationValue",
    tone: "negative",
  },
] as const

export function LandingCredits() {
  return (
    <section
      aria-labelledby="landing-credits-title"
      className="mx-auto w-full max-w-6xl px-4 py-16 sm:px-6 sm:py-24"
    >
      <div className="grid gap-10 lg:grid-cols-2 lg:gap-16">
        <CreditsEditorial />
        <CreditsCard />
      </div>
    </section>
  )
}

function CreditsEditorial() {
  const t = useTranslations("landing.credits")
  return (
    <div className="max-w-xl">
      <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary-deep">
        {t("eyebrow")}
      </p>
      <h2
        id="landing-credits-title"
        className="mt-3 font-serif text-[34px] font-medium leading-[1.05] tracking-[-0.02em] text-foreground sm:text-[44px]"
      >
        <span className="block sm:inline">{t("titleLead")}</span>{" "}
        <span className="italic text-primary">{t("titleAccent")}</span>
      </h2>
      <div className="mt-6 space-y-5 text-[14px] leading-[1.6] text-muted-foreground sm:text-[15px]">
        <p>{t("p1")}</p>
        <p>
          <span>{t("p2Lead")}</span>{" "}
          <strong className="font-semibold text-foreground">
            {t("p2Accent")}
          </strong>{" "}
          <span>{t("p2Trail")}</span>
        </p>
        <p>
          <span>{t("p3Lead")}</span>{" "}
          <strong className="font-semibold text-foreground">
            {t("p3Accent")}
          </strong>{" "}
          <span>{t("p3Trail")}</span>
        </p>
      </div>
    </div>
  )
}

function CreditsCard() {
  const t = useTranslations("landing.credits")
  return (
    <aside
      aria-label={t("cardTitle")}
      className="self-start rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)] sm:p-7"
    >
      <CreditsCardHeader />
      <CreditsProgressBar />
      <CreditsRowList />
      <p className="mt-5 rounded-xl bg-success-soft px-4 py-3 text-[12.5px] leading-[1.5] text-success">
        {t("cardFootnote")}
      </p>
    </aside>
  )
}

function CreditsCardHeader() {
  const t = useTranslations("landing.credits")
  return (
    <div className="flex items-start justify-between gap-4">
      <div>
        <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-subtle-foreground">
          {t("cardEyebrow")}
        </p>
        <h3 className="mt-1 font-serif text-[20px] font-medium text-foreground">
          {t("cardTitle")}
        </h3>
      </div>
      <p className="font-serif text-[40px] font-medium leading-none tracking-[-0.02em] text-primary">
        {t("cardCount")}
      </p>
    </div>
  )
}

function CreditsProgressBar() {
  const t = useTranslations("landing.credits")
  return (
    <>
      <div
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={20}
        aria-valuenow={14}
        aria-label={t("cardTitle")}
        className="mt-4 h-1.5 w-full overflow-hidden rounded-full bg-border"
      >
        <div className="h-full w-[70%] rounded-full bg-primary" />
      </div>
      <div className="mt-2 flex items-center justify-between text-[11px] text-muted-foreground">
        <span>{t("cardScaleMin")}</span>
        <span>{t("cardScaleMax")}</span>
      </div>
    </>
  )
}

function CreditsRowList() {
  const t = useTranslations("landing.credits")
  return (
    <ul className="mt-6 divide-y divide-border">
      {ROWS.map((row) => (
        <li
          key={`${row.labelKey}-${row.valueKey}`}
          className="flex items-start justify-between gap-4 py-3 first:pt-0 last:pb-0"
        >
          <div className="min-w-0 flex-1">
            <p className="text-[13.5px] font-medium text-foreground">
              {t(row.labelKey)}
            </p>
            <p className="mt-0.5 text-[12px] text-muted-foreground">
              {t(row.detailKey)}
            </p>
          </div>
          <p
            className={
              row.tone === "positive"
                ? "shrink-0 font-mono text-[14px] font-medium text-success"
                : "shrink-0 font-mono text-[14px] font-medium text-primary-deep"
            }
          >
            {t(row.valueKey)}
          </p>
        </li>
      ))}
    </ul>
  )
}
