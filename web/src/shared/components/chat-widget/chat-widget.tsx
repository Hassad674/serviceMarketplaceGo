"use client"

import dynamic from "next/dynamic"
import { MessageCircle, ChevronUp } from "lucide-react"
import { useTranslations } from "next-intl"
import { usePathname } from "next/navigation"
import { cn } from "@/shared/lib/utils"
import { useUnreadCount } from "@/shared/hooks/use-unread-count"
import { useMediaQuery } from "@/shared/hooks/use-media-query"
import { useChatWidget } from "./use-chat-widget"

import { Button } from "@/shared/components/ui/button"

// Lazy-loaded panel — keeps initial bundle small when no chat is opened.
const ChatWidgetPanel = dynamic(
  () =>
    import("./chat-widget-panel").then((m) => ({
      default: m.ChatWidgetPanel,
    })),
  { ssr: false },
)

// ChatWidget — Soleil v2 floating shortcut to messaging.
//
// Three states (driven by useChatWidget — OFF-LIMITS hook, do not touch):
//  - collapsed: a corail pill anchored bottom-right
//  - list: 380×540 panel showing all conversations
//  - chat: same panel size, single conversation thread
//
// Hidden on /messages route to avoid double UI. Hidden on mobile —
// the navbar already exposes a Messages tab on small screens.
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

  // Hide on /messages pages (any locale prefix) — the full page is
  // already the active surface, the widget would just duplicate it.
  if (pathname.includes("/messages")) return null

  // Hide on mobile — bottom navbar carries the Messages entry.
  if (!isDesktop) return null

  return (
    <>
      {/* Collapsed: corail pill, bottom-right, 24px margin */}
      {!isOpen && (
        <Button
          variant="ghost"
          size="auto"
          onClick={open}
          className={cn(
            "fixed bottom-6 right-6 z-50 inline-flex items-center gap-2.5",
            "h-12 rounded-full pl-2 pr-4",
            "bg-card text-foreground",
            "border border-border",
            "shadow-[0_8px_24px_rgba(42,31,21,0.12)]",
            "transition-all duration-200 ease-out",
            "hover:shadow-[0_12px_28px_rgba(42,31,21,0.16)]",
            "hover:-translate-y-0.5",
          )}
          aria-label={t("messagingWidget_open")}
        >
          {/* Icon disc with optional counter badge */}
          <span className="relative flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-soft text-primary-deep">
            <MessageCircle className="h-[18px] w-[18px]" strokeWidth={1.6} />
            {unreadCount > 0 && (
              <span
                className={cn(
                  "absolute -right-1 -top-1 flex h-[18px] min-w-[18px] items-center justify-center",
                  "rounded-full bg-primary px-1 text-[10px] font-semibold leading-none text-primary-foreground",
                  "border-2 border-card",
                )}
              >
                {unreadCount > 99 ? "99+" : unreadCount}
              </span>
            )}
          </span>
          <span className="text-sm font-semibold">
            {t("messagingWidget_title")}
          </span>
          <ChevronUp
            className="h-4 w-4 text-muted-foreground"
            strokeWidth={1.5}
          />
        </Button>
      )}

      {/* Open panel (list or chat view) */}
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
