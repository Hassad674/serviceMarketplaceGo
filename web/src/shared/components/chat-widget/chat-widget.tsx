"use client"

import dynamic from "next/dynamic"
import { MessageSquare, ChevronUp } from "lucide-react"
import { useTranslations } from "next-intl"
import { usePathname } from "next/navigation"
import { cn } from "@/shared/lib/utils"
import { useUnreadCount } from "@/shared/hooks/use-unread-count"
import { useMediaQuery } from "@/shared/hooks/use-media-query"
import { useChatWidget } from "./use-chat-widget"

import { Button } from "@/shared/components/ui/button"
const ChatWidgetPanel = dynamic(
  () =>
    import("./chat-widget-panel").then((m) => ({
      default: m.ChatWidgetPanel,
    })),
  { ssr: false },
)

export function ChatWidget() {
  const t = useTranslations("messaging")
  const pathname = usePathname()
  const isDesktop = useMediaQuery("(min-width: 1024px)")
  const { data: unreadData } = useUnreadCount()
  const unreadCount = unreadData?.count ?? 0

  const {
    isOpen,
    view,
    activeConversationId,
    pendingRecipient,
    open,
    close,
    selectConversation,
    goBack,
    resolvePendingConversation,
  } = useChatWidget()

  // Hide on /messages pages (any locale prefix)
  if (pathname.includes("/messages")) return null

  // Hide bar on mobile — the navbar badge is enough
  if (!isDesktop) return null

  function handleToggle() {
    if (isOpen) {
      close()
    } else {
      open()
    }
  }

  return (
    <>
      {/* Contra-style bottom bar (closed state) or panel header (open state) */}
      {!isOpen && (
        <Button variant="ghost" size="auto"
          onClick={handleToggle}
          className={cn(
            "fixed bottom-0 right-6 z-50 flex h-12 w-[320px] items-center gap-2.5 px-4",
            "rounded-t-xl border border-b-0 border-gray-200 bg-white shadow-lg",
            "transition-all duration-200 hover:shadow-xl",
            "dark:border-gray-700 dark:bg-gray-900",
          )}
          aria-label={t("openChat")}
        >
          <MessageSquare
            className="h-[18px] w-[18px] text-gray-700 dark:text-gray-300"
            strokeWidth={1.5}
          />
          <span className="flex-1 text-left text-sm font-semibold text-gray-900 dark:text-white">
            {t("title")}
          </span>
          {unreadCount > 0 && (
            <span
              className={cn(
                "flex h-5 min-w-5 items-center justify-center",
                "rounded-full bg-rose-500 px-1.5 text-[10px] font-bold text-white",
              )}
            >
              {unreadCount > 99 ? "99+" : unreadCount}
            </span>
          )}
          <ChevronUp
            className="h-4 w-4 text-gray-400 dark:text-gray-500"
            strokeWidth={1.5}
          />
        </Button>
      )}

      {/* Desktop panel */}
      {isOpen && (
        <ChatWidgetPanel
          view={view}
          activeConversationId={activeConversationId}
          pendingRecipient={pendingRecipient}
          onSelectConversation={selectConversation}
          onBack={goBack}
          onClose={close}
          onPendingConversationResolved={resolvePendingConversation}
        />
      )}
    </>
  )
}
