"use client"

import { Suspense, useEffect, useState } from "react"
import { useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"
import { confirmDeletion } from "@/features/account/api/gdpr"

/**
 * Public landing page reached from the email link
 * /account/confirm-deletion?token=<jwt>.
 *
 * On mount we POST the token to the backend and render one of three
 * states (loading / success / error). The page is intentionally NOT
 * auth-guarded — the JWT in the URL IS the auth.
 */
export default function ConfirmDeletionPage() {
  return (
    <Suspense fallback={<Skeleton />}>
      <ConfirmInner />
    </Suspense>
  )
}

function ConfirmInner() {
  const t = useTranslations("account.gdpr.confirmPage")
  const searchParams = useSearchParams()
  const [state, setState] = useState<"loading" | "success" | "error">("loading")
  const [hardDeleteAt, setHardDeleteAt] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const token = searchParams.get("token")
    if (!token) {
      setState("error")
      setError(t("errors.missingToken"))
      return
    }
    confirmDeletion(token)
      .then((res) => {
        setHardDeleteAt(res.hard_delete_at)
        setState("success")
      })
      .catch((err) => {
        setState("error")
        setError(err instanceof Error ? err.message : t("errors.generic"))
      })
  }, [searchParams, t])

  if (state === "loading") {
    return <Skeleton />
  }

  return (
    <main className="mx-auto max-w-lg px-6 py-16">
      <div className="rounded-2xl border border-slate-200 bg-white p-8 shadow-sm dark:border-slate-700 dark:bg-slate-800">
        {state === "success" ? (
          <>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
              {t("success.title")}
            </h1>
            <p className="mt-3 text-sm text-slate-600 dark:text-slate-400">
              {t("success.body", {
                date: hardDeleteAt ? new Date(hardDeleteAt).toLocaleDateString() : "—",
              })}
            </p>
            <p className="mt-4 text-sm text-slate-600 dark:text-slate-400">
              {t("success.cancelHint")}
            </p>
          </>
        ) : (
          <>
            <h1 className="text-2xl font-bold text-red-700 dark:text-red-400">
              {t("error.title")}
            </h1>
            <p className="mt-3 text-sm text-slate-600 dark:text-slate-400">
              {error ?? t("errors.generic")}
            </p>
          </>
        )}
      </div>
    </main>
  )
}

function Skeleton() {
  return (
    <main className="mx-auto max-w-lg px-6 py-16">
      <div className="h-32 animate-pulse rounded-2xl bg-slate-200 dark:bg-slate-700" />
    </main>
  )
}
