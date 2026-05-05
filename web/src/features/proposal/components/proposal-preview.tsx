"use client"

import { Calendar, Paperclip, User } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ProposalFormData } from "../types"

// Soleil v2 — Live preview of the proposal as the recipient will see it.
// Soleil card with Fraunces title + Geist Mono amount + tabac details.

interface ProposalPreviewProps {
  formData: ProposalFormData
  recipientName: string
}

export function ProposalPreview({ formData, recipientName }: ProposalPreviewProps) {
  const t = useTranslations("proposal")

  const amount = Number(formData.amount) || 0
  const hasDeadline = formData.deadline.length > 0
  const formattedDeadline = hasDeadline
    ? new Intl.DateTimeFormat("fr-FR", {
        day: "numeric",
        month: "long",
        year: "numeric",
      }).format(new Date(formData.deadline))
    : null

  return (
    <div
      className={cn(
        "rounded-2xl border border-border bg-card p-6",
      )}
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <p className="font-mono text-[10.5px] font-bold uppercase tracking-[0.12em] text-primary">
        {t("proposalPreview")}
      </p>

      <div className="mt-4 space-y-4">
        {/* Title */}
        <p className="font-serif text-[20px] font-medium leading-tight tracking-[-0.015em] text-foreground">
          {formData.title || t("proposalTitlePlaceholder")}
        </p>

        <div className="border-t border-dashed border-border" />

        {/* Amount — Geist Mono */}
        <div className="flex items-center justify-between gap-3">
          <span className="font-mono text-[10.5px] font-bold uppercase tracking-[0.08em] text-subtle-foreground">
            {t("proposalAmount")}
          </span>
          <span className="font-mono text-[20px] font-bold text-foreground">
            {amount > 0 ? `${amount.toLocaleString("fr-FR")} €` : "—"}
          </span>
        </div>

        <div className="border-t border-dashed border-border" />

        {/* Deadline */}
        <PreviewLine
          icon={Calendar}
          label={t("proposalDeadline")}
          value={formattedDeadline ?? "—"}
        />

        {/* Documents */}
        <PreviewLine
          icon={Paperclip}
          label={t("proposalDocuments")}
          value={String(formData.files.length)}
        />

        <div className="border-t border-dashed border-border" />

        {/* Recipient */}
        <PreviewLine
          icon={User}
          label={t("proposalRecipient")}
          value={recipientName || "—"}
        />
      </div>
    </div>
  )
}

interface PreviewLineProps {
  icon: React.ElementType
  label: string
  value: string
}

function PreviewLine({ icon: Icon, label, value }: PreviewLineProps) {
  return (
    <div className="flex items-center gap-3">
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary-soft text-primary">
        <Icon className="h-3.5 w-3.5" strokeWidth={1.7} aria-hidden="true" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="font-mono text-[10.5px] font-bold uppercase tracking-[0.08em] text-subtle-foreground">
          {label}
        </p>
        <p className="truncate text-[13.5px] font-medium text-foreground">
          {value}
        </p>
      </div>
    </div>
  )
}
