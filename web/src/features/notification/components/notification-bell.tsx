"use client"

import { useState, useRef, useEffect } from "react"
import { Bell } from "lucide-react"
import { useUnreadNotificationCount } from "../hooks/use-unread-notification-count"
import { NotificationDropdown } from "./notification-dropdown"

import { Button } from "@/shared/components/ui/button"
export function NotificationBell() {
  const { data: count = 0 } = useUnreadNotificationCount()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) {
      document.addEventListener("mousedown", handleClickOutside)
    }
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [open])

  return (
    <div ref={ref} className="relative">
      <Button variant="ghost" size="auto"
        onClick={() => setOpen((prev) => !prev)}
        className="relative rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-slate-700 dark:hover:text-slate-200"
        aria-label="Notifications"
      >
        <Bell className="h-[18px] w-[18px]" strokeWidth={1.5} />
        {count > 0 && (
          <span className="absolute -right-0.5 -top-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-rose-500 px-1 text-[10px] font-bold text-white">
            {count > 99 ? "99+" : count}
          </span>
        )}
      </Button>
      {open && <NotificationDropdown onClose={() => setOpen(false)} />}
    </div>
  )
}
