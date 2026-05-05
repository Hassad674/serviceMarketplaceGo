"use client"

import { useTranslations } from "next-intl"

interface TypingIndicatorProps {
  userName: string
}

export function TypingIndicator({ userName }: TypingIndicatorProps) {
  const t = useTranslations("messaging")

  return (
    <div className="flex items-center gap-1.5 text-[12px] text-muted-foreground">
      <span>{t("typing", { name: userName })}</span>
      <span className="flex gap-0.5">
        <span className="h-1 w-1 animate-bounce rounded-full bg-muted-foreground [animation-delay:0ms]" />
        <span className="h-1 w-1 animate-bounce rounded-full bg-muted-foreground [animation-delay:150ms]" />
        <span className="h-1 w-1 animate-bounce rounded-full bg-muted-foreground [animation-delay:300ms]" />
      </span>
    </div>
  )
}
