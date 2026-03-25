"use client"

import { create } from "zustand"
import { persist } from "zustand/middleware"

type ThemeState = {
  theme: "light" | "dark"
  toggle: () => void
  setTheme: (theme: "light" | "dark") => void
}

export const useTheme = create<ThemeState>()(
  persist(
    (set, get) => ({
      theme: "light",
      toggle: () => {
        const next = get().theme === "light" ? "dark" : "light"
        set({ theme: next })
        document.documentElement.classList.toggle("dark", next === "dark")
      },
      setTheme: (theme) => {
        set({ theme })
        document.documentElement.classList.toggle("dark", theme === "dark")
      },
    }),
    { name: "marketplace-theme" },
  ),
)
