import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { Star } from "lucide-react"
import { Portrait } from "@/shared/components/ui/portrait"

// LandingReferrers — the dark editorial section "L'innovation
// DesignedTrust · Les apporteurs d'affaires, enfin notés."
//
// Encre background (#2a1f15 via --foreground reversed onto bg), warm
// ivoire body copy, corail italic accent on "enfin". Right column:
// referrer profile card with average score chip and 4 recommended
// freelancers each with star rating.
//
// Server Component.

const ITEMS = [
  {
    portraitId: 1,
    nameKey: "cardItem1Name",
    roleKey: "cardItem1Role",
    forKey: "cardItem1For",
    scoreKey: "cardItem1Score",
  },
  {
    portraitId: 2,
    nameKey: "cardItem2Name",
    roleKey: "cardItem2Role",
    forKey: "cardItem2For",
    scoreKey: "cardItem2Score",
  },
  {
    portraitId: 3,
    nameKey: "cardItem3Name",
    roleKey: "cardItem3Role",
    forKey: "cardItem3For",
    scoreKey: "cardItem3Score",
  },
  {
    portraitId: 5,
    nameKey: "cardItem4Name",
    roleKey: "cardItem4Role",
    forKey: "cardItem4For",
    scoreKey: "cardItem4Score",
  },
] as const

export function LandingReferrers() {
  return (
    <section
      id="referrers"
      aria-labelledby="landing-referrers-title"
      className="mx-auto my-16 w-full max-w-6xl rounded-[28px] bg-foreground px-6 py-14 text-background shadow-[var(--shadow-card-strong)] sm:my-24 sm:px-12 sm:py-20"
    >
      <div className="grid gap-10 lg:grid-cols-2 lg:gap-16">
        <Editorial />
        <ProfileCard />
        <MobileCtas />
      </div>
    </section>
  )
}

function Editorial() {
  const t = useTranslations("landing.referrers")
  return (
    <div>
      <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary">
        {t("eyebrow")}
      </p>
      <h2
        id="landing-referrers-title"
        className="mt-3 font-serif text-[34px] font-medium leading-[1.05] tracking-[-0.02em] sm:text-[48px]"
      >
        <span>{t("titleLead")}</span>{" "}
        <span className="italic text-primary">{t("titleAccent")}</span>{" "}
        <span>{t("titleTrail")}</span>
      </h2>
      <div className="mt-6 space-y-5 text-[14.5px] leading-[1.6] text-background/80 sm:text-[15px]">
        <p>{t("p1")}</p>
        <p>
          <span>{t("p2Prefix")}</span>{" "}
          <strong className="font-semibold text-background">
            {t("p2Highlight")}
          </strong>{" "}
          <span>{t("p2Suffix")}</span>
        </p>
      </div>
      <div className="mt-8 hidden flex-wrap items-center gap-3 sm:flex">
        <Link
          href="/register"
          className="inline-flex h-11 items-center justify-center rounded-full bg-primary px-5 text-sm font-semibold text-primary-foreground shadow-[var(--shadow-message)] transition-colors hover:bg-primary-deep"
        >
          {t("ctaPrimary")}
        </Link>
        <Link
          href="/referrers"
          className="inline-flex h-11 items-center justify-center rounded-full border border-background/20 px-5 text-sm font-semibold text-background transition-colors hover:border-background/40"
        >
          {t("ctaSecondary")}
        </Link>
      </div>
    </div>
  )
}

function ProfileCard() {
  const t = useTranslations("landing.referrers")
  return (
    <div className="self-start rounded-2xl border border-background/10 bg-background/[0.04] p-5 backdrop-blur-sm sm:p-6">
      <div className="flex items-start gap-4">
        <Portrait id={0} size={48} alt={t("cardName")} />
        <div className="min-w-0 flex-1">
          <p className="font-serif text-[18px] font-medium text-background">
            {t("cardName")}
          </p>
          <p className="mt-0.5 text-[12.5px] text-background/70">
            {t("cardRole")}
          </p>
        </div>
        <div className="shrink-0 rounded-2xl bg-primary px-3 py-2 text-center text-primary-foreground">
          <p className="font-serif text-[18px] font-medium leading-none">
            {t("cardScore")}
          </p>
          <p className="mt-1 text-[10px] uppercase tracking-[0.06em] text-primary-foreground/80">
            {t("cardScoreLabel")}
          </p>
        </div>
      </div>
      <p className="mt-5 font-mono text-[10.5px] font-bold uppercase tracking-[0.08em] text-background/60">
        {t("cardListLabel")}
      </p>
      <ul className="mt-3 space-y-2">
        {ITEMS.map((item) => (
          <ProfileCardItem key={item.nameKey} item={item} />
        ))}
      </ul>
      <p className="mt-5 rounded-xl bg-background/[0.06] px-4 py-3 text-[11.5px] leading-[1.55] text-background/70">
        {t("cardFootnote")}
      </p>
    </div>
  )
}

interface ProfileCardItemProps {
  item: (typeof ITEMS)[number]
}

function ProfileCardItem({ item }: ProfileCardItemProps) {
  const t = useTranslations("landing.referrers")
  return (
    <li className="flex items-center gap-3 rounded-xl bg-background/[0.04] px-3 py-2.5">
      <Portrait id={item.portraitId} size={32} alt={t(item.nameKey)} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-semibold text-background">
          <span>{t(item.nameKey)}</span>{" "}
          <span className="font-normal text-background/60">
            · {t(item.roleKey)}
          </span>
        </p>
        <p className="text-[11.5px] text-background/55">{t(item.forKey)}</p>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        <span aria-hidden="true" className="flex">
          {Array.from({ length: 5 }).map((_, i) => (
            <Star
              key={i}
              className="h-3 w-3 fill-primary text-primary"
              strokeWidth={0}
            />
          ))}
        </span>
        <span className="font-mono text-[12.5px] font-medium text-background">
          {t(item.scoreKey)}
        </span>
      </div>
    </li>
  )
}

function MobileCtas() {
  const t = useTranslations("landing.referrers")
  return (
    <div className="flex flex-wrap items-center gap-3 sm:hidden">
      <Link
        href="/register"
        className="inline-flex h-11 flex-1 items-center justify-center rounded-full bg-primary px-5 text-sm font-semibold text-primary-foreground shadow-[var(--shadow-message)] transition-colors hover:bg-primary-deep"
      >
        {t("ctaPrimary")}
      </Link>
      <Link
        href="/referrers"
        className="inline-flex h-11 flex-1 items-center justify-center rounded-full border border-background/20 px-5 text-sm font-semibold text-background transition-colors hover:border-background/40"
      >
        {t("ctaSecondary")}
      </Link>
    </div>
  )
}
