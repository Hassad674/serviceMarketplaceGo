"use client"

import { forwardRef, useId, type InputHTMLAttributes } from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/shared/lib/utils"

/**
 * Input — the project's single source of truth for text-like
 * controls. Mirrors the admin primitive but adopts the polished
 * design tokens documented in CLAUDE.md: h-10, rounded-lg, shadow-xs
 * at rest, focus:border-rose-500 focus:ring-4 ring-rose-500/10.
 *
 * Errors use `border-red-500` + `ring-4 ring-red-500/10`. Callers
 * should pass `aria-invalid` + `aria-describedby` (this component
 * wires `aria-describedby` automatically when an `error` prop is
 * provided so screen readers announce the error message).
 *
 * Label association is handled by the optional `label` prop. When
 * provided, an auto-generated id (via React.useId) ties the label
 * and input — but the caller can still pass an explicit `id` to
 * win. When no label is provided, callers MUST supply `aria-label`
 * to remain WCAG-compliant; ESLint enforces this in the codebase
 * indirectly via the forbid-elements rule on raw `<input>`.
 */
const inputVariants = cva(
	cn(
		"block w-full rounded-lg border bg-white px-3 text-sm shadow-xs",
		"transition-all duration-200 ease-out",
		"placeholder:text-slate-400",
		"focus:outline-none focus:ring-4",
		"disabled:cursor-not-allowed disabled:opacity-60 disabled:bg-slate-50",
		"dark:bg-slate-900 dark:text-slate-100 dark:placeholder:text-slate-500",
		"dark:disabled:bg-slate-800",
	),
	{
		variants: {
			state: {
				default:
					"border-slate-200 focus:border-rose-500 focus:ring-rose-500/10 dark:border-slate-700",
				error:
					"border-red-500 focus:border-red-500 focus:ring-red-500/10",
			},
			size: {
				sm: "h-8 text-xs",
				md: "h-10 text-sm",
				lg: "h-12 text-base",
			},
		},
		defaultVariants: {
			state: "default",
			size: "md",
		},
	},
)

export type InputProps = Omit<InputHTMLAttributes<HTMLInputElement>, "size"> &
	VariantProps<typeof inputVariants> & {
		/** Optional label rendered above the input and tied via htmlFor. */
		label?: string
		/** Optional hint shown beneath the input. */
		hint?: string
		/** Error message — when set, switches the input to the `error` state. */
		error?: string
		/** Extra class for the root wrapper (label + input + hint). */
		wrapperClassName?: string
	}

export const Input = forwardRef<HTMLInputElement, InputProps>(
	(
		{
			id: idProp,
			className,
			wrapperClassName,
			label,
			hint,
			error,
			state,
			size,
			"aria-invalid": ariaInvalid,
			"aria-describedby": ariaDescribedByProp,
			...props
		},
		ref,
	) => {
		const generatedId = useId()
		const id = idProp ?? generatedId
		const describedByIds = [
			error ? `${id}-error` : null,
			hint && !error ? `${id}-hint` : null,
			ariaDescribedByProp ?? null,
		].filter(Boolean) as string[]
		const finalState = state ?? (error ? "error" : "default")

		const inputElement = (
			<input
				ref={ref}
				id={id}
				aria-invalid={ariaInvalid ?? (error ? true : undefined)}
				aria-describedby={
					describedByIds.length > 0 ? describedByIds.join(" ") : undefined
				}
				className={cn(inputVariants({ state: finalState, size }), className)}
				{...props}
			/>
		)

		// When the caller supplies none of the wrapper-only props, render
		// the input inline. This keeps the migration from raw <input>
		// sites visually identical: a wrapping <div> would otherwise
		// break flex/grid layouts where the original <input> was a
		// direct child.
		const needsWrapper = Boolean(label || error || hint || wrapperClassName)
		if (!needsWrapper) {
			return inputElement
		}

		return (
			<div className={cn("flex flex-col gap-1", wrapperClassName)}>
				{label && (
					<label
						htmlFor={id}
						className="text-sm font-medium text-slate-900 dark:text-slate-200"
					>
						{label}
					</label>
				)}
				{inputElement}
				{error && (
					<p
						id={`${id}-error`}
						role="alert"
						className="text-xs text-red-600 dark:text-red-400"
					>
						{error}
					</p>
				)}
				{!error && hint && (
					<p id={`${id}-hint`} className="text-xs text-slate-500 dark:text-slate-400">
						{hint}
					</p>
				)}
			</div>
		)
	},
)

Input.displayName = "Input"

export { inputVariants }
