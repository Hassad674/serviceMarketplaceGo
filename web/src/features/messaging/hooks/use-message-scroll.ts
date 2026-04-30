"use client"

import { useEffect, useRef, type RefObject } from "react"
import type { Message } from "../types"

interface UseMessageScrollOptions {
  messages: Message[]
  hasMore: boolean
  onLoadMore: () => void
}

interface UseMessageScrollResult {
  scrollRef: RefObject<HTMLDivElement | null>
  topSentinelRef: RefObject<HTMLDivElement | null>
}

/**
 * Manages two side effects for a chat-style message list:
 *
 *   1. Auto-scrolls to the bottom whenever a NEW message lands at the
 *      end of the list (initial mount = instant, subsequent appends =
 *      smooth). Older messages prepended at the top (load-more) do
 *      NOT trigger auto-scroll — the user keeps their reading
 *      position.
 *   2. Wires an IntersectionObserver on a sentinel element near the
 *      top of the scroll container. When the sentinel becomes visible
 *      the hook calls `onLoadMore` to fetch the next page.
 *
 * Returns two refs that the caller MUST attach to:
 *   - `scrollRef` → the scroll container (overflow-y: auto element)
 *   - `topSentinelRef` → a thin element rendered at the top of the
 *     visible content, BEFORE the first message
 */
export function useMessageScroll({
  messages,
  hasMore,
  onLoadMore,
}: UseMessageScrollOptions): UseMessageScrollResult {
  const scrollRef = useRef<HTMLDivElement>(null)
  const topSentinelRef = useRef<HTMLDivElement>(null)
  const prevMessageCountRef = useRef(0)

  // Effect 1 — scroll to bottom when new messages land at the end.
  useEffect(() => {
    if (messages.length > prevMessageCountRef.current && scrollRef.current) {
      const isNewMessageAtEnd =
        messages.length > 0 &&
        prevMessageCountRef.current > 0 &&
        messages[messages.length - 1]?.id !==
          messages[prevMessageCountRef.current - 1]?.id
      if (isNewMessageAtEnd || prevMessageCountRef.current === 0) {
        scrollRef.current.scrollTo({
          top: scrollRef.current.scrollHeight,
          behavior: prevMessageCountRef.current === 0 ? "instant" : "smooth",
        })
      }
    }
    prevMessageCountRef.current = messages.length
  }, [messages])

  // Effect 2 — infinite scroll up to load older messages.
  useEffect(() => {
    const sentinel = topSentinelRef.current
    const container = scrollRef.current
    if (!sentinel || !container || !hasMore) return

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          onLoadMore()
        }
      },
      { root: container, threshold: 0.1 },
    )

    observer.observe(sentinel)
    return () => observer.disconnect()
  }, [hasMore, onLoadMore])

  return { scrollRef, topSentinelRef }
}
