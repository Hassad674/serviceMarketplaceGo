"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cancelDeletion } from "@/features/account/api/gdpr"
import { Button } from "@/shared/components/ui/button"

/**
 * Public landing page for the cancel link sent in the T+25j reminder
 * email. The user lands here logged-out (the email may be opened on
 * a different device) and clicks the button. The button POSTs to the
 * backend and shows a confirmation. Auth is required by the API; if
 * the user is not logged in they will get a 401 and we redirect them
 * to /login with a return path.
 */
export default function CancelDeletionPage() {
  const t = useTranslations("account.gdpr.cancelPage")
  const router = useRouter()
  const [submitting, setSubmitting] = useState(false)
  const [done, setDone] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleCancel() {
    setSubmitting(true)
    setError(null)
    try {
      await cancelDeletion()
      setDone(true)
    } catch (err) {
      const isAuthError =
        err instanceof Error &&
        (err.message.toLowerCase().includes("not authenticated") ||
          err.message.toLowerCase().includes("unauthor"))
      if (isAuthError) {
        router.replace(`/login?next=${encodeURIComponent("/account/cancel-deletion")}`)
        return
      }
      setError(err instanceof Error ? err.message : t("errors.generic"))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main className="mx-auto max-w-lg px-6 py-16">
      <div className="rounded-2xl border border-slate-200 bg-white p-8 shadow-sm dark:border-slate-700 dark:bg-slate-800">
        {done ? (
          <>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
              {t("done.title")}
            </h1>
            <p className="mt-3 text-sm text-slate-600 dark:text-slate-400">
              {t("done.body")}
            </p>
          </>
        ) : (
          <>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
              {t("title")}
            </h1>
            <p className="mt-3 text-sm text-slate-600 dark:text-slate-400">
              {t("body")}
            </p>
            {error && (
              <p className="mt-3 text-sm text-red-600" role="alert">
                {error}
              </p>
            )}
            <Button
              type="button"
              variant="primary"
              className="mt-4"
              onClick={handleCancel}
              disabled={submitting}
            >
              {submitting ? t("submitting") : t("button")}
            </Button>
          </>
        )}
      </div>
    </main>
  )
}
