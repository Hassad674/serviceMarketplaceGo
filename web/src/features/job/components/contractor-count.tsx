"use client"

import { Minus, Plus } from "lucide-react"
import { cn } from "@/shared/lib/utils"

type ContractorCountProps = {
  label: string
  value: number
  onChange: (value: number) => void
}

const MIN_COUNT = 1
const MAX_COUNT = 20

export function ContractorCount({ label, value, onChange }: ContractorCountProps) {
  function decrement() {
    if (value > MIN_COUNT) onChange(value - 1)
  }

  function increment() {
    if (value < MAX_COUNT) onChange(value + 1)
  }

  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {label}
      </label>
      <div className="inline-flex items-center gap-3 rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 px-3 py-2">
        <button
          type="button"
          onClick={decrement}
          disabled={value <= MIN_COUNT}
          className={cn(
            "flex h-8 w-8 items-center justify-center rounded-lg transition-all duration-200",
            value <= MIN_COUNT
              ? "cursor-not-allowed text-gray-300 dark:text-gray-600"
              : "text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 active:scale-95",
          )}
          aria-label="Decrease count"
        >
          <Minus className="h-4 w-4" strokeWidth={2} />
        </button>
        <span className="min-w-[2rem] text-center text-sm font-semibold tabular-nums text-gray-900 dark:text-white">
          {value}
        </span>
        <button
          type="button"
          onClick={increment}
          disabled={value >= MAX_COUNT}
          className={cn(
            "flex h-8 w-8 items-center justify-center rounded-lg transition-all duration-200",
            value >= MAX_COUNT
              ? "cursor-not-allowed text-gray-300 dark:text-gray-600"
              : "text-gray-600 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 active:scale-95",
          )}
          aria-label="Increase count"
        >
          <Plus className="h-4 w-4" strokeWidth={2} />
        </button>
      </div>
    </div>
  )
}
