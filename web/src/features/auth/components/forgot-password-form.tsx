"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useState } from "react"
import { Link } from "@i18n/navigation"
import { CheckCircle2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { forgotPassword } from "@/features/auth/api/auth-api"
import { Button } from "@/shared/components/ui/button"

const forgotPasswordSchema = z.object({
  email: z.string().email("Invalid email address"),
})

type ForgotPasswordValues = z.infer<typeof forgotPasswordSchema>

export function ForgotPasswordForm() {
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const t = useTranslations("auth")
  const tCommon = useTranslations("common")

  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ForgotPasswordValues>({
    resolver: zodResolver(forgotPasswordSchema),
  })

  async function onSubmit(values: ForgotPasswordValues) {
    setError(null)
    try {
      await forgotPassword(values.email)
      setSuccess(true)
    } catch (err) {
      setError(
        err instanceof Error ? err.message : tCommon("errorOccurred"),
      )
    }
  }

  if (success) {
    return (
      <div className="animate-scale-in rounded-2xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-8 shadow-lg space-y-4 text-center">
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-500/20">
          <CheckCircle2 className="h-7 w-7 text-emerald-600 dark:text-emerald-400" />
        </div>
        <h2 className="text-lg font-bold text-gray-900 dark:text-white">{tCommon("emailSent")}</h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("resetEmailSent")}
        </p>
        <Link
          href="/login"
          className="inline-block text-sm font-medium text-rose-500 hover:text-rose-600"
        >
          {t("backToLogin")}
        </Link>
      </div>
    )
  }

  return (
    <div className="animate-scale-in rounded-2xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-8 shadow-lg">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
        {error && (
          <div className="rounded-xl border border-red-200 dark:border-red-500/20 bg-red-50 dark:bg-red-500/10 p-3 text-sm text-red-600 dark:text-red-400" role="alert">
            {error}
          </div>
        )}

        <div className="space-y-1.5">
          <label htmlFor="email" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("email")}
          </label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            placeholder={t("emailPlaceholder")}
            className="h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 text-sm text-gray-900 dark:text-white transition-all placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
            {...registerField("email")}
          />
          {errors.email && (
            <p className="text-sm text-red-500 dark:text-red-400 mt-1">{errors.email.message}</p>
          )}
        </div>

        <Button variant="primary" size="auto"
          type="submit"
          disabled={isSubmitting}
          className="h-12 w-full rounded-xl font-semibold text-white shadow-md transition-all disabled:opacity-50"
        >
          {isSubmitting ? t("sending") : t("sendResetLink")}
        </Button>
      </form>

      <p className="mt-6 text-center text-sm text-gray-500 dark:text-gray-400">
        <Link
          href="/login"
          className="font-medium text-rose-500 hover:text-rose-600"
        >
          {t("backToLogin")}
        </Link>
      </p>
    </div>
  )
}
