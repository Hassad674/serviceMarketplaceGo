import { Search } from "lucide-react"
import { cn } from "@/shared/lib/utils"

type DataTableToolbarProps = {
  searchValue: string
  onSearchChange: (value: string) => void
  searchPlaceholder?: string
  children?: React.ReactNode
  className?: string
}

export function DataTableToolbar({
  searchValue,
  onSearchChange,
  searchPlaceholder = "Rechercher...",
  children,
  className,
}: DataTableToolbarProps) {
  return (
    <div className={cn("flex flex-wrap items-center gap-3", className)}>
      <div className="relative flex-1 min-w-[200px] max-w-sm">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <input
          type="text"
          value={searchValue}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder={searchPlaceholder}
          className="w-full rounded-lg border border-border bg-background py-2 pl-9 pr-3 text-sm transition-all duration-200 ease-out placeholder:text-muted-foreground focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-500/20"
        />
      </div>
      {children}
    </div>
  )
}
