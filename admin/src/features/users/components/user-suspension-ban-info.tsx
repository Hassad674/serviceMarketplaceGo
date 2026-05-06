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
    <div className="rounded-xl border border-[var(--warning)]/30 bg-[var(--amber-soft)] p-6">
      <h2 className="mb-3 text-sm font-semibold text-[var(--warning)]">Suspension active</h2>
      <dl className="space-y-2 text-sm">
        <div>
          <dt className="text-[var(--warning)]/80">Raison</dt>
          <dd className="font-medium text-foreground">{reason}</dd>
        </div>
        {suspendedAt && (
          <div>
            <dt className="text-[var(--warning)]/80">Suspendu le</dt>
            <dd className="text-foreground">{formatDate(suspendedAt)}</dd>
          </div>
        )}
        {expiresAt && (
          <div>
            <dt className="text-[var(--warning)]/80">Expire le</dt>
            <dd className="text-foreground">{formatDate(expiresAt)}</dd>
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
    <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6">
      <h2 className="mb-3 text-sm font-semibold text-destructive">Bannissement actif</h2>
      <dl className="space-y-2 text-sm">
        <div>
          <dt className="text-destructive/80">Raison</dt>
          <dd className="font-medium text-foreground">{reason}</dd>
        </div>
        {bannedAt && (
          <div>
            <dt className="text-destructive/80">Banni le</dt>
            <dd className="text-foreground">{formatDate(bannedAt)}</dd>
          </div>
        )}
      </dl>
    </div>
  )
}
