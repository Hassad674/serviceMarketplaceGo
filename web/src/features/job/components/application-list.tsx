"use client"

import { FileText, Trash2, Briefcase, Calendar, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { useMyApplications, useWithdrawApplication } from "../hooks/use-job-applications"
import { Button } from "@/shared/components/ui/button"

export function ApplicationList() {
  const t = useTranslations("opportunity")
  const { data, isLoading } = useMyApplications()
  const withdrawMutation = useWithdrawApplication()

  if (isLoading) {
    return (
      <div className="space-y-4">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="h-28 rounded-2xl bg-slate-100 animate-shimmer dark:bg-slate-800" />
        ))}
      </div>
    )
  }

  if (!data || data.data.length === 0) {
    return (
      <div className="text-center py-12">
        <FileText className="mx-auto h-10 w-10 text-slate-300 mb-3" />
        <p className="text-sm text-slate-500 dark:text-slate-400">{t("noApplications")}</p>
        <Link href="/opportunities" className="mt-3 inline-block text-sm font-medium text-rose-600 hover:underline">
          {t("browse")}
        </Link>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {data.data.map(({ application, job }) => (
        <div
          key={application.id}
          className="rounded-2xl border border-slate-100 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800/80"
        >
          <div className="flex items-start justify-between gap-3">
            <div className="flex-1 min-w-0">
              <Link href={`/opportunities/${job.id}`} className="hover:underline">
                <h3 className="text-base font-semibold text-slate-900 dark:text-white truncate">{job.title}</h3>
              </Link>
              <div className="flex items-center gap-3 mt-1 text-xs text-slate-500 dark:text-slate-400">
                <span className="flex items-center gap-1"><Briefcase className="h-3.5 w-3.5" />{job.min_budget.toLocaleString("fr-FR")}€ - {job.max_budget.toLocaleString("fr-FR")}€</span>
                <span className="flex items-center gap-1"><Calendar className="h-3.5 w-3.5" />{t("applied")} {new Date(application.created_at).toLocaleDateString("fr-FR")}</span>
              </div>
              <p className="mt-2 text-sm text-slate-600 dark:text-slate-300 line-clamp-2">{application.message}</p>
            </div>
            <Button variant="ghost" size="auto"
              type="button"
              onClick={() => {
                if (confirm(t("withdrawConfirm"))) {
                  withdrawMutation.mutate(application.id)
                }
              }}
              disabled={withdrawMutation.isPending}
              className="shrink-0 flex items-center gap-1 rounded-lg px-3 py-1.5 text-xs font-medium text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 transition-colors"
            >
              {withdrawMutation.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
              {t("withdraw")}
            </Button>
          </div>
        </div>
      ))}
    </div>
  )
}
