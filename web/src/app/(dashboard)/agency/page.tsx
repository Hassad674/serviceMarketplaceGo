import Link from "next/link"

export default function AgencyDashboardPage() {
  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">
          Tableau de bord - Agence
        </h1>
        <p className="mt-1 text-sm text-gray-500">
          Vue d&apos;ensemble de votre activite
        </p>
      </div>

      {/* Stats cards */}
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-xl border border-gray-200 bg-white p-6">
          <p className="text-sm font-medium text-gray-500">Missions en cours</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">0</p>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-6">
          <p className="text-sm font-medium text-gray-500">Membres d&apos;equipe</p>
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
      </div>

      {/* Quick links */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Link
          href="/agency/missions"
          className="rounded-xl border border-gray-200 bg-white p-6 transition hover:border-gray-300 hover:shadow-sm"
        >
          <h3 className="font-semibold text-gray-900">Mes Missions</h3>
          <p className="mt-1 text-sm text-gray-500">
            Consultez et gerez vos missions actives
          </p>
        </Link>
        <Link
          href="/agency/team"
          className="rounded-xl border border-gray-200 bg-white p-6 transition hover:border-gray-300 hover:shadow-sm"
        >
          <h3 className="font-semibold text-gray-900">Mon Equipe</h3>
          <p className="mt-1 text-sm text-gray-500">
            Gerez les membres de votre agence
          </p>
        </Link>
        <Link
          href="/agency/messaging"
          className="rounded-xl border border-gray-200 bg-white p-6 transition hover:border-gray-300 hover:shadow-sm"
        >
          <h3 className="font-semibold text-gray-900">Messagerie</h3>
          <p className="mt-1 text-sm text-gray-500">
            Echangez avec vos clients et prestataires
          </p>
        </Link>
      </div>
    </div>
  )
}
