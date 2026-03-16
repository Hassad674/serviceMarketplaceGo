"use client"

import Link from "next/link"
import { useState } from "react"

export default function ProviderDashboardPage() {
  const [referrerMode, setReferrerMode] = useState(false)

  return (
    <div className="space-y-8">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">
            Tableau de bord - Prestataire
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            Vue d&apos;ensemble de votre activite
          </p>
        </div>

        {/* Referrer toggle */}
        <div className="flex items-center gap-3 rounded-xl border border-gray-200 bg-white px-4 py-3">
          <span className="text-sm font-medium text-gray-700">
            Mode apporteur d&apos;affaire
          </span>
          <button
            type="button"
            role="switch"
            aria-checked={referrerMode}
            onClick={() => setReferrerMode((prev) => !prev)}
            className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${
              referrerMode ? "bg-gray-900" : "bg-gray-200"
            }`}
          >
            <span
              className={`pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform ${
                referrerMode ? "translate-x-5" : "translate-x-0"
              }`}
            />
          </button>
        </div>
      </div>

      {/* Stats cards */}
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-xl border border-gray-200 bg-white p-6">
          <p className="text-sm font-medium text-gray-500">Missions en cours</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">0</p>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-6">
          <p className="text-sm font-medium text-gray-500">Messages non lus</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">0</p>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-6">
          <p className="text-sm font-medium text-gray-500">Revenus du mois</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">0 &euro;</p>
        </div>
        {referrerMode && (
          <div className="rounded-xl border border-gray-200 bg-white p-6">
            <p className="text-sm font-medium text-gray-500">
              Commissions du mois
            </p>
            <p className="mt-2 text-3xl font-bold text-gray-900">0 &euro;</p>
          </div>
        )}
      </div>

      {/* Quick links */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Link
          href="/provider/missions"
          className="rounded-xl border border-gray-200 bg-white p-6 transition hover:border-gray-300 hover:shadow-sm"
        >
          <h3 className="font-semibold text-gray-900">Mes Missions</h3>
          <p className="mt-1 text-sm text-gray-500">
            Consultez vos missions en cours
          </p>
        </Link>
        {referrerMode && (
          <Link
            href="/provider/referral"
            className="rounded-xl border border-gray-200 bg-white p-6 transition hover:border-gray-300 hover:shadow-sm"
          >
            <h3 className="font-semibold text-gray-900">
              Apport d&apos;affaire
            </h3>
            <p className="mt-1 text-sm text-gray-500">
              Gerez vos recommandations et commissions
            </p>
          </Link>
        )}
        <Link
          href="/provider/messaging"
          className="rounded-xl border border-gray-200 bg-white p-6 transition hover:border-gray-300 hover:shadow-sm"
        >
          <h3 className="font-semibold text-gray-900">Messagerie</h3>
          <p className="mt-1 text-sm text-gray-500">
            Echangez avec vos clients
          </p>
        </Link>
      </div>
    </div>
  )
}
