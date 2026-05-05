"use client"

import { Loader2, X, UserMinus } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRemoveMember } from "../hooks/use-team"
import type { TeamMember } from "../types"

// Soleil v2 — Remove member confirmation. Stateless dialog: the
// mutation + its loading state live in the hook. Operator accounts
// get deleted server-side; marketplace-owner accounts just lose
// their membership.

type RemoveMemberDialogProps = {
  open: boolean
  onClose: () => void
  orgID: string
  member: TeamMember
}

export function RemoveMemberDialog({
  open,
  onClose,
  orgID,
  member,
}: RemoveMemberDialogProps) {
  const t = useTranslations("team")
  const mutation = useRemoveMember(orgID, member.user_id)

  if (!open) return null

  const displayName =
    member.user?.display_name ||
    `${member.user?.first_name ?? ""} ${member.user?.last_name ?? ""}`.trim() ||
    t("memberFallbackName")

  function handleConfirm() {
    mutation.mutate(undefined, {
      onSuccess: () => onClose(),
    })
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(42,31,21,0.45)] p-4 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="animate-scale-in w-full max-w-md rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-6 shadow-[var(--shadow-card-strong)]"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-start justify-between gap-3">
          <div className="flex items-start gap-3">
            <span
              aria-hidden="true"
              className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[var(--primary-soft)] text-[var(--primary-deep)]"
            >
              <UserMinus className="h-5 w-5" strokeWidth={1.8} />
            </span>
            <h3 className="font-serif text-[20px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
              {t("removeMemberTitle")}
            </h3>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label={t("cancel")}
            className="rounded-full p-1 text-[var(--muted-foreground)] transition-colors hover:bg-[var(--background)] hover:text-[var(--foreground)]"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <p className="font-serif text-[14px] italic text-[var(--muted-foreground)]">
          {t("removeMemberConfirm", { name: displayName })}
        </p>

        {mutation.isError && (
          <p className="mt-3 rounded-xl border border-[var(--primary-soft)] bg-[var(--primary-soft)] px-3 py-2 text-[13px] text-[var(--primary-deep)]">
            {t("errors.generic")}
          </p>
        )}

        <div className="mt-6 flex justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            disabled={mutation.isPending}
            className="rounded-full border border-[var(--border)] bg-[var(--surface)] px-4 py-2 text-[13px] font-semibold text-[var(--foreground)] transition-colors hover:border-[var(--border-strong)] hover:bg-[var(--background)] disabled:opacity-50"
          >
            {t("cancel")}
          </button>
          <button
            type="button"
            onClick={handleConfirm}
            disabled={mutation.isPending}
            className="inline-flex items-center gap-2 rounded-full bg-[var(--primary)] px-4 py-2 text-[13px] font-semibold text-[var(--primary-foreground)] shadow-[var(--shadow-message)] transition-colors hover:bg-[var(--primary-deep)] disabled:opacity-50"
          >
            {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("remove")}
          </button>
        </div>
      </div>
    </div>
  )
}
