import { Bell } from "lucide-react"

export function Header() {
  return (
    <header className="sticky top-0 z-30 flex h-14 items-center justify-between border-b border-gray-100/50 bg-white/80 px-6 backdrop-blur-xl">
      <div />
      <div className="flex items-center gap-3">
        <button
          className="rounded-lg p-2 text-muted-foreground transition-all duration-200 ease-out hover:bg-gray-50 hover:text-foreground"
          aria-label="Notifications"
        >
          <Bell className="h-5 w-5" />
        </button>
      </div>
    </header>
  )
}
