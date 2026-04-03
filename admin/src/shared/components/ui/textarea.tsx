import { forwardRef, type TextareaHTMLAttributes } from "react"
import { cn } from "@/shared/lib/utils"

type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement> & {
  label?: string
  error?: string
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, label, error, id, ...props }, ref) => {
    const textareaId = id || label?.toLowerCase().replace(/\s+/g, "-")
    return (
      <div>
        {label && (
          <label htmlFor={textareaId} className="mb-1 block text-sm font-medium text-foreground">
            {label}
          </label>
        )}
        <textarea
          ref={ref}
          id={textareaId}
          className={cn(
            "w-full rounded-lg border bg-background px-3 py-2 text-sm transition-all duration-200 ease-out",
            "placeholder:text-muted-foreground",
            "focus:outline-none focus:ring-2 focus:ring-rose-500/20",
            error ? "border-destructive focus:ring-destructive/20" : "border-border focus:border-rose-500",
            className,
          )}
          {...props}
        />
        {error && <p className="mt-1 text-xs text-destructive">{error}</p>}
      </div>
    )
  },
)

Textarea.displayName = "Textarea"
