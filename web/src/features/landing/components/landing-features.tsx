import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

// LandingFeatures — "Pourquoi DesignedTrust · Trois choix qui
// changent tout pour les deux côtés du marché."
//
// Three colored cards (rose pâle, sapin pâle, corail soft) anchor
// the value propositions: 0% commission, audited agencies, rated
// referrers. Each card has an inline-dot tag, Fraunces sub-title,
// and a tabac body copy.
//
// Server Component — pure layout.

const CARDS = [
  {
    tagKey: "card1Tag",
    titleKey: "card1Title",
    bodyKey: "card1Body",
    bg: "bg-pink-soft",
    dot: "bg-pink",
    titleColor: "text-foreground",
  },
  {
    tagKey: "card2Tag",
    titleKey: "card2Title",
    bodyKey: "card2Body",
    bg: "bg-success-soft",
    dot: "bg-success",
    titleColor: "text-foreground",
  },
  {
    tagKey: "card3Tag",
    titleKey: "card3Title",
    bodyKey: "card3Body",
    bg: "bg-primary-soft",
    dot: "bg-primary",
    titleColor: "text-foreground",
  },
] as const

export function LandingFeatures() {
  return (
    <section
      id="how-it-works"
      aria-labelledby="landing-features-title"
      className="mx-auto w-full max-w-6xl px-4 py-16 sm:px-6 sm:py-24"
    >
      <FeaturesHeader />
      <div className="mt-10 grid gap-5 sm:mt-14 sm:grid-cols-1 lg:grid-cols-3 lg:gap-6">
        {CARDS.map((card) => (
          <FeatureCard key={card.tagKey} card={card} />
        ))}
      </div>
    </section>
  )
}

function FeaturesHeader() {
  const t = useTranslations("landing.features")
  return (
    <div className="max-w-3xl">
      <p className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary-deep">
        {t("eyebrow")}
      </p>
      <h2
        id="landing-features-title"
        className="mt-3 font-serif text-[34px] font-medium leading-[1.05] tracking-[-0.02em] text-foreground sm:text-[44px]"
      >
        <span className="block sm:inline">{t("titleLead")}</span>{" "}
        <span className="italic text-primary">{t("titleAccent")}</span>{" "}
        <span>{t("titleTrail")}</span>
      </h2>
    </div>
  )
}

interface FeatureCardProps {
  card: (typeof CARDS)[number]
}

function FeatureCard({ card }: FeatureCardProps) {
  const t = useTranslations("landing.features")
  return (
    <article
      className={cn(
        "rounded-2xl p-6 sm:p-8",
        card.bg,
        "shadow-[var(--shadow-card)]",
      )}
    >
      <div className="inline-flex items-center gap-2 text-[12px] font-semibold text-foreground">
        <span
          aria-hidden="true"
          className={cn("h-1.5 w-1.5 rounded-full", card.dot)}
        />
        <span>{t(card.tagKey)}</span>
      </div>
      <h3
        className={cn(
          "mt-4 font-serif text-[22px] font-medium leading-[1.15] tracking-[-0.01em]",
          card.titleColor,
        )}
      >
        {t(card.titleKey)}
      </h3>
      <p className="mt-3 text-[14px] leading-[1.55] text-muted-foreground">
        {t(card.bodyKey)}
      </p>
    </article>
  )
}
