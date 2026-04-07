import { NotificationDropdown } from "./notification-dropdown"

export function Header() {
  return (
    <header className="sticky top-0 z-30 flex h-14 items-center justify-between border-b border-gray-100/50 bg-white/80 px-6 backdrop-blur-xl">
      <div />
      <div className="flex items-center gap-3">
        <NotificationDropdown />
      </div>
    </header>
  )
}
