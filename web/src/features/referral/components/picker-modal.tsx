"use client"

import { useEffect } from "react"
import { X } from "lucide-react"

import { cn } from "@/shared/lib/utils"

interface PickerModalProps {
  open: boolean
  onClose: () => void
  title: string
  description?: string
  children: React.ReactNode
}

// PickerModal is a lightweight dialog primitive shared by the provider and
// client pickers. It lives inside the referral feature rather than
// @/shared/components/ui because no other feature needs it yet — moving it
// to shared/ is a later refactor once a third consumer shows up.
//
// Implementation notes:
//
//   - Portal-free: the overlay is a fixed-positioned div under the normal
//     React tree. That's fine for this use case — we never have more than
//     one picker modal open at a time and we don't need to escape a
//     clipping ancestor.
//   - Escape key closes. Click on the backdrop closes. Click inside the
//     panel does NOT close (event stopPropagation on the panel).
//   - role="dialog" + aria-modal for screen readers.
export function PickerModal({
  open,
  onClose,
  title,
  description,
  children,
}: PickerModalProps) {
  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose()
    }
    document.addEventListener("keydown", onKey)
    return () => document.removeEventListener("keydown", onKey)
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-slate-900/50 px-4 pt-16 pb-8 animate-in fade-in"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label={title}
    >
      <div
        className={cn(
          "flex max-h-[calc(100vh-8rem)] w-full max-w-xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl",
        )}
        onClick={(e) => e.stopPropagation()}
      >
        <header className="flex items-start justify-between gap-4 border-b border-slate-100 px-6 py-4">
          <div>
            <h2 className="text-base font-semibold text-slate-900">{title}</h2>
            {description && (
              <p className="mt-0.5 text-xs text-slate-500">{description}</p>
            )}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
            aria-label="Fermer"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </header>
        <div className="flex flex-1 flex-col overflow-y-auto">{children}</div>
      </div>
    </div>
  )
}

// PickerTrigger renders the non-button clickable input the pickers sit on.
// It has to be a <div role="button"> (not a real <button>) so the optional
// inline X-clear control can be a real <button> without hitting the "button
// cannot be a descendant of button" HTML / React hydration error.
interface PickerTriggerProps {
  onOpen: () => void
  onClear: (() => void) | null
  open: boolean
  children: React.ReactNode
  label: string
}

export function PickerTrigger({
  onOpen,
  onClear,
  open,
  children,
  label,
}: PickerTriggerProps) {
  return (
    <div>
      <span className="mb-1.5 block text-sm font-medium text-slate-700">
        {label}
      </span>
      <div
        role="button"
        tabIndex={0}
        onClick={onOpen}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault()
            onOpen()
          }
        }}
        className={cn(
          "flex w-full cursor-pointer items-center justify-between gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2.5 text-left text-sm transition",
          "focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-100",
          "hover:border-slate-400",
          open && "border-rose-500 ring-2 ring-rose-100",
        )}
      >
        <div className="flex min-w-0 flex-1 items-center gap-2">{children}</div>
        {onClear && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation()
              onClear()
            }}
            className="shrink-0 rounded p-0.5 text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
            aria-label="Effacer la sélection"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        )}
      </div>
    </div>
  )
}
