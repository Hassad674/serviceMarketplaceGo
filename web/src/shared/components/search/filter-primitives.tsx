"use client"

import { cn } from "@/shared/lib/utils"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"
// Shared leaf primitives used by every filter section. Lives next to
// search-filter-sidebar so the section files don't import from each
// other and the duplication-rule of three is honored once.

export function SectionShell({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <section className="flex flex-col gap-2">
      <h3 className="text-[13px] font-semibold uppercase tracking-wide text-muted-foreground">
        {title}
      </h3>
      {children}
    </section>
  )
}

export function PillButton({
  label,
  selected,
  onClick,
}: {
  label: string
  selected: boolean
  onClick: () => void
}) {
  return (
    <Button variant="ghost" size="auto"
      type="button"
      onClick={onClick}
      aria-pressed={selected}
      className={cn(
        "rounded-full border px-3 py-1 text-xs font-medium transition-colors",
        selected
          ? "border-rose-500 bg-rose-50 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300"
          : "border-border bg-background text-muted-foreground hover:text-foreground",
        "focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-rose-500/20",
      )}
    >
      {label}
    </Button>
  )
}

export function NumberInput({
  value,
  onChange,
  placeholder,
  ariaLabel,
}: {
  value: number | null
  onChange: (next: number | null) => void
  placeholder: string
  ariaLabel: string
}) {
  return (
    <Input
      type="number"
      min={0}
      inputMode="numeric"
      value={value ?? ""}
      placeholder={placeholder}
      aria-label={ariaLabel}
      onChange={(e) => {
        const raw = e.target.value.trim()
        onChange(raw === "" ? null : Math.max(0, Number(raw) || 0))
      }}
      className="h-10 w-full min-w-0 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
    />
  )
}

export function CheckboxRow({
  checked,
  onChange,
  label,
}: {
  checked: boolean
  onChange: () => void
  label: string
}) {
  return (
    <label className="flex cursor-pointer items-center gap-2 rounded-md px-1 py-1 text-sm text-foreground hover:bg-muted/50">
      <Input
        type="checkbox"
        checked={checked}
        onChange={onChange}
        className="h-4 w-4 rounded border-border text-rose-500 focus:ring-rose-500/20"
      />
      <span className="flex-1 truncate">{label}</span>
    </label>
  )
}

export function toggle<T>(list: T[], value: T): T[] {
  return list.includes(value) ? list.filter((item) => item !== value) : [...list, value]
}
