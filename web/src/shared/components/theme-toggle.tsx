"use client"

import { Moon, Sun } from "lucide-react"
import { useTheme } from "@/shared/hooks/use-theme"

type ThemeToggleProps = {
  className?: string
}

export function ThemeToggle({ className }: ThemeToggleProps) {
  const { theme, toggle } = useTheme()

  return (
    <button
      onClick={toggle}
      className={`p-2 rounded-full bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm border border-gray-200 dark:border-gray-700 shadow-sm hover:shadow-md transition-all ${className ?? ""}`}
      aria-label={theme === "light" ? "Switch to dark mode" : "Switch to light mode"}
    >
      {theme === "light" ? (
        <Moon className="h-4 w-4 text-gray-600" />
      ) : (
        <Sun className="h-4 w-4 text-yellow-500" />
      )}
    </button>
  )
}
