"use client"

import { cn } from "@/shared/lib/utils"

interface AvatarFallbackProps {
  name: string
  size: "sm" | "md" | "lg"
}

const SIZE_CLASSES: Record<AvatarFallbackProps["size"], string> = {
  sm: "h-10 w-10 text-sm",
  md: "h-20 w-20 text-xl",
  lg: "h-[120px] w-[120px] text-3xl",
}

export function AvatarFallback({ name, size }: AvatarFallbackProps) {
  const initials = name
    .split(" ")
    .map((w) => w.charAt(0))
    .join("")
    .slice(0, 2)
    .toUpperCase()

  return (
    <div
      className={cn(
        "flex items-center justify-center rounded-full",
        "bg-gradient-to-br from-rose-500 to-purple-600",
        "font-bold text-white shadow-lg",
        SIZE_CLASSES[size],
      )}
    >
      {initials}
    </div>
  )
}
