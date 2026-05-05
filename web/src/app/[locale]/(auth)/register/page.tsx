import { useTranslations } from "next-intl"
import {
  ArrowRight,
  Building2,
  Check,
  Sparkles,
  type LucideIcon,
} from "lucide-react"
import { Link } from "@i18n/navigation"
import { Portrait } from "@/shared/components/ui/portrait"
import { ThemeToggle } from "@/shared/components/theme-toggle"
import { cn } from "@/shared/lib/utils"

// W-02 · Inscription · choix de rôle — Soleil v2.
//
// Source: design/assets/sources/phase1/soleil-lotE.jsx `SoleilSignupRole`
// (lines 133-238) + design/assets/pdf/web-desktop.pdf p.6.
//
// Visual chrome: top bar with AtelierMark + "Déjà un compte ?" link.
// Centered editorial header (eyebrow + Fraunces display with italic
// corail accent) above a 3-column grid of role cards. Each card is a
// `<Link>` to its dedicated registration sub-route — no client state
// here, the role is "selected" by navigating into the sub-flow.
//
// The 3 cards align on the existing repo flow (Agency / Freelance·Apporteur /
// Enterprise) — the e2e test in `web/e2e/auth.spec.ts` pins these three
// strings + the three sub-routes, so the role taxonomy is unchanged.
// The middle "Freelance" card is highlighted in corail (matches the
// "selected by default" look from the Soleil source) to lead the eye
// toward the most common signup path.
//
// Out-of-scope flagged (NOT shipped this batch):
//   - Apporteur d'affaires as a SEPARATE third role card. The Soleil
//     source shows 3 distinct cards (Freelance + Entreprise + Apporteur),
//     but the repo currently bundles "Apporteur d'affaire" inside the
//     Freelance card (provider role with a `referrer_enabled` toggle).
//     Splitting them requires a backend / register schema change —
//     out of scope for this UI-only batch.
//   - "↳ Étape 1 sur 3" stepper position assumes a 3-step register
//     flow. The repo's freelance flow currently has its own internal
//     steps inside /register/provider — kept the eyebrow as a static
//     "Étape 1 sur 3" matching the design copy.
//   - "Continuer en tant que freelance" CTA at the bottom of the
//     source (drives the selected role forward). Removed because the
//     role cards themselves are the navigation in this repo — adding
//     a second CTA would be redundant and conflict with the existing
//     test contract.
//   - "NOUVEAU" badge on Apporteur card — not shipped (no separate
//     Apporteur card; see first item).

type RoleCard = {
  id: "agency" | "provider" | "enterprise"
  href: "/register/agency" | "/register/provider" | "/register/enterprise"
  eyebrowKey: "W02_eyebrowAgency" | "W02_eyebrowProvider" | "W02_eyebrowEnterprise"
  titleKey: "roleAgency" | "roleFreelance" | "roleEnterprise"
  descKey: "W02_descAgency" | "W02_descProvider" | "W02_descEnterprise"
  bulletKeys: readonly [string, string, string]
  visual: "portrait" | "icon"
  portraitId?: number
  icon?: LucideIcon
  iconGradient?: string
  iconColor?: string
  highlighted: boolean
}

const ROLE_CARDS: readonly RoleCard[] = [
  {
    id: "agency",
    href: "/register/agency",
    eyebrowKey: "W02_eyebrowAgency",
    titleKey: "roleAgency",
    descKey: "W02_descAgency",
    bulletKeys: [
      "W02_agencyBullet1",
      "W02_agencyBullet2",
      "W02_agencyBullet3",
    ],
    visual: "icon",
    icon: Building2,
    iconGradient: "linear-gradient(135deg, #fbf0dc, #fde6ed)",
    iconColor: "text-foreground",
    highlighted: false,
  },
  {
    id: "provider",
    href: "/register/provider",
    eyebrowKey: "W02_eyebrowProvider",
    titleKey: "roleFreelance",
    descKey: "W02_descProvider",
    bulletKeys: [
      "W02_providerBullet1",
      "W02_providerBullet2",
      "W02_providerBullet3",
    ],
    visual: "portrait",
    portraitId: 1,
    highlighted: true,
  },
  {
    id: "enterprise",
    href: "/register/enterprise",
    eyebrowKey: "W02_eyebrowEnterprise",
    titleKey: "roleEnterprise",
    descKey: "W02_descEnterprise",
    bulletKeys: [
      "W02_enterpriseBullet1",
      "W02_enterpriseBullet2",
      "W02_enterpriseBullet3",
    ],
    visual: "icon",
    icon: Sparkles,
    iconGradient: "linear-gradient(135deg, #fde9e3, #fbf0dc)",
    iconColor: "text-primary-deep",
    highlighted: false,
  },
] as const

export default function RegisterPage() {
  const t = useTranslations("auth")
  const tCommon = useTranslations("common")

  return (
    <div className="flex min-h-screen flex-col bg-background">
      {/* Top bar — full width, AtelierMark left, login link right */}
      <header className="flex items-center justify-between px-6 py-7 sm:px-10 lg:px-14">
        <Link
          href="/"
          className="flex items-center gap-2.5 transition-opacity hover:opacity-80"
        >
          <span className="flex h-8 w-8 items-center justify-center rounded-full bg-primary font-serif text-base font-semibold italic text-primary-foreground">
            a
          </span>
          <span className="font-serif text-2xl font-medium tracking-tight text-foreground">
            Atelier
          </span>
        </Link>
        <div className="flex items-center gap-4">
          <p className="hidden text-[13px] text-muted-foreground sm:block">
            {t("alreadyRegistered")}{" "}
            <Link
              href="/login"
              className="font-semibold text-primary transition-colors hover:text-primary-deep"
            >
              {tCommon("signIn")}
            </Link>
          </p>
          <ThemeToggle />
        </div>
      </header>

      {/* Centered editorial content */}
      <div className="mx-auto w-full max-w-[920px] px-6 pb-16 pt-4 text-center sm:px-8">
        <p className="mb-3 font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
          {t("W02_eyebrow")}
        </p>
        <h1 className="font-serif text-[36px] font-normal leading-[1.05] tracking-[-0.025em] text-foreground sm:text-[44px] lg:text-[52px]">
          {t("W02_titlePrefix")}{" "}
          <span className="italic text-primary">{t("W02_titleAccent")}</span>
        </h1>
        <p className="mx-auto mt-3.5 max-w-[540px] text-[15px] leading-[1.55] text-muted-foreground sm:text-[16px]">
          {t("W02_subtitle")}
        </p>

        {/* 3-card grid — middle card highlighted in corail */}
        <div className="mt-11 grid grid-cols-1 gap-3.5 text-left sm:grid-cols-3">
          {ROLE_CARDS.map((card) => (
            <RoleSelectionCard
              key={card.id}
              card={card}
              eyebrow={t(card.eyebrowKey)}
              title={t(card.titleKey)}
              description={t(card.descKey)}
              bullets={card.bulletKeys.map((key) => t(key))}
            />
          ))}
        </div>

        {/* Mobile-only bottom login link (top bar hides it under sm) */}
        <p className="mt-8 text-center text-[13px] text-muted-foreground sm:hidden">
          {t("alreadyRegistered")}{" "}
          <Link
            href="/login"
            className="font-semibold text-primary transition-colors hover:text-primary-deep"
          >
            {tCommon("signIn")}
          </Link>
        </p>
      </div>
    </div>
  )
}

interface RoleSelectionCardProps {
  card: RoleCard
  eyebrow: string
  title: string
  description: string
  bullets: string[]
}

function RoleSelectionCard({
  card,
  eyebrow,
  title,
  description,
  bullets,
}: RoleSelectionCardProps) {
  const Icon = card.icon
  return (
    <Link
      href={card.href}
      className={cn(
        "group relative block rounded-3xl bg-card p-7 transition-all duration-200",
        "hover:-translate-y-0.5",
        card.highlighted
          ? "border-2 border-primary"
          : "border border-border hover:border-border-strong",
      )}
      style={{
        boxShadow: card.highlighted
          ? "0 8px 28px rgba(232, 93, 74, 0.12)"
          : "var(--shadow-card)",
      }}
    >
      {/* Corail check pip top-right (highlighted card only) */}
      {card.highlighted && (
        <span
          className="absolute right-[18px] top-[18px] flex h-[26px] w-[26px] items-center justify-center rounded-full bg-primary text-primary-foreground"
          aria-hidden="true"
        >
          <Check className="h-3.5 w-3.5" strokeWidth={2.5} />
        </span>
      )}

      {/* Centered visual — Portrait for the highlighted card, gradient
          square with icon for the neutral cards. */}
      <div className="mb-4 flex justify-center">
        {card.visual === "portrait" && card.portraitId !== undefined ? (
          <Portrait id={card.portraitId} size={76} rounded="2xl" alt="" />
        ) : Icon ? (
          <span
            className="flex h-[76px] w-[76px] items-center justify-center rounded-2xl"
            style={{ background: card.iconGradient }}
            aria-hidden="true"
          >
            <Icon
              className={cn("h-9 w-9", card.iconColor)}
              strokeWidth={1.5}
            />
          </span>
        ) : null}
      </div>

      <p
        className={cn(
          "mb-1.5 text-center font-mono text-[11px] font-bold uppercase tracking-[0.1em]",
          card.highlighted ? "text-primary-deep" : "text-muted-foreground",
        )}
      >
        {eyebrow}
      </p>
      <h2 className="mb-2 text-center font-serif text-[24px] font-medium leading-tight tracking-[-0.02em] text-foreground">
        {title}
      </h2>
      <p className="mb-[18px] text-center text-[13px] leading-[1.5] text-muted-foreground text-pretty">
        {description}
      </p>

      {/* Bullet checkmarks divider */}
      <div className="flex flex-col gap-2 border-t border-border pt-3.5">
        {bullets.map((bullet) => (
          <div
            key={bullet}
            className="flex items-center gap-2.5 text-[12.5px] text-foreground"
          >
            <span
              className={cn(
                "flex h-4 w-4 shrink-0 items-center justify-center rounded-full",
                card.highlighted
                  ? "bg-primary-soft text-primary"
                  : "bg-background text-muted-foreground",
              )}
              aria-hidden="true"
            >
              <Check className="h-2.5 w-2.5" strokeWidth={2.5} />
            </span>
            {bullet}
          </div>
        ))}
      </div>

      {/* Hover affordance — arrow appears on the highlighted card only */}
      {card.highlighted && (
        <div className="mt-5 flex items-center justify-center gap-1.5 text-[13px] font-medium text-primary opacity-0 transition-opacity duration-200 group-hover:opacity-100">
          <ArrowRight className="h-3.5 w-3.5" strokeWidth={2} />
        </div>
      )}
    </Link>
  )
}
