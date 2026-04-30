"use client"

import { useState, useEffect } from "react"
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
  const [collapsed, setCollapsed] = useState(false)
  const { data: user } = useUser()

  // Maintain a global WS connection so the sidebar unread badge updates
  // in real time on every page, not just on /messages. The call event
  // handler is registered lazily by `CallSlot` once it mounts — keeping
  // LiveKit out of the dashboard's eager bundle.
  // `registerCallEventHandler` is identity-stable across renders
  // (memoised via useCallback inside useGlobalWS), so passing it
  // directly to CallSlot is safe.
  const { registerCallEventHandler } = useGlobalWS(user?.id)

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

  // CallSlot wraps the main tree because downstream features (e.g.
  // messaging) read `useCallContext()` to start outgoing calls. The
  // slot only mounts the LiveKit runtime on demand — until then it
  // costs nothing beyond a thin context provider.
  return (
    <CallSlot registerCallEventHandler={registerCallEventHandler}>
      <div className="flex h-screen bg-gray-50/50 dark:bg-gray-950">
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
          <main className="flex-1 overflow-y-auto p-5">
            <div className="mx-auto w-full max-w-4xl">
              {user && <KYCBanner user={user} />}
              {children}
            </div>
          </main>
        </div>
        <ChatWidget />
      </div>
    </CallSlot>
  )
}
