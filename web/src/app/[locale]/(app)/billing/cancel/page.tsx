"use client"

import Link from "next/link"
import { ArrowRight, XCircle } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

/**
 * Landing page Stripe Checkout redirects to when the user clicks back /
 * cancel during the payment flow. No subscription row is ever created
 * for a cancelled checkout — Stripe simply discards the session — so we
 * just acknowledge the cancellation and offer a retry path.
 *
 * Soleil v2 styling: ivoire surface hero card with rounded-2xl,
 * amber-soft warning icon disc, Fraunces title with italic corail
 * accent, corail rounded-full back-to-dashboard CTA. All user-visible
 * strings flow through the `billingProfile.*` i18n namespace.
 */
export default function BillingCancelPage() {
	const t = useTranslations("billingProfile")
	return (
		<div className="flex min-h-[60vh] items-center justify-center p-6">
			<div className="w-full max-w-md rounded-2xl border border-border bg-surface p-8 text-center">
				<p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
					{t("billingProfile_w20_cancelEyebrow")}
				</p>
				<div className="mx-auto mt-4 flex h-16 w-16 items-center justify-center rounded-full bg-amber-soft">
					<XCircle className="h-8 w-8 text-warning" aria-hidden="true" />
				</div>
				<h1 className="mt-5 font-serif text-[26px] font-medium leading-tight tracking-[-0.02em] text-foreground">
					{t("billingProfile_w20_cancelTitlePrefix")}{" "}
					<span className="italic text-primary">
						{t("billingProfile_w20_cancelTitleAccent")}
					</span>
				</h1>
				<p className="mt-3 text-sm leading-relaxed text-muted-foreground">
					{t("billingProfile_w20_cancelBody")}
				</p>
				<Link
					href="/dashboard"
					className={cn(
						"mt-7 inline-flex w-full items-center justify-center gap-2 rounded-full",
						"bg-primary px-5 py-3 text-sm font-bold text-primary-foreground",
						"transition-all duration-200 ease-out",
						"hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)]",
						"active:scale-[0.98]",
						"focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
					)}
				>
					{t("billingProfile_w20_backToDashboardCta")}
					<ArrowRight className="h-4 w-4" aria-hidden="true" />
				</Link>
			</div>
		</div>
	)
}
