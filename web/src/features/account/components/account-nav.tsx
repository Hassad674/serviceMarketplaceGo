"use client"

import { Bell, Mail, Lock, Shield } from "lucide-react"
import { useTranslations } from "next-intl"
import type { AccountSection } from "../types"
import { cn } from "@/shared/lib/utils"

import { Button } from "@/shared/components/ui/button"

const NAV_ITEMS: { section: AccountSection; icon: React.ElementType; labelKey: string }[] = [
  { section: "notifications", icon: Bell, labelKey: "notifications" },
  { section: "email", icon: Mail, labelKey: "email" },
  { section: "password", icon: Lock, labelKey: "password" },
  { section: "data-and-deletion", icon: Shield, labelKey: "dataAndDeletion" },
]

interface AccountNavProps {
  activeSection: AccountSection
  onSectionChange: (section: AccountSection) => void
}

/**
 * AccountNav — Soleil v2 sidebar tabs (240px on lg+, horizontal pill row
 * on smaller viewports). Active state uses `bg-primary-soft` + corail-deep
 * text per soleil-lotE.jsx `SoleilAccount`.
 */
export function AccountNav({ activeSection, onSectionChange }: AccountNavProps) {
  const t = useTranslations("account")

  return (
    <nav
      aria-label={t("title")}
      className="rounded-2xl border border-border bg-card p-2 shadow-[var(--shadow-card)]"
    >
      {/* Desktop: vertical pill list */}
      <div className="hidden lg:flex lg:flex-col lg:gap-0.5">
        {NAV_ITEMS.map((item) => {
          const isActive = activeSection === item.section
          const Icon = item.icon
          return (
            <Button
              variant="ghost"
              size="auto"
              key={item.section}
              onClick={() => onSectionChange(item.section)}
              className={cn(
                "flex items-center gap-2.5 rounded-xl px-3.5 py-2.5 text-left text-sm transition-colors",
                isActive
                  ? "bg-primary-soft font-semibold text-[var(--primary-deep)] hover:bg-primary-soft"
                  : "font-medium text-foreground hover:bg-[var(--background)]",
              )}
            >
              <Icon
                className={cn(
                  "h-[15px] w-[15px] shrink-0",
                  isActive ? "text-[var(--primary-deep)]" : "text-muted-foreground",
                )}
                strokeWidth={1.6}
                aria-hidden="true"
              />
              <span>{t(item.labelKey)}</span>
            </Button>
          )
        })}
      </div>

      {/* Mobile: horizontal scroll */}
      <div className="flex gap-1 overflow-x-auto lg:hidden">
        {NAV_ITEMS.map((item) => {
          const isActive = activeSection === item.section
          const Icon = item.icon
          return (
            <Button
              variant="ghost"
              size="auto"
              key={item.section}
              onClick={() => onSectionChange(item.section)}
              className={cn(
                "flex shrink-0 items-center gap-2 rounded-xl px-3 py-2 text-sm transition-colors",
                isActive
                  ? "bg-primary-soft font-semibold text-[var(--primary-deep)]"
                  : "font-medium text-foreground hover:bg-[var(--background)]",
              )}
            >
              <Icon className="h-4 w-4" strokeWidth={1.6} aria-hidden="true" />
              <span>{t(item.labelKey)}</span>
            </Button>
          )
        })}
      </div>
    </nav>
  )
}
