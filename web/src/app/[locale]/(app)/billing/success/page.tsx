"use client"

import { useEffect } from "react"
import Link from "next/link"
import { CheckCircle2 } from "lucide-react"
import { useQueryClient } from "@tanstack/react-query"
import { cn } from "@/shared/lib/utils"

/**
 * Landing page Stripe Checkout redirects to on successful payment.
 *
 * The subscription webhook activates the DB row asynchronously, so when
 * this page first mounts the subscription cache may still contain "free".
 * We invalidate the subscription queries on mount so the next read (badge
 * re-render on dashboard return, for instance) refetches and reflects
 * Premium. A short lingering delay between payment and badge flip is
 * acceptable — the cache TTL is 60 s and the webhook usually lands in
 * under 2 s on Stripe test mode.
 */
export default function BillingSuccessPage() {
	const queryClient = useQueryClient()

	useEffect(() => {
		queryClient.invalidateQueries({ queryKey: ["subscription"] })
	}, [queryClient])

	return (
		<div className="flex min-h-[60vh] items-center justify-center p-6">
			<div className="w-full max-w-md rounded-2xl border border-emerald-200 bg-gradient-to-br from-emerald-50 to-white p-8 text-center shadow-sm dark:border-emerald-500/30 dark:from-emerald-500/10 dark:to-slate-900/40">
				<div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-emerald-500">
					<CheckCircle2 className="h-7 w-7 text-white" aria-hidden="true" />
				</div>
				<h1 className="mt-5 text-xl font-semibold text-slate-900 dark:text-white">
					Paiement confirmé
				</h1>
				<p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
					Ton abonnement Premium est actif. Tes prochaines missions ne seront plus
					soumises aux frais de plateforme.
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
			</div>
		</div>
	)
}
