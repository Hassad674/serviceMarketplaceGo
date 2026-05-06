"use client"

import { forwardRef, type ButtonHTMLAttributes } from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/shared/lib/utils"

/**
 * Button — the project's single source of truth for clickable
 * actions in the web app. Mirrors the admin primitive and ports the
 * Soleil v2 visual language (corail primary, calm shadows, sable
 * borders for outline/secondary). Press feedback stays
 * `active:scale-[0.98]` and tactile sizes 8/9/10.
 *
 * Variants:
 *   - primary    — solid corail-deep CTA on shadow-sm, hover lightens
 *                  back to corail (--primary), active presses. The
 *                  base bg is `--primary-deep` (#c43a26) so white text
 *                  on the resting state passes WCAG AA contrast
 *                  (5.83:1) — corail (#e85d4a) only reaches 3.45:1
 *                  which fails AA for normal-size labels. The brand
 *                  identity stays warm (corail family) but at the
 *                  AA-compliant value.
 *   - secondary  — primary-soft pill (rose pâle) for low-emphasis actions
 *   - outline    — sable border on ivoire surface for card actions
 *   - ghost      — no chrome, primary-soft hover, used in icon buttons
 *   - destructive — corail foncé for irreversible actions
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
		"focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/30 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
		"disabled:pointer-events-none disabled:opacity-50",
	),
	{
		variants: {
			variant: {
				primary:
					"bg-primary-deep text-white shadow-sm hover:bg-primary active:scale-[0.98]",
				secondary:
					"bg-secondary text-secondary-foreground hover:bg-secondary/80",
				outline:
					"border border-border bg-card text-foreground hover:bg-muted hover:border-border-strong",
				ghost:
					"text-foreground hover:bg-muted",
				destructive:
					"bg-destructive text-destructive-foreground shadow-sm hover:bg-destructive/90 active:scale-[0.98]",
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
