"use client"

import { useTranslations } from "next-intl"

interface TypingIndicatorProps {
  userName: string
}

export function TypingIndicator({ userName }: TypingIndicatorProps) {
  const t = useTranslations("messaging")

  return (
    <div className="flex items-center gap-1.5 text-xs text-gray-400 dark:text-gray-500">
      <span>{t("typing", { name: userName })}</span>
      <span className="flex gap-0.5">
        <span className="h-1 w-1 animate-bounce rounded-full bg-gray-400 dark:bg-gray-500 [animation-delay:0ms]" />
        <span className="h-1 w-1 animate-bounce rounded-full bg-gray-400 dark:bg-gray-500 [animation-delay:150ms]" />
        <span className="h-1 w-1 animate-bounce rounded-full bg-gray-400 dark:bg-gray-500 [animation-delay:300ms]" />
      </span>
    </div>
  )
}
