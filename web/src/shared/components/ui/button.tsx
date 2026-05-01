"use client"

import { forwardRef, type ButtonHTMLAttributes } from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/shared/lib/utils"

/**
 * Button — the project's single source of truth for clickable
 * actions in the web app. Mirrors the admin primitive but pulls in
 * the design-system polish documented in CLAUDE.md (rose primary
 * gradient, shadow-glow on hover, active:scale-[0.98] press feedback,
 * tactile sizes 8/9/10).
 *
 * Variants:
 *   - primary    — gradient-primary CTA, glows on hover, presses on tap
 *   - secondary  — muted grey for low-emphasis actions
 *   - outline    — transparent with border, used in card actions
 *   - ghost      — no chrome, used in icon buttons / list rows
 *   - destructive — red for irreversible actions (cancel, delete)
 *
 * Sizes:
 *   - sm  (h-8)  — table rows, compact toolbars
 *   - md  (h-9)  — DEFAULT, forms and modals
 *   - lg  (h-10) — primary onboarding CTAs, hero panels
 *
 * The component intentionally forwards `type` so callers always set
 * it explicitly. We do NOT default `type` to "button" — leaving the
 * default lets accessibility tests catch missing types instead of
 * silently masking them.
 */
const buttonVariants = cva(
	cn(
		"inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-lg font-medium",
		"transition-all duration-200 ease-out",
		"focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-rose-500/50 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
		"disabled:pointer-events-none disabled:opacity-50",
	),
	{
		variants: {
			variant: {
				primary:
					"gradient-primary text-white shadow-sm hover:shadow-glow active:scale-[0.98]",
				secondary:
					"bg-slate-100 text-slate-900 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-100 dark:hover:bg-slate-700",
				outline:
					"border border-slate-200 bg-white text-slate-900 hover:bg-slate-50 hover:border-rose-200 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-100 dark:hover:bg-slate-800",
				ghost:
					"text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800",
				destructive:
					"bg-red-500 text-white shadow-sm hover:bg-red-600 active:scale-[0.98]",
			},
			size: {
				sm: "h-8 px-3 text-xs",
				md: "h-9 px-4 text-sm",
				lg: "h-10 px-6 text-sm",
				/**
				 * `auto` opts out of size classes entirely — useful when the
				 * caller already controls height/padding (icon menus, list
				 * rows, special layouts where forcing h-9 would break the
				 * design). Most common during the migration from raw
				 * `<button>` since callers carried bespoke spacing.
				 */
				auto: "",
			},
		},
		defaultVariants: {
			variant: "primary",
			size: "md",
		},
	},
)

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> &
	VariantProps<typeof buttonVariants>

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
	({ className, variant, size, type, ...props }, ref) => {
		return (
			<button
				ref={ref}
				type={type}
				className={cn(buttonVariants({ variant, size }), className)}
				{...props}
			/>
		)
	},
)

Button.displayName = "Button"

export { buttonVariants }
