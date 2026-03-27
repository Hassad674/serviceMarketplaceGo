"use client"

import dynamic from "next/dynamic"
import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import { usePathname } from "next/navigation"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useUnreadCount } from "@/shared/hooks/use-unread-count"
import { useMediaQuery } from "@/shared/hooks/use-media-query"
import { useChatWidget } from "./use-chat-widget"

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
  const router = useRouter()
  const isDesktop = useMediaQuery("(min-width: 1024px)")
  const { data: unreadData } = useUnreadCount()
  const unreadCount = unreadData?.count ?? 0

  const {
    isOpen,
    view,
    activeConversationId,
    open,
    close,
    selectConversation,
    goBack,
  } = useChatWidget()

  // Hide on /messages pages (any locale prefix)
  if (pathname.includes("/messages")) return null

  function handleToggle() {
    if (!isDesktop) {
      router.push("/messages")
      return
    }
    if (isOpen) {
      close()
    } else {
      open()
    }
  }

  return (
    <>
      {/* Floating action button */}
      <button
        onClick={handleToggle}
        className={cn(
          "fixed bottom-6 right-6 z-50 flex h-14 w-14 items-center justify-center",
          "rounded-full bg-rose-500 text-white shadow-lg",
          "transition-all duration-200 hover:bg-rose-600 hover:shadow-glow",
          "active:scale-[0.96]",
        )}
        aria-label={t("openChat")}
      >
        <MessageSquare className="h-6 w-6" strokeWidth={1.5} />
        {unreadCount > 0 && (
          <span
            className={cn(
              "absolute -right-1 -top-1 flex h-5 min-w-5 items-center justify-center",
              "rounded-full bg-rose-600 px-1.5 text-[10px] font-bold text-white",
              "ring-2 ring-white dark:ring-gray-950",
            )}
          >
            {unreadCount > 99 ? "99+" : unreadCount}
          </span>
        )}
      </button>

      {/* Desktop panel */}
      {isDesktop && isOpen && (
        <ChatWidgetPanel
          view={view}
          activeConversationId={activeConversationId}
          onSelectConversation={selectConversation}
          onBack={goBack}
          onClose={close}
        />
      )}
    </>
  )
}
