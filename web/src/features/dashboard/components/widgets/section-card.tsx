"use client"

import { ChevronRight } from "lucide-react"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"

// SectionCard is the half-width or full-width content card used for
// pipeline / opportunities / recruitments lists below the stat strip.
// Renders a header (title + optional CTA) and accepts arbitrary
// children for the body — the per-role layouts compose the body from
// their own slots.

interface SectionCardProps {
  title: string
  /** Pre-translated CTA label. Hidden when undefined. */
  ctaLabel?: string
  ctaHref?: string
  /** Pre-translated empty-state copy, rendered when `isEmpty` is true. */
  emptyMessage?: string
  isEmpty?: boolean
  isLoading?: boolean
  children?: React.ReactNode
  /** Optional className appended to the wrapper for layout overrides. */
  className?: string
}

export function SectionCard(props: SectionCardProps) {
  return (
    <section
      className={cn(
        "rounded-2xl border border-border bg-card p-5 shadow-card",
        props.className,
      )}
    >
      <header className="flex items-center justify-between gap-3">
        <h2 className="font-serif text-[18px] font-medium tracking-[-0.01em] text-foreground">
          {props.title}
        </h2>
        {props.ctaLabel && props.ctaHref ? (
          <Link
            href={props.ctaHref}
            className={cn(
              "group inline-flex items-center gap-1 text-[13px] font-medium text-primary-deep",
              "hover:text-primary",
              "focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/15 focus-visible:rounded",
            )}
          >
            {props.ctaLabel}
            <ChevronRight
              className="h-3.5 w-3.5 transition-transform group-hover:translate-x-0.5"
              aria-hidden
            />
          </Link>
        ) : null}
      </header>
      <SectionBody
        isLoading={Boolean(props.isLoading)}
        isEmpty={Boolean(props.isEmpty)}
        emptyMessage={props.emptyMessage}
      >
        {props.children}
      </SectionBody>
    </section>
  )
}

interface SectionBodyProps {
  isLoading: boolean
  isEmpty: boolean
  emptyMessage?: string
  children?: React.ReactNode
}

function SectionBody({ isLoading, isEmpty, emptyMessage, children }: SectionBodyProps) {
  if (isLoading) {
    return (
      <div className="mt-4 space-y-2" aria-busy="true">
        <div className="h-12 animate-pulse rounded-xl bg-muted/60" />
        <div className="h-12 animate-pulse rounded-xl bg-muted/60" />
        <div className="h-12 animate-pulse rounded-xl bg-muted/40" />
      </div>
    )
  }
  if (isEmpty && emptyMessage) {
    return (
      <p className="mt-4 rounded-xl border border-dashed border-border bg-card p-4 text-sm text-muted-foreground">
        {emptyMessage}
      </p>
    )
  }
  return <div className="mt-4">{children}</div>
}
