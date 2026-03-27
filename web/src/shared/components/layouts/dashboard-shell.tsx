"use client"

import { useState, useEffect } from "react"
import dynamic from "next/dynamic"
import { Sidebar, SIDEBAR_STORAGE_KEY } from "./sidebar"
import { Header } from "./header"
import { cn } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import { useGlobalWS } from "@/shared/hooks/use-global-ws"
import { useCall } from "@/features/call/hooks/use-call"

const ChatWidget = dynamic(
  () =>
    import("../chat-widget/chat-widget").then((m) => ({
      default: m.ChatWidget,
    })),
  { ssr: false },
)

const IncomingCallOverlay = dynamic(
  () =>
    import("@/features/call/components/incoming-call-overlay").then((m) => ({
      default: m.IncomingCallOverlay,
    })),
  { ssr: false },
)

const ActiveCallOverlay = dynamic(
  () =>
    import("@/features/call/components/active-call-overlay").then((m) => ({
      default: m.ActiveCallOverlay,
    })),
  { ssr: false },
)

export function DashboardShell({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [collapsed, setCollapsed] = useState(false)
  const { data: user } = useUser()

  // Call feature — global overlay
  const call = useCall()

  // Maintain a global WS connection so the sidebar unread badge updates
  // in real time on every page, not just on /messages.
  useGlobalWS(user?.id, call.handleCallEvent)

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
            {children}
          </div>
        </main>
      </div>
      <ChatWidget />

      {/* Call overlays */}
      {call.state === "ringing_incoming" && call.incomingCall && (
        <IncomingCallOverlay
          call={call.incomingCall}
          onAccept={call.acceptIncoming}
          onDecline={call.declineIncoming}
        />
      )}

      {(call.state === "active" || call.state === "ringing_outgoing") && (
        <ActiveCallOverlay
          state={call.state}
          recipientName=""
          duration={call.duration}
          isMuted={call.isMuted}
          onToggleMute={call.toggleMute}
          onHangup={call.hangup}
        />
      )}
    </div>
  )
}
