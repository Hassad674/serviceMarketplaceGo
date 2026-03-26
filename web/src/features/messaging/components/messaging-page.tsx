"use client"

import { useState, useCallback, useEffect } from "react"
import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import { useSearchParams } from "next/navigation"
import { cn } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import { ConversationList } from "./conversation-list"
import { ConversationHeader } from "./conversation-header"
import { MessageArea } from "./message-area"
import { MessageInput } from "./message-input"
import { useConversations } from "../hooks/use-conversations"
import { useMessages, useSendMessage, useEditMessage, useDeleteMessage } from "../hooks/use-messages"
import { useMessagingWS } from "../hooks/use-messaging-ws"
import { markAsRead } from "../api/messaging-api"
import type { Conversation } from "../types"

export function MessagingPage() {
  const t = useTranslations("messaging")
  const searchParams = useSearchParams()
  const { data: user } = useUser()

  const [activeId, setActiveId] = useState<string | null>(
    searchParams.get("id"),
  )
  const [roleFilter, setRoleFilter] = useState<"all" | string>("all")
  const [searchQuery, setSearchQuery] = useState("")
  const [mobileView, setMobileView] = useState<"list" | "chat">("list")

  const { data: conversationsData, isLoading: conversationsLoading } = useConversations()
  const messagesQuery = useMessages(activeId)
  const sendMessage = useSendMessage(activeId)
  const editMessageMut = useEditMessage(activeId)
  const deleteMessageMut = useDeleteMessage(activeId)
  const { typingUsers, sendTyping, isConnected } = useMessagingWS(user?.id)

  const conversations = conversationsData?.data ?? []
  const activeConversation = conversations.find(
    (c: Conversation) => c.id === activeId,
  )

  // Deep-link from query param
  useEffect(() => {
    const paramId = searchParams.get("id")
    if (paramId && paramId !== activeId) {
      setActiveId(paramId)
      setMobileView("chat")
    }
  }, [searchParams, activeId])

  // Mark as read when opening a conversation
  useEffect(() => {
    if (activeId && activeConversation && activeConversation.unread_count > 0) {
      markAsRead(activeId).catch(() => {
        // Silent fail — unread count will refresh via WS
      })
    }
  }, [activeId, activeConversation])

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
      sendMessage.mutate({ content, type: "text" })
    },
    [activeId, sendMessage],
  )

  const handleSendFile = useCallback(
    (content: string, metadata: { file_key: string; filename: string; size: number; mime_type: string }) => {
      if (!activeId) return
      sendMessage.mutate({ content, type: "file", metadata })
    },
    [activeId, sendMessage],
  )

  const handleEdit = useCallback(
    (messageId: string, content: string) => {
      editMessageMut.mutate({ messageId, content })
    },
    [editMessageMut],
  )

  const handleDelete = useCallback(
    (messageId: string) => {
      deleteMessageMut.mutate(messageId)
    },
    [deleteMessageMut],
  )

  const handleTyping = useCallback(() => {
    if (activeId) sendTyping(activeId)
  }, [activeId, sendTyping])

  const typingUserForConversation = activeId ? typingUsers[activeId] : undefined

  // Gather all messages from infinite query pages
  const allMessages = messagesQuery.data?.pages.flatMap((page) => page.data) ?? []

  return (
    <div className="-mx-5 -mt-5 flex h-[calc(100vh-3.5rem)] overflow-hidden bg-white dark:bg-gray-900">
      {/* Left panel: conversation list */}
      <div
        className={cn(
          "w-full shrink-0 border-r border-gray-100 bg-white dark:border-gray-800 dark:bg-gray-900",
          "lg:w-[400px] lg:block",
          mobileView === "list" ? "block" : "hidden lg:block",
        )}
      >
        {conversationsLoading ? (
          <ConversationListSkeleton />
        ) : (
          <ConversationList
            conversations={conversations}
            activeId={activeId}
            roleFilter={roleFilter}
            searchQuery={searchQuery}
            onSelect={handleSelect}
            onRoleFilterChange={setRoleFilter}
            onSearchChange={setSearchQuery}
          />
        )}
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
              typingUserName={typingUserForConversation ? activeConversation.other_user_name : undefined}
              isConnected={isConnected}
            />
            <MessageArea
              messages={allMessages}
              currentUserId={user?.id ?? ""}
              isLoading={messagesQuery.isLoading}
              hasMore={messagesQuery.hasNextPage ?? false}
              onLoadMore={() => messagesQuery.fetchNextPage()}
              onEdit={handleEdit}
              onDelete={handleDelete}
            />
            <MessageInput
              onSend={handleSend}
              onSendFile={handleSendFile}
              onTyping={handleTyping}
              isSending={sendMessage.isPending}
            />
          </>
        ) : (
          <EmptyState label={t("noConversations")} />
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

function ConversationListSkeleton() {
  return (
    <div className="flex h-full flex-col">
      <div className="px-5 pt-5 pb-3">
        <div className="h-6 w-32 animate-pulse rounded-md bg-gray-200 dark:bg-gray-700" />
      </div>
      <div className="flex gap-1.5 px-5 pb-3">
        {[1, 2, 3, 4].map((i) => (
          <div
            key={i}
            className="h-7 w-16 animate-pulse rounded-full bg-gray-200 dark:bg-gray-700"
          />
        ))}
      </div>
      <div className="px-5 pb-3">
        <div className="h-9 w-full animate-pulse rounded-lg bg-gray-200 dark:bg-gray-700" />
      </div>
      <div className="flex-1 space-y-1 overflow-hidden">
        {[1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="flex items-center gap-3 px-5 py-3">
            <div className="h-10 w-10 shrink-0 animate-pulse rounded-full bg-gray-200 dark:bg-gray-700" />
            <div className="min-w-0 flex-1 space-y-2">
              <div className="h-4 w-28 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
              <div className="h-3 w-40 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
