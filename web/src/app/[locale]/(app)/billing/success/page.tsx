"use client"

import { useEffect, useState } from "react"
import Link from "next/link"
import { CheckCircle2, Loader2 } from "lucide-react"
import { useQueryClient } from "@tanstack/react-query"
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
			<div
				className={cn(
					"w-full max-w-md rounded-2xl border p-8 text-center shadow-sm",
					status === "pending"
						? "border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800/40"
						: status === "ready"
							? "border-emerald-200 bg-gradient-to-br from-emerald-50 to-white dark:border-emerald-500/30 dark:from-emerald-500/10 dark:to-slate-900/40"
							: "border-amber-200 bg-amber-50/60 dark:border-amber-500/30 dark:bg-amber-500/10",
				)}
			>
				{status === "pending" ? <PendingContent /> : null}
				{status === "ready" ? <ReadyContent /> : null}
				{status === "timeout" ? <TimeoutContent /> : null}
			</div>
		</div>
	)
}

function PendingContent() {
	return (
		<>
			<div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-slate-200 dark:bg-slate-700">
				<Loader2
					className="h-7 w-7 animate-spin text-slate-500 dark:text-slate-300"
					aria-hidden="true"
				/>
			</div>
			<h1 className="mt-5 text-xl font-semibold text-slate-900 dark:text-white">
				Activation en cours…
			</h1>
			<p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
				Paiement confirmé. Nous finalisons ton abonnement Premium — ça prend
				quelques secondes.
			</p>
		</>
	)
}

function ReadyContent() {
	return (
		<>
			<div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-emerald-500">
				<CheckCircle2 className="h-7 w-7 text-white" aria-hidden="true" />
			</div>
			<h1 className="mt-5 text-xl font-semibold text-slate-900 dark:text-white">
				Premium activé
			</h1>
			<p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
				Tes prochaines missions ne seront plus soumises aux frais de plateforme.
			</p>
			<Link
				href="/dashboard"
				className={cn(
					"mt-6 inline-flex w-full items-center justify-center rounded-full",
					"bg-gradient-to-r from-rose-500 to-rose-600 px-4 py-3 text-sm font-semibold text-white",
					"transition-all duration-200 hover:shadow-glow active:scale-[0.98]",
					"focus:outline-none focus:ring-2 focus:ring-rose-500/40",
				)}
			>
				Retour au tableau de bord
			</Link>
		</>
	)
}

function TimeoutContent() {
	return (
		<>
			<div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-500/20">
				<CheckCircle2
					className="h-7 w-7 text-amber-600 dark:text-amber-400"
					aria-hidden="true"
				/>
			</div>
			<h1 className="mt-5 text-xl font-semibold text-slate-900 dark:text-white">
				Paiement confirmé
			</h1>
			<p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
				Ton paiement est bien passé. L&apos;activation Premium prend parfois un peu
				plus de temps — rafraîchis le tableau de bord dans une minute.
			</p>
			<Link
				href="/dashboard"
				className={cn(
					"mt-6 inline-flex w-full items-center justify-center rounded-full",
					"border border-slate-300 bg-white px-4 py-3 text-sm font-semibold text-slate-700",
					"transition-colors duration-200 hover:bg-slate-50",
					"dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700",
					"focus:outline-none focus:ring-2 focus:ring-rose-500/40",
				)}
			>
				Retour au tableau de bord
			</Link>
		</>
	)
}
