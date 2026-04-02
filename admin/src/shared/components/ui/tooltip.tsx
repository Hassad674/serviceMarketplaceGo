import { useState, type ReactNode } from "react"
import { cn } from "@/shared/lib/utils"

type TooltipProps = {
  content: string
  children: ReactNode
  className?: string
}

export function Tooltip({ content, children, className }: TooltipProps) {
  const [visible, setVisible] = useState(false)

  return (
    <div
      className="relative inline-block"
      onMouseEnter={() => setVisible(true)}
      onMouseLeave={() => setVisible(false)}
    >
      {children}
      {visible && (
        <div
          role="tooltip"
          className={cn(
            "absolute bottom-full left-1/2 z-50 mb-2 -translate-x-1/2 rounded-md bg-foreground px-2.5 py-1 text-xs text-background whitespace-nowrap",
            className,
          )}
        >
          {content}
        </div>
      )}
    </div>
  )
}
