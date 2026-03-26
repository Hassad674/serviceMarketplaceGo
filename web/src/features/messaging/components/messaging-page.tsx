"use client"

import { useState, useCallback } from "react"
import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { ConversationList } from "./conversation-list"
import { ConversationHeader } from "./conversation-header"
import { MessageArea } from "./message-area"
import { MessageInput } from "./message-input"
import { MOCK_CONVERSATIONS, MOCK_MESSAGES } from "../mock-data"
import type { ConversationRole, Message } from "../types"

export function MessagingPage() {
  const t = useTranslations("messaging")
  const [activeId, setActiveId] = useState<string | null>(null)
  const [roleFilter, setRoleFilter] = useState<"all" | ConversationRole>("all")
  const [searchQuery, setSearchQuery] = useState("")
  const [messages, setMessages] = useState<Message[]>(MOCK_MESSAGES)
  const [mobileView, setMobileView] = useState<"list" | "chat">("list")

  const activeConversation = MOCK_CONVERSATIONS.find((c) => c.id === activeId)
  const conversationMessages = messages.filter(
    (m) => m.conversationId === activeId,
  )

  const handleSelect = useCallback((id: string) => {
    setActiveId(id)
    setMobileView("chat")
  }, [])

  const handleBack = useCallback(() => {
    setMobileView("list")
  }, [])

  const handleSend = useCallback(
    (content: string) => {
      if (!activeId) return
      const newMessage: Message = {
        id: `m${Date.now()}`,
        conversationId: activeId,
        senderId: "me",
        content,
        sentAt: new Date().toLocaleTimeString([], {
          hour: "2-digit",
          minute: "2-digit",
        }),
        isOwn: true,
      }
      setMessages((prev) => [...prev, newMessage])
    },
    [activeId],
  )

  return (
    <div className="-mx-5 -mt-5 flex h-[calc(100vh-3.5rem)] overflow-hidden bg-white dark:bg-gray-900">
      {/* Left panel: conversation list */}
      <div
        className={cn(
          "w-full shrink-0 border-r border-gray-100 bg-white dark:border-gray-800 dark:bg-gray-900",
          "lg:w-[350px] lg:block",
          mobileView === "list" ? "block" : "hidden lg:block",
        )}
      >
        <ConversationList
          conversations={MOCK_CONVERSATIONS}
          activeId={activeId}
          roleFilter={roleFilter}
          searchQuery={searchQuery}
          onSelect={handleSelect}
          onRoleFilterChange={setRoleFilter}
          onSearchChange={setSearchQuery}
        />
      </div>

      {/* Right panel: active conversation */}
      <div
        className={cn(
          "flex min-w-0 flex-1 flex-col",
          mobileView === "chat" ? "flex" : "hidden lg:flex",
        )}
      >
        {activeConversation ? (
          <>
            <ConversationHeader
              conversation={activeConversation}
              onBack={handleBack}
            />
            <MessageArea messages={conversationMessages} />
            <MessageInput onSend={handleSend} />
          </>
        ) : (
          <EmptyState label={t("noMessages")} />
        )}
      </div>
    </div>
  )
}

function EmptyState({ label }: { label: string }) {
  return (
    <div className="flex flex-1 items-center justify-center bg-gray-50/50 dark:bg-gray-950/50">
      <div className="text-center">
        <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-gray-800">
          <MessageSquare
            className="h-8 w-8 text-gray-300 dark:text-gray-600"
            strokeWidth={1}
          />
        </div>
        <p className="mt-4 text-sm text-gray-400 dark:text-gray-500">
          {label}
        </p>
      </div>
    </div>
  )
}
