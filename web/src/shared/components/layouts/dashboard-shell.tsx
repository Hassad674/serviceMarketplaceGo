"use client"

import { useState, useEffect } from "react"
import { Sidebar, SIDEBAR_STORAGE_KEY } from "./sidebar"
import { Header } from "./header"
import { cn } from "@/shared/lib/utils"

export function DashboardShell({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [collapsed, setCollapsed] = useState(false)

  useEffect(() => {
    const stored = localStorage.getItem(SIDEBAR_STORAGE_KEY)
    if (stored === "true") {
      setCollapsed(true)
    }
  }, [])

  function toggleCollapse() {
    const next = !collapsed
    setCollapsed(next)
    localStorage.setItem(SIDEBAR_STORAGE_KEY, String(next))
  }

  return (
    <div className="flex h-screen bg-gray-50/50">
      <Sidebar
        open={sidebarOpen}
        onClose={() => setSidebarOpen(false)}
        collapsed={collapsed}
        onToggleCollapse={toggleCollapse}
      />
      <div
        className={cn(
          "flex flex-1 flex-col overflow-hidden transition-[margin,padding] duration-300",
        )}
      >
        <Header onMenuToggle={() => setSidebarOpen((prev) => !prev)} />
        <main className="flex-1 overflow-y-auto p-5">
          {children}
        </main>
      </div>
    </div>
  )
}
