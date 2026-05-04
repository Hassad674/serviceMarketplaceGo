"use client"

import { useState } from "react"
import dynamic from "next/dynamic"
import { Sidebar, SIDEBAR_STORAGE_KEY } from "./sidebar"
import { Header } from "./header"
import { cn } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import { KYCBanner } from "@/shared/components/kyc-banner"
import { useGlobalWS } from "@/shared/hooks/use-global-ws"

// CallSlot is the lazy-loading boundary for the call feature
// (PERF-W-01). Importing it directly is safe because CallSlot itself
// does NOT import `livekit-client` — only the inner `CallRuntime`
// does, behind a `dynamic(() => import())`. The 1.3 MB LiveKit chunk
// is loaded only when an incoming call arrives or the user clicks
// "start call" for the first time in the session.
const CallSlot = dynamic(
  () =>
    import("@/features/call/components/call-slot").then((m) => ({
      default: m.CallSlot,
    })),
  { ssr: false, loading: () => null },
)

const ChatWidget = dynamic(
  () =>
    import("../chat-widget/chat-widget").then((m) => ({
      default: m.ChatWidget,
    })),
  { ssr: false },
)

export function DashboardShell({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  // Initial collapsed state is derived from localStorage at mount time
  // via a lazy initializer that bails out on the server (no
  // localStorage). This keeps the dashboard hydration-safe without a
  // setState-in-effect bootstrap.
  const [collapsed, setCollapsed] = useState<boolean>(() => {
    if (typeof window === "undefined") return false
    return window.localStorage.getItem(SIDEBAR_STORAGE_KEY) === "true"
  })
  const { data: user } = useUser()

  // Maintain a global WS connection so the sidebar unread badge updates
  // in real time on every page, not just on /messages. The call event
  // handler is registered lazily by `CallSlot` once it mounts — keeping
  // LiveKit out of the dashboard's eager bundle.
  // `registerCallEventHandler` is identity-stable across renders
  // (memoised via useCallback inside useGlobalWS), so passing it
  // directly to CallSlot is safe.
  const { registerCallEventHandler } = useGlobalWS(user?.id)

  function toggleCollapse() {
    const next = !collapsed
    setCollapsed(next)
    localStorage.setItem(SIDEBAR_STORAGE_KEY, String(next))
  }

  // CallSlot wraps the main tree because downstream features (e.g.
  // messaging) read `useCallContext()` to start outgoing calls. The
  // slot only mounts the LiveKit runtime on demand — until then it
  // costs nothing beyond a thin context provider.
  return (
    <CallSlot registerCallEventHandler={registerCallEventHandler}>
      <div className="flex h-screen bg-background">
        <Sidebar
          open={sidebarOpen}
          onClose={() => setSidebarOpen(false)}
          collapsed={collapsed}
          onToggleCollapse={toggleCollapse}
        />
        <div
          className={cn(
            "flex min-w-0 flex-1 flex-col overflow-hidden transition-all duration-300",
          )}
        >
          <Header onMenuToggle={() => setSidebarOpen((prev) => !prev)} />
          <main className="flex-1 overflow-y-auto px-6 py-8 md:px-9">
            {/* No max-width container: Soleil dashboard layouts are
             * full-bleed inside the sidebar offset. The previous
             * max-w-4xl (896px) was a holdover from the legacy
             * single-column design and made every page feel cramped
             * on 1440px+ screens. Pages that need a narrower reading
             * column (e.g. account settings forms) wrap their own
             * content in a max-width container. */}
            {user && <KYCBanner user={user} />}
            {children}
          </main>
        </div>
        <ChatWidget />
      </div>
    </CallSlot>
  )
}
