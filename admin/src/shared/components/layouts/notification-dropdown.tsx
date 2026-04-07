import { useState, useRef, useEffect } from "react"
import { useNavigate } from "react-router-dom"
import { Bell } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import {
  useAdminNotifications,
  useResetNotification,
} from "@/shared/hooks/use-admin-notifications"

type CategoryConfig = {
  key: string
  label: string
  path: string
}

const CATEGORIES: CategoryConfig[] = [
  { key: "reports", label: "Nouveaux signalements", path: "/moderation?source=human_report" },
  { key: "media_rejected", label: "M\u00E9dias auto-rejet\u00E9s", path: "/media?status=rejected" },
  { key: "users_suspended", label: "Utilisateurs suspendus", path: "/users?status=suspended" },
  { key: "messages_hidden", label: "Messages masqu\u00E9s", path: "/moderation?source=auto_text" },
  { key: "media_flagged", label: "M\u00E9dias en attente", path: "/media?status=flagged" },
  { key: "messages_flagged", label: "Messages flagg\u00E9s", path: "/moderation?source=auto_text&severity=flagged" },
  { key: "reviews_flagged", label: "Avis flagg\u00E9s", path: "/reviews?filter=reported" },
]

export function NotificationDropdown() {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()
  const { data } = useAdminNotifications()
  const resetMutation = useResetNotification()

  const counters = data?.data ?? {}
  const totalCount = Object.values(counters).reduce((sum, n) => sum + n, 0)
  const activeCategories = CATEGORIES.filter((c) => (counters[c.key] ?? 0) > 0)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    if (open) document.addEventListener("mousedown", handleClick)
    return () => document.removeEventListener("mousedown", handleClick)
  }, [open])

  function handleCategoryClick(category: CategoryConfig) {
    resetMutation.mutate(category.key)
    setOpen(false)
    navigate(category.path)
  }

  return (
    <div ref={ref} className="relative inline-block">
      <button
        onClick={() => setOpen((v) => !v)}
        className="relative rounded-lg p-2 text-muted-foreground transition-all duration-200 ease-out hover:bg-gray-50 hover:text-foreground"
        aria-label="Notifications"
      >
        <Bell className="h-5 w-5" />
        {totalCount > 0 && (
          <span className="absolute -right-0.5 -top-0.5 inline-flex min-w-[18px] items-center justify-center rounded-full bg-destructive px-1 py-0.5 text-[10px] font-semibold text-white">
            {totalCount > 99 ? "99+" : totalCount}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 z-50 mt-1 w-72 rounded-lg border border-border bg-card py-1 shadow-lg">
          <div className="border-b border-border px-3 py-2">
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              Notifications
            </p>
          </div>

          {activeCategories.length === 0 ? (
            <EmptyNotifications />
          ) : (
            <div className="max-h-72 overflow-y-auto py-1">
              {activeCategories.map((category) => (
                <NotificationItem
                  key={category.key}
                  category={category}
                  count={counters[category.key] ?? 0}
                  onClick={() => handleCategoryClick(category)}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function EmptyNotifications() {
  return (
    <div className="px-3 py-6 text-center text-sm text-muted-foreground">
      Aucune notification
    </div>
  )
}

type NotificationItemProps = {
  category: CategoryConfig
  count: number
  onClick: () => void
}

function NotificationItem({ category, count, onClick }: NotificationItemProps) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-sm",
        "text-foreground transition-colors hover:bg-muted",
      )}
    >
      <span className="truncate">{category.label}</span>
      <span className="inline-flex min-w-[20px] shrink-0 items-center justify-center rounded-full bg-destructive px-1.5 py-0.5 text-[10px] font-semibold text-white">
        {count > 99 ? "99+" : count}
      </span>
    </button>
  )
}
