import { cn } from "@/shared/lib/utils"

const variants = {
  default: "bg-primary/10 text-primary",
  success: "bg-success/10 text-success",
  warning: "bg-warning/10 text-warning",
  destructive: "bg-destructive/10 text-destructive",
  outline: "border border-border text-muted-foreground",
} as const

const roleVariants: Record<string, string> = {
  agency: "bg-blue-50 text-blue-700",
  enterprise: "bg-violet-50 text-violet-700",
  provider: "bg-rose-50 text-rose-700",
  admin: "bg-slate-100 text-slate-700",
}

const roleLabels: Record<string, string> = {
  agency: "Prestataire",
  enterprise: "Entreprise",
  provider: "Freelance",
}

type BadgeProps = {
  variant?: keyof typeof variants
  className?: string
  children: React.ReactNode
}

export function Badge({ variant = "default", className, children }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
        variants[variant],
        className,
      )}
    >
      {children}
    </span>
  )
}

export function RoleBadge({ role }: { role: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
        roleVariants[role] || roleVariants.admin,
      )}
    >
      {roleLabels[role] || role}
    </span>
  )
}

export function StatusBadge({ status }: { status: string }) {
  const map: Record<string, keyof typeof variants> = {
    active: "success",
    suspended: "warning",
    banned: "destructive",
  }
  const labels: Record<string, string> = {
    active: "Actif",
    suspended: "Suspendu",
    banned: "Banni",
  }
  return (
    <Badge variant={map[status] || "outline"}>
      {labels[status] || status}
    </Badge>
  )
}
