import Link from "next/link"

export default function EnterpriseDashboardPage() {
  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">
          Tableau de bord - Entreprise
        </h1>
        <p className="mt-1 text-sm text-gray-500">
          Vue d&apos;ensemble de vos projets
        </p>
      </div>

      {/* Stats cards */}
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-xl border border-gray-200 bg-white p-6">
          <p className="text-sm font-medium text-gray-500">Projets actifs</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">0</p>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-6">
          <p className="text-sm font-medium text-gray-500">Candidatures recues</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">0</p>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-6">
          <p className="text-sm font-medium text-gray-500">Messages non lus</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">0</p>
        </div>
        <div className="rounded-xl border border-gray-200 bg-white p-6">
          <p className="text-sm font-medium text-gray-500">Depenses du mois</p>
          <p className="mt-2 text-3xl font-bold text-gray-900">0 &euro;</p>
        </div>
      </div>

      {/* Quick links */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Link
          href="/enterprise/projects"
          className="rounded-xl border border-gray-200 bg-white p-6 transition hover:border-gray-300 hover:shadow-sm"
        >
          <h3 className="font-semibold text-gray-900">Mes Projets</h3>
          <p className="mt-1 text-sm text-gray-500">
            Publiez et gerez vos projets
          </p>
        </Link>
        <Link
          href="/enterprise/search"
          className="rounded-xl border border-gray-200 bg-white p-6 transition hover:border-gray-300 hover:shadow-sm"
        >
          <h3 className="font-semibold text-gray-900">Rechercher</h3>
          <p className="mt-1 text-sm text-gray-500">
            Trouvez les meilleurs prestataires
          </p>
        </Link>
        <Link
          href="/enterprise/messaging"
          className="rounded-xl border border-gray-200 bg-white p-6 transition hover:border-gray-300 hover:shadow-sm"
        >
          <h3 className="font-semibold text-gray-900">Messagerie</h3>
          <p className="mt-1 text-sm text-gray-500">
            Echangez avec vos prestataires
          </p>
        </Link>
      </div>
    </div>
  )
}
