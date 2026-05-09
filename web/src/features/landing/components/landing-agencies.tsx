import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { Star } from "lucide-react"

// LandingAgencies — "Choisir une agence sans croire sur parole"
//
// Two-column section. Left column is a before/after side-by-side
// card pair: AVANT shows a generic agency self-quote with grayed
// fake-review lines; SUR ATELIER shows the same agency with a 4.7
// score and three named brand reviews. Right column tells the
// editorial copy + a black filled CTA.
//
// Server Component.

const AFTER_REVIEWS = [
  { brandKey: "compareAfterReview1", stars: 5 },
  { brandKey: "compareAfterReview2", stars: 5 },
  { brandKey: "compareAfterReview3", stars: 4 },
] as const

export function LandingAgencies() {
  return (
    <section
      id="agencies"
      aria-labelledby="landing-agencies-title"
      className="mx-auto w-full max-w-6xl px-4 py-16 sm:px-6 sm:py-24"
    >
      <div className="grid gap-10 lg:grid-cols-2 lg:gap-16">
        <CompareCards />
        <AgenciesEditorial />
      </div>
    </section>
  )
}

function CompareCards() {
  return (
    <div className="order-2 lg:order-1">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <BeforeCard />
        <AfterCard />
      </div>
    </div>
  )
}

function BeforeCard() {
  const t = useTranslations("landing.agencies")
  return (
    <article className="rounded-2xl border border-border bg-card p-5 shadow-[var(--shadow-card)]">
      <p className="font-mono text-[10.5px] font-bold uppercase tracking-[0.1em] text-subtle-foreground">
        {t("compareBeforeLabel")}
      </p>
      <h3 className="mt-3 font-serif text-[16px] font-medium text-foreground">
        {t("compareBeforeName")}
      </h3>
      <p className="mt-3 font-serif text-[12.5px] italic leading-[1.5] text-muted-foreground">
        {t("compareBeforeQuote")}
      </p>
      <div aria-hidden="true" className="mt-4 space-y-2">
        <span className="block h-1.5 w-3/4 rounded-full bg-border" />
        <span className="block h-1.5 w-1/2 rounded-full bg-border" />
      </div>
      <p className="mt-5 font-serif text-[11.5px] italic leading-[1.5] text-subtle-foreground">
        {t("compareBeforeFootnote")}
      </p>
    </article>
  )
}

function AfterCard() {
  const t = useTranslations("landing.agencies")
  return (
    <article className="rounded-2xl border border-primary/40 bg-primary-soft p-5 shadow-[var(--shadow-card)]">
      <p className="font-mono text-[10.5px] font-bold uppercase tracking-[0.1em] text-primary-deep">
        {t("compareAfterLabel")}
      </p>
      <h3 className="mt-3 font-serif text-[16px] font-medium text-foreground">
        {t("compareAfterName")}
      </h3>
      <div className="mt-3 flex items-baseline gap-2">
        <span aria-hidden="true" className="flex">
          {Array.from({ length: 5 }).map((_, i) => (
            <Star
              key={i}
              className="h-3.5 w-3.5 fill-primary text-primary"
              strokeWidth={0}
            />
          ))}
        </span>
        <span className="font-mono text-[14px] font-medium text-foreground">
          {t("compareAfterScore")}
        </span>
        <span className="font-serif text-[11.5px] italic text-muted-foreground">
          · {t("compareAfterMissions")}
        </span>
      </div>
      <ul className="mt-4 space-y-1.5">
        {AFTER_REVIEWS.map((row) => (
          <AfterReviewRow key={row.brandKey} review={row} />
        ))}
      </ul>
      <p className="mt-5 font-serif text-[11.5px] italic leading-[1.5] text-primary-deep">
        {t("compareAfterFootnote")}
      </p>
    </article>
  )
}

interface AfterReviewRowProps {
  review: (typeof AFTER_REVIEWS)[number]
}

function AfterReviewRow({ review }: AfterReviewRowProps) {
  const t = useTranslations("landing.agencies")
  return (
    <li className="flex items-center justify-between gap-3 text-[12.5px] text-foreground">
      <span>{t(review.brandKey)}</span>
      <span aria-hidden="true" className="flex">
        {Array.from({ length: 5 }).map((_, i) => (
          <Star
            key={i}
            className={
              i < review.stars
                ? "h-3 w-3 fill-primary text-primary"
                : "h-3 w-3 fill-border text-border"
            }
            strokeWidth={0}
          />
        ))}
      </span>
    </li>
  )
}

function AgenciesEditorial() {
  const t = useTranslations("landing.agencies")
  return (
    <div className="order-1 lg:order-2">
      <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary-deep">
        {t("eyebrow")}
      </p>
      <h2
        id="landing-agencies-title"
        className="mt-3 font-serif text-[34px] font-medium leading-[1.05] tracking-[-0.02em] text-foreground sm:text-[44px]"
      >
        <span className="block sm:inline">{t("titleLead")}</span>{" "}
        <span className="italic text-primary">{t("titleAccent")}</span>
      </h2>
      <div className="mt-6 space-y-5 text-[14.5px] leading-[1.6] text-muted-foreground sm:text-[15px]">
        <p>{t("p1")}</p>
        <p>
          <strong className="font-semibold text-foreground">
            {t("p2Highlight")}
          </strong>{" "}
          <span>{t("p2Trail")}</span>
        </p>
      </div>
      <Link
        href="/register"
        className="mt-7 inline-flex h-11 items-center justify-center rounded-full bg-foreground px-6 text-sm font-semibold text-background transition-colors hover:bg-foreground/90"
      >
        {t("cta")}
      </Link>
    </div>
  )
}
