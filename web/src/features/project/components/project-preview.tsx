"use client"

import { Calendar, DollarSign, Layers, Tag, Users } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ProjectFormData } from "../types"

type ProjectPreviewProps = {
  formData: ProjectFormData
}

export function ProjectPreview({ formData }: ProjectPreviewProps) {
  const t = useTranslations("projects")

  const totalBudget = computeTotalBudget(formData)
  const hasTitle = formData.title.trim().length > 0
  const hasDescription = formData.description.trim().length > 0
  const hasMilestones = formData.paymentType === "escrow"
    && formData.escrowStructure === "milestone"
    && formData.milestones.some((m) => m.title.trim().length > 0)

  return (
    <div
      className={cn(
        "sticky top-6 rounded-xl border border-gray-200 dark:border-gray-700",
        "bg-gray-50 dark:bg-gray-900 p-5",
      )}
    >
      <h3 className="text-sm font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
        {t("preview")}
      </h3>

      <div className="mt-4 space-y-4">
        {/* Title */}
        <div>
          <h4 className="text-lg font-bold text-gray-900 dark:text-white">
            {hasTitle ? formData.title : (
              <span className="text-gray-300 dark:text-gray-600">
                {t("projectTitlePlaceholder")}
              </span>
            )}
          </h4>
          {hasDescription && (
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400 line-clamp-3">
              {formData.description}
            </p>
          )}
        </div>

        {/* Divider */}
        <div className="border-t border-gray-200 dark:border-gray-700" />

        {/* Budget */}
        <PreviewRow
          icon={DollarSign}
          label={t("amount")}
          value={totalBudget > 0 ? `${totalBudget.toLocaleString("fr-FR")} \u20AC` : "\u2014"}
        />

        {/* Payment type */}
        <PreviewRow
          icon={Layers}
          label={t("paymentType")}
          value={formData.paymentType === "escrow" ? t("escrowPayments") : t("invoiceBilling")}
        />

        {/* Timeline */}
        <PreviewRow
          icon={Calendar}
          label={t("startDate")}
          value={formData.startDate || "\u2014"}
        />
        <PreviewRow
          icon={Calendar}
          label={t("deadline")}
          value={formData.isOngoing ? t("ongoing") : (formData.deadline || "\u2014")}
        />

        {/* Who can apply */}
        <PreviewRow
          icon={Users}
          label={t("whoCanApply")}
          value={getApplicantLabel(formData.applicantType, t)}
        />

        {/* Skills */}
        {formData.skills.length > 0 && (
          <div className="flex items-start gap-2.5">
            <Tag className="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" strokeWidth={1.5} />
            <div>
              <p className="text-xs font-medium text-gray-400 dark:text-gray-500">
                {t("requiredSkills")}
              </p>
              <div className="mt-1 flex flex-wrap gap-1">
                {formData.skills.map((skill) => (
                  <span
                    key={skill}
                    className="rounded-md bg-gray-200 dark:bg-gray-700 px-2 py-0.5 text-xs font-medium text-gray-700 dark:text-gray-300"
                  >
                    {skill}
                  </span>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* Milestones list */}
        {hasMilestones && (
          <>
            <div className="border-t border-gray-200 dark:border-gray-700" />
            <div>
              <p className="text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
                {t("milestone")}s
              </p>
              <div className="mt-2 space-y-2">
                {formData.milestones
                  .filter((m) => m.title.trim().length > 0)
                  .map((milestone, index) => (
                    <div
                      key={milestone.id}
                      className={cn(
                        "flex items-center justify-between rounded-lg",
                        "bg-white dark:bg-gray-800 px-3 py-2 border border-gray-100 dark:border-gray-700",
                      )}
                    >
                      <div className="flex items-center gap-2 min-w-0">
                        <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-rose-100 dark:bg-rose-500/20 text-[10px] font-bold text-rose-600 dark:text-rose-400">
                          {index + 1}
                        </span>
                        <span className="truncate text-sm text-gray-700 dark:text-gray-300">
                          {milestone.title}
                        </span>
                      </div>
                      {milestone.amount && (
                        <span className="ml-2 shrink-0 text-sm font-semibold text-gray-900 dark:text-white">
                          {Number(milestone.amount).toLocaleString("fr-FR")} &euro;
                        </span>
                      )}
                    </div>
                  ))}
              </div>
            </div>
          </>
        )}

        {/* Negotiable badge */}
        {formData.isNegotiable && (
          <div className="rounded-lg bg-amber-50 dark:bg-amber-500/10 px-3 py-2 text-center">
            <span className="text-xs font-medium text-amber-700 dark:text-amber-400">
              {t("negotiable")}
            </span>
          </div>
        )}
      </div>
    </div>
  )
}

type PreviewRowProps = {
  icon: React.ElementType
  label: string
  value: string
}

function PreviewRow({ icon: Icon, label, value }: PreviewRowProps) {
  return (
    <div className="flex items-center gap-2.5">
      <Icon className="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" strokeWidth={1.5} />
      <div className="min-w-0">
        <p className="text-xs font-medium text-gray-400 dark:text-gray-500">{label}</p>
        <p className="truncate text-sm font-medium text-gray-700 dark:text-gray-300">{value}</p>
      </div>
    </div>
  )
}

function computeTotalBudget(data: ProjectFormData): number {
  if (data.paymentType === "escrow") {
    if (data.escrowStructure === "one-time") {
      return Number(data.oneTimeAmount) || 0
    }
    return data.milestones.reduce(
      (sum, m) => sum + (Number(m.amount) || 0),
      0,
    )
  }
  return Number(data.invoiceAmount) || 0
}

function getApplicantLabel(
  type: string,
  t: (key: string) => string,
): string {
  if (type === "freelancers") return t("freelancersOnly")
  if (type === "agencies") return t("agenciesOnly")
  return t("freelancersAndAgencies")
}
