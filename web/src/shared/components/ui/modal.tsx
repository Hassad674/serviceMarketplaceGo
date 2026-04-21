"use client"

import { useCallback, useEffect, useId, useRef } from "react"
import { createPortal } from "react-dom"
import { X } from "lucide-react"
import { cn } from "@/shared/lib/utils"

/**
 * Modal is the shared dialog primitive for the app.
 *
 * It renders through a React portal on `document.body` so that
 * `position: fixed` escapes any ancestor CSS that would otherwise
 * create a containing block for fixed elements (transform, filter,
 * will-change, contain: paint — the stuff that silently pins a
 * "fixed" element inside the wrong ancestor).
 *
 * The OVERLAY is the scroll container (Headless UI / Radix pattern):
 * short modals center vertically, tall ones let the user scroll the
 * backdrop. Nothing ever clips because the dialog has no max-height
 * constraint of its own.
 *
 * Responsibilities:
 *   - Portal mount + unmount
 *   - Backdrop with click-to-dismiss
 *   - Escape closes
 *   - Focus trap + restore focus to the trigger on close
 *   - aria-modal wiring
 *   - Optional header (title + close button)
 *
 * Intentionally NOT responsible for: state management (callers own
 * `open`), animation choreography (CSS handles it), or scroll-locking
 * the page body (the backdrop already covers interaction, and
 * preventing body scroll on every modal creates edge cases with iOS
 * Safari that are out of scope for V1).
 */
type ModalProps = {
	open: boolean
	onClose: () => void
	/** Accessible dialog title. Rendered in the header when `showHeader` is true. */
	title: string
	/** Set false when the caller wants a bare dialog (no title bar / close button). */
	showHeader?: boolean
	/** Tailwind max-width class applied to the dialog. Defaults to `max-w-md`. */
	maxWidthClassName?: string
	children: React.ReactNode
}

export function Modal({
	open,
	onClose,
	title,
	showHeader = true,
	maxWidthClassName = "max-w-md",
	children,
}: ModalProps) {
	const triggerRef = useRef<HTMLElement | null>(null)
	const dialogRef = useRef<HTMLDivElement | null>(null)
	const titleId = useId()

	useEffect(() => {
		if (!open) return

		// Remember what was focused before we opened, so we can restore it.
		triggerRef.current = document.activeElement as HTMLElement | null

		// Focus the first focusable inside the dialog on open.
		const firstFocusable = dialogRef.current?.querySelector<HTMLElement>(
			'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
		)
		firstFocusable?.focus()

		return () => {
			triggerRef.current?.focus?.()
		}
	}, [open])

	useEffect(() => {
		if (!open) return

		function onKey(e: KeyboardEvent) {
			if (e.key === "Escape") {
				e.stopPropagation()
				onClose()
				return
			}
			if (e.key === "Tab") {
				trapFocus(e, dialogRef.current)
			}
		}

		document.addEventListener("keydown", onKey)
		return () => document.removeEventListener("keydown", onKey)
	}, [open, onClose])

	const onBackdropClick = useCallback(
		(e: React.MouseEvent) => {
			if (e.target === e.currentTarget) onClose()
		},
		[onClose],
	)

	// Portal target only exists on the client — this check also guards
	// against SSR / static rendering where `document` is undefined. When
	// the component is `"use client"`, React still server-renders an
	// initial pass for hydration, so we MUST handle that.
	if (!open || typeof document === "undefined") return null

	const overlay = (
		<div
			className="fixed inset-0 z-[100] overflow-y-auto bg-black/50 backdrop-blur-sm"
			onClick={onBackdropClick}
			role="presentation"
		>
			<div
				className="flex min-h-full items-center justify-center p-4"
				onClick={onBackdropClick}
			>
				<div
					ref={dialogRef}
					role="dialog"
					aria-modal="true"
					aria-labelledby={titleId}
					className={cn(
						"w-full animate-scale-in rounded-2xl bg-white p-6 shadow-xl dark:bg-slate-900",
						maxWidthClassName,
					)}
				>
					{showHeader ? (
						<div className="mb-5 flex items-start justify-between gap-3">
							<h2
								id={titleId}
								className="text-lg font-semibold text-slate-900 dark:text-white"
							>
								{title}
							</h2>
							<button
								type="button"
								onClick={onClose}
								aria-label="Fermer"
								className="rounded-lg p-1.5 text-slate-500 transition-colors hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800"
							>
								<X className="h-5 w-5" aria-hidden="true" />
							</button>
						</div>
					) : null}
					{children}
				</div>
			</div>
		</div>
	)

	return createPortal(overlay, document.body)
}

function trapFocus(e: KeyboardEvent, dialog: HTMLElement | null) {
	if (!dialog) return
	const nodes = dialog.querySelectorAll<HTMLElement>(
		'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
	)
	if (nodes.length === 0) return
	const first = nodes[0]
	const last = nodes[nodes.length - 1]
	if (e.shiftKey && document.activeElement === first) {
		e.preventDefault()
		last.focus()
	} else if (!e.shiftKey && document.activeElement === last) {
		e.preventDefault()
		first.focus()
	}
}

