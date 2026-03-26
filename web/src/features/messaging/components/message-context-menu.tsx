"use client"

import { useState, useRef, useEffect } from "react"
import { MoreHorizontal, Pencil, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

interface MessageContextMenuProps {
  onEdit: () => void
  onDelete: () => void
}

export function MessageContextMenu({ onEdit, onDelete }: MessageContextMenuProps) {
  const t = useTranslations("messaging")
  const [open, setOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setOpen(false)
      }
    }
    if (open) {
      document.addEventListener("mousedown", handleClickOutside)
    }
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [open])

  return (
    <div ref={menuRef} className="relative">
      <button
        onClick={() => setOpen((prev) => !prev)}
        className={cn(
          "rounded-md p-1 text-gray-400 transition-colors",
          "hover:bg-gray-100 hover:text-gray-600",
          "dark:hover:bg-gray-700 dark:hover:text-gray-300",
          "opacity-0 group-hover:opacity-100 focus:opacity-100",
        )}
        aria-label="Message options"
      >
        <MoreHorizontal className="h-4 w-4" strokeWidth={1.5} />
      </button>

      {open && (
        <div
          className={cn(
            "absolute right-0 top-full z-10 mt-1 w-36 overflow-hidden rounded-lg",
            "border border-gray-100 bg-white shadow-lg",
            "dark:border-gray-700 dark:bg-gray-800",
            "animate-in fade-in slide-in-from-top-1 duration-150",
          )}
        >
          <button
            onClick={() => {
              setOpen(false)
              onEdit()
            }}
            className={cn(
              "flex w-full items-center gap-2 px-3 py-2 text-sm text-gray-700",
              "transition-colors hover:bg-gray-50",
              "dark:text-gray-300 dark:hover:bg-gray-700",
            )}
          >
            <Pencil className="h-3.5 w-3.5" strokeWidth={1.5} />
            {t("editMessage")}
          </button>
          <button
            onClick={() => {
              setOpen(false)
              onDelete()
            }}
            className={cn(
              "flex w-full items-center gap-2 px-3 py-2 text-sm text-red-600",
              "transition-colors hover:bg-red-50",
              "dark:text-red-400 dark:hover:bg-red-500/10",
            )}
          >
            <Trash2 className="h-3.5 w-3.5" strokeWidth={1.5} />
            {t("deleteMessage")}
          </button>
        </div>
      )}
    </div>
  )
}
