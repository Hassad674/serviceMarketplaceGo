"use client"

import { Euro, Calendar, Paperclip, User } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ProposalFormData } from "../types"

interface ProposalPreviewProps {
  formData: ProposalFormData
  recipientName: string
}

export function ProposalPreview({ formData, recipientName }: ProposalPreviewProps) {
  const t = useTranslations("proposal")

  const amount = Number(formData.amount) || 0
  const hasDeadline = formData.deadline.length > 0
  const formattedDeadline = hasDeadline
    ? new Intl.DateTimeFormat("fr-FR", { day: "numeric", month: "long", year: "numeric" }).format(new Date(formData.deadline))
    : null

  return (
    <div
      className={cn(
        "rounded-2xl border border-gray-200 bg-white p-6",
        "dark:border-gray-700 dark:bg-gray-800/80",
        "shadow-sm",
      )}
    >
      <h3 className="text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
        {t("proposalPreview")}
      </h3>

      <div className="mt-4 space-y-4">
        {/* Title */}
        <p className="text-lg font-bold text-gray-900 dark:text-white">
          {formData.title || t("proposalTitlePlaceholder")}
        </p>

        <div className="border-t border-gray-100 dark:border-gray-700" />

        {/* Amount */}
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-rose-50 dark:bg-rose-500/10">
            <Euro className="h-4 w-4 text-rose-600 dark:text-rose-400" strokeWidth={1.5} />
          </div>
          <div>
            <p className="text-xs text-gray-400 dark:text-gray-500">{t("proposalAmount")}</p>
            <p className="text-sm font-bold text-gray-900 dark:text-white">
              {amount > 0 ? `${amount.toLocaleString("fr-FR")} \u20AC` : "\u2014"}
            </p>
          </div>
        </div>

        {/* Deadline */}
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-50 dark:bg-blue-500/10">
            <Calendar className="h-4 w-4 text-blue-600 dark:text-blue-400" strokeWidth={1.5} />
          </div>
          <div>
            <p className="text-xs text-gray-400 dark:text-gray-500">{t("proposalDeadline")}</p>
            <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
              {formattedDeadline ?? "\u2014"}
            </p>
          </div>
        </div>

        {/* Documents */}
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-amber-50 dark:bg-amber-500/10">
            <Paperclip className="h-4 w-4 text-amber-600 dark:text-amber-400" strokeWidth={1.5} />
          </div>
          <div>
            <p className="text-xs text-gray-400 dark:text-gray-500">{t("proposalDocuments")}</p>
            <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
              {formData.files.length}
            </p>
          </div>
        </div>

        <div className="border-t border-gray-100 dark:border-gray-700" />

        {/* Recipient */}
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-purple-50 dark:bg-purple-500/10">
            <User className="h-4 w-4 text-purple-600 dark:text-purple-400" strokeWidth={1.5} />
          </div>
          <div>
            <p className="text-xs text-gray-400 dark:text-gray-500">{t("proposalRecipient")}</p>
            <p className="text-sm font-medium text-gray-700 dark:text-gray-300">
              {recipientName || "\u2014"}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
