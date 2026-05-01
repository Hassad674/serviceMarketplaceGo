"use client"

import { forwardRef, useId, type SelectHTMLAttributes } from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/shared/lib/utils"

/**
 * Select — minimal native-select wrapper that mirrors the Input
 * primitive's visual language (rose focus ring, h-10 default height,
 * shadow-xs at rest). Wrapping `<select>` keeps native keyboard
 * behaviour for free: Up/Down to move through options, Enter to
 * commit, Escape to dismiss the dropdown — all without bringing in
 * a Radix dependency.
 *
 * Callers can either pass `<option>` children OR an `options` array
 * (the latter is convenient for data-driven selects from API DTOs).
 * `placeholder`, when provided, becomes a disabled empty `<option>`.
 *
 * Label association mirrors `Input`: an auto-generated id ties the
 * label and select unless an explicit `id` is provided.
 */
const selectVariants = cva(
	cn(
		"block w-full appearance-none rounded-lg border bg-white pr-10 pl-3 text-sm shadow-xs",
		"transition-all duration-200 ease-out",
		"focus:outline-none focus:ring-4",
		"disabled:cursor-not-allowed disabled:opacity-60 disabled:bg-slate-50",
		"dark:bg-slate-900 dark:text-slate-100",
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

type SelectOption = {
	value: string
	label: string
	disabled?: boolean
}

export type SelectProps = Omit<SelectHTMLAttributes<HTMLSelectElement>, "size"> &
	VariantProps<typeof selectVariants> & {
		label?: string
		hint?: string
		error?: string
		placeholder?: string
		options?: ReadonlyArray<SelectOption>
		wrapperClassName?: string
	}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
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
			placeholder,
			options,
			children,
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

		const selectField = (
			<div className="relative">
				<select
					ref={ref}
					id={id}
					aria-invalid={ariaInvalid ?? (error ? true : undefined)}
					aria-describedby={
						describedByIds.length > 0 ? describedByIds.join(" ") : undefined
					}
					className={cn(
						selectVariants({ state: finalState, size }),
						className,
					)}
					{...props}
				>
					{placeholder !== undefined && (
						<option value="" disabled={!props.value}>
							{placeholder}
						</option>
					)}
					{options
						? options.map((opt) => (
								<option
									key={opt.value}
									value={opt.value}
									disabled={opt.disabled}
								>
									{opt.label}
								</option>
							))
						: children}
				</select>
				{/* Decorative chevron — pointer-events-none so the native
				    click target stays the select itself. */}
				<svg
					aria-hidden="true"
					className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-slate-400"
					width="16"
					height="16"
					viewBox="0 0 16 16"
					fill="none"
					xmlns="http://www.w3.org/2000/svg"
				>
					<path
						d="M4 6l4 4 4-4"
						stroke="currentColor"
						strokeWidth="1.5"
						strokeLinecap="round"
						strokeLinejoin="round"
					/>
				</svg>
			</div>
		)

		const needsWrapper = Boolean(label || error || hint || wrapperClassName)
		if (!needsWrapper) {
			return selectField
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
				{selectField}
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

Select.displayName = "Select"

export { selectVariants }
