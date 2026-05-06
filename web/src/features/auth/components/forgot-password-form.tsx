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

import { Input } from "@/shared/components/ui/input"
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
      <div className="animate-scale-in rounded-2xl border border-border bg-card p-8 shadow-[var(--shadow-card)] space-y-4 text-center">
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-success-soft">
          <CheckCircle2 className="h-7 w-7 text-success" />
        </div>
        <h2 className="text-lg font-bold text-foreground">{tCommon("emailSent")}</h2>
        <p className="text-sm text-muted-foreground">
          {t("resetEmailSent")}
        </p>
        <Link
          href="/login"
          className="inline-block text-sm font-medium text-[var(--text-link)] hover:text-primary"
        >
          {t("backToLogin")}
        </Link>
      </div>
    )
  }

  return (
    <div className="animate-scale-in rounded-2xl border border-border bg-card p-8 shadow-[var(--shadow-card)]">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
        {error && (
          <div className="rounded-xl border border-red-200 dark:border-red-500/20 bg-red-50 dark:bg-red-500/10 p-3 text-sm text-red-600 dark:text-red-400" role="alert">
            {error}
          </div>
        )}

        <div className="space-y-1.5">
          <label htmlFor="email" className="block text-sm font-medium text-foreground">
            {t("email")}
          </label>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            placeholder={t("emailPlaceholder")}
            className="h-12 rounded-xl px-4"
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

      <p className="mt-6 text-center text-sm text-muted-foreground">
        <Link
          href="/login"
          className="font-medium text-[var(--text-link)] hover:text-primary"
        >
          {t("backToLogin")}
        </Link>
      </p>
    </div>
  )
}
