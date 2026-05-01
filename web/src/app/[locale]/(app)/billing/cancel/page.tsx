"use client"

import Link from "next/link"
import { XCircle } from "lucide-react"
import { cn } from "@/shared/lib/utils"

/**
 * Landing page Stripe Checkout redirects to when the user clicks back /
 * cancel during the payment flow. No subscription row is ever created
 * for a cancelled checkout — Stripe simply discards the session — so we
 * just acknowledge the cancellation and offer a retry path.
 */
export default function BillingCancelPage() {
	return (
		<div className="flex min-h-[60vh] items-center justify-center p-6">
			<div className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-8 text-center shadow-sm dark:border-slate-700 dark:bg-slate-800/40">
				<div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-slate-200 dark:bg-slate-700">
					<XCircle className="h-7 w-7 text-slate-500 dark:text-slate-300" aria-hidden="true" />
				</div>
				<h1 className="mt-5 text-xl font-semibold text-slate-900 dark:text-white">
					Paiement annulé
				</h1>
				<p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
					Aucun prélèvement n&apos;a été effectué. Tu peux réessayer à tout moment
					depuis le badge Premium en haut de page.
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
			</div>
		</div>
	)
}
