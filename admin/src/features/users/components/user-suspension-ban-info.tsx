import { formatDate } from "@/shared/lib/utils"

// Two information banners that render when a user is currently under a
// moderation action. They appear between the main info cards and the
// action buttons so the admin reads the reason before deciding what
// to do next.

type SuspensionInfoCardProps = {
  reason: string
  suspendedAt?: string
  expiresAt?: string
}

export function SuspensionInfoCard({ reason, suspendedAt, expiresAt }: SuspensionInfoCardProps) {
  return (
    <div className="rounded-xl border border-amber-200 bg-amber-50 p-6">
      <h2 className="mb-3 text-sm font-semibold text-amber-800">Suspension active</h2>
      <dl className="space-y-2 text-sm">
        <div>
          <dt className="text-amber-600">Raison</dt>
          <dd className="font-medium text-amber-900">{reason}</dd>
        </div>
        {suspendedAt && (
          <div>
            <dt className="text-amber-600">Suspendu le</dt>
            <dd className="text-amber-900">{formatDate(suspendedAt)}</dd>
          </div>
        )}
        {expiresAt && (
          <div>
            <dt className="text-amber-600">Expire le</dt>
            <dd className="text-amber-900">{formatDate(expiresAt)}</dd>
          </div>
        )}
      </dl>
    </div>
  )
}

type BanInfoCardProps = {
  reason: string
  bannedAt?: string
}

export function BanInfoCard({ reason, bannedAt }: BanInfoCardProps) {
  return (
    <div className="rounded-xl border border-red-200 bg-red-50 p-6">
      <h2 className="mb-3 text-sm font-semibold text-red-800">Bannissement actif</h2>
      <dl className="space-y-2 text-sm">
        <div>
          <dt className="text-red-600">Raison</dt>
          <dd className="font-medium text-red-900">{reason}</dd>
        </div>
        {bannedAt && (
          <div>
            <dt className="text-red-600">Banni le</dt>
            <dd className="text-red-900">{formatDate(bannedAt)}</dd>
          </div>
        )}
      </dl>
    </div>
  )
}
