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

export function AccountNav({ activeSection, onSectionChange }: AccountNavProps) {
  const t = useTranslations("account")

  return (
    <nav className="rounded-xl border border-slate-200 bg-white p-1.5 dark:border-slate-700 dark:bg-slate-800">
      {/* Desktop: vertical list */}
      <div className="hidden lg:flex lg:flex-col lg:gap-0.5">
        {NAV_ITEMS.map((item) => {
          const isActive = activeSection === item.section
          const Icon = item.icon
          return (
            <Button variant="ghost" size="auto"
              key={item.section}
              onClick={() => onSectionChange(item.section)}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-all",
                isActive
                  ? "bg-rose-50 font-medium text-rose-600 dark:bg-rose-500/10 dark:text-rose-400"
                  : "text-slate-500 hover:bg-slate-50 hover:text-slate-700 dark:text-slate-400 dark:hover:bg-slate-700/50 dark:hover:text-slate-300",
              )}
            >
              {isActive && (
                <div className="absolute left-0 h-6 w-1 rounded-r-full bg-rose-500" />
              )}
              <Icon className="h-[18px] w-[18px]" strokeWidth={1.5} />
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
            <Button variant="ghost" size="auto"
              key={item.section}
              onClick={() => onSectionChange(item.section)}
              className={cn(
                "flex shrink-0 items-center gap-2 rounded-lg px-3 py-2 text-sm transition-all",
                isActive
                  ? "bg-rose-50 font-medium text-rose-600 dark:bg-rose-500/10 dark:text-rose-400"
                  : "text-slate-500 hover:bg-slate-50 dark:text-slate-400 dark:hover:bg-slate-700/50",
              )}
            >
              <Icon className="h-4 w-4" strokeWidth={1.5} />
              <span>{t(item.labelKey)}</span>
            </Button>
          )
        })}
      </div>
    </nav>
  )
}
