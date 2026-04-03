import { forwardRef, type InputHTMLAttributes } from "react"
import { cn } from "@/shared/lib/utils"

type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  label?: string
  error?: string
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, error, id, ...props }, ref) => {
    const inputId = id || label?.toLowerCase().replace(/\s+/g, "-")
    return (
      <div>
        {label && (
          <label htmlFor={inputId} className="mb-1 block text-sm font-medium text-foreground">
            {label}
          </label>
        )}
        <input
          ref={ref}
          id={inputId}
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

Input.displayName = "Input"
