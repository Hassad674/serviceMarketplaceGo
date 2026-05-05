"use client"

// Soleil v2 — topbar notification bell.
//
// Calm circular bell button that flips to a corail-soft tint when the
// dropdown is open. Unread count rendered as a corail badge with white
// text, mirroring the JSX `SoleilTopbar` (soleil.jsx lines 139-154).

import { useState, useRef, useEffect } from "react"
import { useTranslations } from "next-intl"
import { Bell } from "lucide-react"
import { useUnreadNotificationCount } from "../hooks/use-unread-notification-count"
import { NotificationDropdown } from "./notification-dropdown"

export function NotificationBell() {
  const t = useTranslations("notifications")
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
      <button
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        aria-label={t("title")}
        aria-expanded={open}
        aria-haspopup="menu"
        className={`relative flex h-9 w-9 items-center justify-center rounded-full transition-colors duration-150 ${
          open
            ? "bg-[var(--primary-soft)] text-[var(--primary-deep)]"
            : "text-[var(--muted-foreground)] hover:bg-[var(--background)] hover:text-[var(--foreground)]"
        }`}
      >
        <Bell className="h-[18px] w-[18px]" strokeWidth={1.6} />
        {count > 0 && (
          <span
            aria-label={`${count}`}
            className="absolute -right-0.5 -top-0.5 flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-[var(--primary)] px-1 font-mono text-[10px] font-bold leading-none text-white shadow-sm ring-2 ring-[var(--surface)]"
          >
            {count > 99 ? "99+" : count}
          </span>
        )}
      </button>
      {open && <NotificationDropdown onClose={() => setOpen(false)} />}
    </div>
  )
}
