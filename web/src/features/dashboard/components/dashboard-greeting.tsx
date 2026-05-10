"use client"

import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { ArrowRightLeft, Sparkles } from "lucide-react"
import { Button } from "@/shared/components/ui/button"
import { useWorkspace } from "@/shared/hooks/use-workspace"
import { cn } from "@/shared/lib/utils"
import type { DashboardLayout } from "../types"

// DashboardGreeting renders the editorial header — eyebrow line,
// Fraunces display headline, role-specific subtitle, and the
// referrer/freelance switch button (Provider role only). Splitting it
// out of the layout files keeps the role layouts focused on data
// visualisation and avoids three duplicates of the same headline.

interface DashboardGreetingProps {
  layout: DashboardLayout
  /** Already-formatted display name (first name or org name). */
  displayName: string
  /** Whether to show the workspace switch (Provider only). */
  canSwitchWorkspace: boolean
}

export function DashboardGreeting(props: DashboardGreetingProps) {
  const t = useTranslations("dashboard")
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
      <div className="animate-slide-up">
        <p className="mb-2 font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
          {t("welcomeEyebrow")}
        </p>
        <h1 className="font-serif text-[32px] font-normal leading-[1.05] tracking-[-0.025em] text-foreground sm:text-[40px]">
          {t("welcomePrefix", { name: props.displayName })}{" "}
          <span className="italic text-primary">{t("welcomeAccent")}</span>
        </h1>
        <p className="mt-2 max-w-xl text-[14.5px] leading-relaxed text-muted-foreground">
          {t(`${props.layout}Subtitle`)}
        </p>
      </div>
      {props.canSwitchWorkspace ? (
        <WorkspaceSwitchButton layout={props.layout} />
      ) : null}
    </div>
  )
}

function WorkspaceSwitchButton({ layout }: { layout: DashboardLayout }) {
  const t = useTranslations("dashboard")
  const router = useRouter()
  const { switchToReferrer, switchToFreelance } = useWorkspace()
  const isReferrer = layout === "referrer"
  const Icon = isReferrer ? ArrowRightLeft : Sparkles
  const label = isReferrer ? t("freelanceDashboard") : t("businessReferrer")
  return (
    <Button
      variant="ghost"
      size="auto"
      onClick={() => {
        const target = isReferrer ? switchToFreelance() : switchToReferrer()
        router.push(target)
      }}
      className={cn(
        "flex items-center gap-2 rounded-full px-4 py-2 text-sm font-medium",
        "transition-all duration-200",
        isReferrer
          ? "bg-success-soft text-success hover:opacity-80"
          : "gradient-coral text-foreground hover:opacity-90",
      )}
    >
      <Icon className="h-4 w-4" strokeWidth={1.5} aria-hidden />
      {label}
    </Button>
  )
}
