"use client"

import { forwardRef, type HTMLAttributes } from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "@/shared/lib/utils"

/**
 * Card — surface primitive for grouping related content.
 *
 * Two variants per the design system:
 *   - default     : static surface, slate-100 border, shadow-sm
 *   - interactive : adds hover:shadow-md, hover:border-rose-200,
 *                   hover:-translate-y-0.5 — used on clickable cards
 *                   (search results, listing tiles, dashboard CTAs)
 *
 * The padding `p-6` is applied to `Card` itself, which matches the
 * design-system "16/24" rhythm. When you need a Card with header
 * and content, prefer the subcomponents (`CardHeader`, `CardContent`,
 * `CardFooter`) which manage their own padding so you can put a
 * `p-0` on `Card` and let the subcomponents own the spacing.
 */
const cardVariants = cva(
	cn(
		"rounded-2xl border bg-white text-slate-900",
		"transition-all duration-200 ease-out",
		"dark:bg-slate-900 dark:text-slate-100",
	),
	{
		variants: {
			variant: {
				default:
					"border-slate-100 shadow-sm dark:border-slate-800",
				interactive:
					"border-slate-100 shadow-sm cursor-pointer hover:shadow-md hover:border-rose-200 hover:-translate-y-0.5 dark:border-slate-800 dark:hover:border-rose-800/50",
			},
			padding: {
				none: "",
				sm: "p-4",
				md: "p-6",
				lg: "p-8",
			},
		},
		defaultVariants: {
			variant: "default",
			padding: "md",
		},
	},
)

export type CardProps = HTMLAttributes<HTMLDivElement> &
	VariantProps<typeof cardVariants>

export const Card = forwardRef<HTMLDivElement, CardProps>(
	({ className, variant, padding, ...props }, ref) => (
		<div
			ref={ref}
			className={cn(cardVariants({ variant, padding }), className)}
			{...props}
		/>
	),
)
Card.displayName = "Card"

export const CardHeader = forwardRef<
	HTMLDivElement,
	HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
	<div
		ref={ref}
		className={cn("flex flex-col gap-1.5 px-6 pt-6", className)}
		{...props}
	/>
))
CardHeader.displayName = "CardHeader"

export const CardTitle = forwardRef<
	HTMLHeadingElement,
	HTMLAttributes<HTMLHeadingElement>
>(({ className, ...props }, ref) => (
	<h3
		ref={ref}
		className={cn(
			"text-lg font-semibold leading-tight text-slate-900 dark:text-slate-50",
			className,
		)}
		{...props}
	/>
))
CardTitle.displayName = "CardTitle"

export const CardDescription = forwardRef<
	HTMLParagraphElement,
	HTMLAttributes<HTMLParagraphElement>
>(({ className, ...props }, ref) => (
	<p
		ref={ref}
		className={cn("text-sm text-slate-500 dark:text-slate-400", className)}
		{...props}
	/>
))
CardDescription.displayName = "CardDescription"

export const CardContent = forwardRef<
	HTMLDivElement,
	HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
	<div ref={ref} className={cn("px-6 py-4", className)} {...props} />
))
CardContent.displayName = "CardContent"

export const CardFooter = forwardRef<
	HTMLDivElement,
	HTMLAttributes<HTMLDivElement>
>(({ className, ...props }, ref) => (
	<div
		ref={ref}
		className={cn(
			"flex items-center border-t border-slate-100 px-6 py-4 dark:border-slate-800",
			className,
		)}
		{...props}
	/>
))
CardFooter.displayName = "CardFooter"

export { cardVariants }
