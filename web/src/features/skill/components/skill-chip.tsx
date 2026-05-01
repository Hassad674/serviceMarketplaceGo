"use client"

import { X } from "lucide-react"
import { cn } from "@/shared/lib/utils"

import { Button } from "@/shared/components/ui/button"
interface SkillChipProps {
  displayText: string
  onRemove?: () => void
  removeLabel?: string
}

// Styled removable chip used in the modal's selected list. The
// "display" version without `onRemove` is reused on the inline
// summary panel of the profile page.
export function SkillChip({
  displayText,
  onRemove,
  removeLabel,
}: SkillChipProps) {
  const interactive = Boolean(onRemove)
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border border-primary/20",
        "bg-primary/10 px-3 py-1 text-sm font-medium text-primary",
        interactive && "pr-1",
      )}
    >
      <span className="truncate max-w-[14rem]">{displayText}</span>
      {onRemove ? (
        <Button variant="ghost" size="auto"
          type="button"
          onClick={onRemove}
          aria-label={removeLabel ?? `Remove ${displayText}`}
          className={cn(
            "inline-flex h-5 w-5 items-center justify-center rounded-full",
            "text-primary/80 hover:bg-primary/15 hover:text-primary",
            "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-1",
          )}
        >
          <X className="h-3 w-3" aria-hidden="true" />
        </Button>
      ) : null}
    </span>
  )
}
