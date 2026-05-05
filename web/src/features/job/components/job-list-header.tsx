"use client"

import { Plus } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"

// W-06 — Editorial header for the Mes annonces (entreprise) listing.
// Corail mono eyebrow + Fraunces title with italic corail accent +
// tabac subtitle, "Publier une annonce" corail pill anchored top-right.

interface JobListHeaderProps {
  canCreate: boolean
}

export function JobListHeader({ canCreate }: JobListHeaderProps) {
  const t = useTranslations("job")
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div className="min-w-0">
        <p className="mb-2 font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
          {t("eyebrow")}
        </p>
        <h1 className="font-serif text-[28px] font-medium leading-[1.05] tracking-[-0.02em] text-foreground sm:text-[36px]">
          {t("soleilTitlePrefix")}{" "}
          <span className="italic text-primary">{t("soleilTitleAccent")}</span>
        </h1>
        <p className="mt-2 max-w-2xl text-[14.5px] leading-relaxed text-muted-foreground">
          {t("soleilSubtitle")}
        </p>
      </div>
      {canCreate && (
        <Link
          href="/jobs/create"
          className={cn(
            "inline-flex shrink-0 items-center justify-center gap-2 self-start rounded-full",
            "px-5 py-2.5 text-[13.5px] font-bold text-primary-foreground sm:self-auto",
            "bg-primary transition-all duration-200 ease-out",
            "hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)]",
            "active:scale-[0.98]",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
          )}
        >
          <Plus className="h-4 w-4" strokeWidth={2} />
          {t("createJob")}
        </Link>
      )}
    </div>
  )
}
