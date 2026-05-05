"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { ArrowRight, CheckCircle2, Loader2 } from "lucide-react"
import { useQueryClient } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { getMySubscription } from "@/features/subscription/api/subscription-api"
import { subscriptionQueryKey } from "@/features/subscription/hooks/keys"

/**
 * Landing page Stripe Checkout redirects to on successful payment.
 *
 * Stripe fires `customer.subscription.created` asynchronously via webhook,
 * so when this page first mounts the backend may not have persisted the
 * subscription yet. A naive "refetch once and go" would race — it would
 * cache a 404 that poisons the badge for 30 s until the query becomes
 * stale.
 *
 * Approach: actively poll `/api/v1/subscriptions/me` every 1 s until it
 * returns 200 (or we hit the cap at 15 s — enough for every observed
 * webhook round trip in dev + a generous margin). Once we see the
 * subscription, seed it into the TanStack cache so subsequent pages
 * (dashboard, manage modal) read it instantly without another fetch.
 *
 * UX: the card shows a gentle "Activation en cours…" spinner during the
 * poll, then flips to the green confirmation once the subscription
 * lands. The CTA is disabled while pending so the user never reaches
 * the dashboard with a stale free-tier state.
 *
 * Soleil v2 styling: ivoire surface hero card with rounded-2xl,
 * corail-soft icon disc on success / amber-soft on timeout, Fraunces
 * title with italic corail accent, corail rounded-full back-to-dashboard
 * CTA. All user-visible strings flow through the `billingProfile.*` i18n
 * namespace.
 */
export default function BillingSuccessPage() {
	const queryClient = useQueryClient()
	const [status, setStatus] = useState<"pending" | "ready" | "timeout">(
		"pending",
	)

	useEffect(() => {
		let cancelled = false
		const maxAttempts = 15
		const intervalMs = 1000

		async function poll(attempt: number) {
			if (cancelled) return
			try {
				const sub = await getMySubscription()
				if (cancelled) return
				if (sub) {
					// Seed the TanStack cache with the fresh data so the
					// badge and manage-modal pick it up instantly.
					queryClient.setQueryData(subscriptionQueryKey.me(), sub)
					setStatus("ready")
					return
				}
			} catch {
				// Network / 5xx — swallow and retry below. A real 404 is
				// not thrown by `getMySubscription` (it returns null).
			}
			if (attempt >= maxAttempts) {
				setStatus("timeout")
				return
			}
			setTimeout(() => poll(attempt + 1), intervalMs)
		}

		poll(1)
		return () => {
			cancelled = true
		}
	}, [queryClient])

	return (
		<div className="flex min-h-[60vh] items-center justify-center p-6">
			<div className="w-full max-w-md rounded-2xl border border-border bg-surface p-8 text-center">
				{status === "pending" ? <PendingContent /> : null}
				{status === "ready" ? <ReadyContent /> : null}
				{status === "timeout" ? <TimeoutContent /> : null}
			</div>
		</div>
	)
}

function PendingContent() {
	const t = useTranslations("billingProfile")
	return (
		<>
			<p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
				{t("billingProfile_w20_successPendingEyebrow")}
			</p>
			<div className="mx-auto mt-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary-soft">
				<Loader2
					className="h-7 w-7 animate-spin text-primary-deep"
					aria-hidden="true"
				/>
			</div>
			<h1 className="mt-5 font-serif text-[26px] font-medium leading-tight tracking-[-0.02em] text-foreground">
				{t("billingProfile_w20_successPendingTitle")}
			</h1>
			<p className="mt-3 text-sm leading-relaxed text-muted-foreground">
				{t("billingProfile_w20_successPendingBody")}
			</p>
		</>
	)
}

function ReadyContent() {
	const t = useTranslations("billingProfile")
	return (
		<>
			<p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
				{t("billingProfile_w20_successReadyEyebrow")}
			</p>
			<div className="mx-auto mt-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary-soft">
				<CheckCircle2 className="h-8 w-8 text-primary-deep" aria-hidden="true" />
			</div>
			<h1 className="mt-5 font-serif text-[26px] font-medium leading-tight tracking-[-0.02em] text-foreground">
				{t("billingProfile_w20_successReadyTitlePrefix")}{" "}
				<span className="italic text-primary">
					{t("billingProfile_w20_successReadyTitleAccent")}
				</span>
			</h1>
			<p className="mt-3 text-sm leading-relaxed text-muted-foreground">
				{t("billingProfile_w20_successReadyBody")}
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
		</>
	)
}

function TimeoutContent() {
	const t = useTranslations("billingProfile")
	return (
		<>
			<p className="font-mono text-[11px] font-bold uppercase tracking-[0.12em] text-primary">
				{t("billingProfile_w20_successTimeoutEyebrow")}
			</p>
			<div className="mx-auto mt-4 flex h-16 w-16 items-center justify-center rounded-full bg-amber-soft">
				<CheckCircle2
					className="h-7 w-7 text-warning"
					aria-hidden="true"
				/>
			</div>
			<h1 className="mt-5 font-serif text-[26px] font-medium leading-tight tracking-[-0.02em] text-foreground">
				{t("billingProfile_w20_successTimeoutTitlePrefix")}{" "}
				<span className="italic text-primary">
					{t("billingProfile_w20_successTimeoutTitleAccent")}
				</span>
			</h1>
			<p className="mt-3 text-sm leading-relaxed text-muted-foreground">
				{t("billingProfile_w20_successTimeoutBody")}
			</p>
			<Link
				href="/dashboard"
				className={cn(
					"mt-7 inline-flex w-full items-center justify-center gap-2 rounded-full",
					"border border-border-strong bg-surface px-5 py-3 text-sm font-semibold text-foreground",
					"transition-colors duration-200 hover:bg-primary-soft/30",
					"focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
				)}
			>
				{t("billingProfile_w20_backToDashboardCta")}
			</Link>
		</>
	)
}
