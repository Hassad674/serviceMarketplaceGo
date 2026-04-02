import { cn } from "@/shared/lib/utils"

type DivProps = React.HTMLAttributes<HTMLDivElement>

export function Card({ className, ...props }: DivProps) {
  return (
    <div
      className={cn("rounded-xl border border-border bg-card shadow-sm", className)}
      {...props}
    />
  )
}

export function CardHeader({ className, ...props }: DivProps) {
  return <div className={cn("px-6 pt-6", className)} {...props} />
}

export function CardTitle({ className, ...props }: DivProps) {
  return (
    <h3 className={cn("text-lg font-semibold text-card-foreground", className)} {...props} />
  )
}

export function CardContent({ className, ...props }: DivProps) {
  return <div className={cn("px-6 py-4", className)} {...props} />
}

export function CardFooter({ className, ...props }: DivProps) {
  return (
    <div
      className={cn("flex items-center border-t border-border px-6 py-4", className)}
      {...props}
    />
  )
}
