"use client"

import { useState } from "react"
import { Loader2, X, AlertTriangle } from "lucide-react"
import { useTranslations } from "next-intl"
import { useInitiateTransfer } from "../hooks/use-team"
import type { TeamMember } from "../types"

// Soleil v2 — Initiates the 2-step ownership transfer flow. Only the
// current Owner sees the trigger that opens this modal (gated upstream
// via permissions). The target MUST be an existing Admin in the org —
// we filter the list here so the operator can't pick an invalid target.

type TransferOwnershipModalProps = {
  open: boolean
  onClose: () => void
  orgID: string
  members: TeamMember[]
  currentOwnerID: string
}

export function TransferOwnershipModal({
  open,
  onClose,
  orgID: _orgID,
  members,
  currentOwnerID,
}: TransferOwnershipModalProps) {
  const t = useTranslations("team")
  const mutation = useInitiateTransfer(_orgID)

  const eligible = members.filter(
    (m) => m.role === "admin" && m.user_id !== currentOwnerID,
  )

  const [targetUserID, setTargetUserID] = useState("")

  if (!open) return null

  function handleConfirm() {
    if (!targetUserID) return
    mutation.mutate(
      { target_user_id: targetUserID },
      {
        onSuccess: () => {
          setTargetUserID("")
          onClose()
        },
      },
    )
  }

  const inputBase =
    "w-full rounded-xl border border-[var(--border)] bg-[var(--surface)] px-3 py-2.5 text-[14px] text-[var(--foreground)] focus:border-[var(--primary)] focus:outline-none focus:ring-2 focus:ring-[var(--primary-soft)]"

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-[rgba(42,31,21,0.45)] p-4 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="animate-scale-in w-full max-w-lg rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-6 shadow-[var(--shadow-card-strong)]"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-start justify-between gap-3">
          <div className="flex items-start gap-3">
            <span
              aria-hidden="true"
              className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[var(--amber-soft)] text-[var(--warning)]"
            >
              <AlertTriangle className="h-5 w-5" strokeWidth={1.8} />
            </span>
            <h3 className="font-serif text-[20px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
              {t("transferTitle")}
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

        <div className="mb-4 rounded-xl border border-[var(--amber-soft)] bg-[var(--amber-soft)]/60 p-3 font-serif text-[12.5px] italic text-[var(--warning)]">
          {t("transferWarning")}
        </div>

        <div className="space-y-4">
          {eligible.length === 0 ? (
            <div className="rounded-xl border border-dashed border-[var(--border)] bg-[var(--background)] p-4 text-center font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
              {t("transferNoEligible")}
            </div>
          ) : (
            <div>
              <label
                htmlFor="transfer-target"
                className="mb-1.5 block text-[12px] font-semibold uppercase tracking-[0.04em] text-[var(--muted-foreground)]"
              >
                {t("transferTargetLabel")}
              </label>
              <select
                id="transfer-target"
                value={targetUserID}
                onChange={(e) => setTargetUserID(e.target.value)}
                className={`${inputBase} cursor-pointer`}
              >
                <option value="">{t("transferSelectPlaceholder")}</option>
                {eligible.map((m) => {
                  const name =
                    m.user?.display_name ||
                    `${m.user?.first_name ?? ""} ${m.user?.last_name ?? ""}`.trim() ||
                    m.user_id.slice(0, 8)
                  return (
                    <option key={m.user_id} value={m.user_id}>
                      {name} {m.title ? `· ${m.title}` : ""}
                    </option>
                  )
                })}
              </select>
            </div>
          )}

          {mutation.isError && (
            <p className="rounded-xl border border-[var(--primary-soft)] bg-[var(--primary-soft)] px-3 py-2 text-[13px] text-[var(--primary-deep)]">
              {t("errors.generic")}
            </p>
          )}
        </div>

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
            disabled={mutation.isPending || !targetUserID || eligible.length === 0}
            className="inline-flex items-center gap-2 rounded-full bg-[var(--primary)] px-4 py-2 text-[13px] font-semibold text-[var(--primary-foreground)] shadow-[var(--shadow-message)] transition-colors hover:bg-[var(--primary-deep)] disabled:opacity-50"
          >
            {mutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("transferConfirm")}
          </button>
        </div>
      </div>
    </div>
  )
}
