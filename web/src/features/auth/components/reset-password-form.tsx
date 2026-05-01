"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useState } from "react"
import { Link } from "@i18n/navigation"
import { Eye, EyeOff, CheckCircle2, XCircle } from "lucide-react"
import { useTranslations } from "next-intl"
import { resetPassword } from "@/features/auth/api/auth-api"
import { Button } from "@/shared/components/ui/button"

const resetPasswordSchema = z
  .object({
    password: z
      .string()
      .min(10, "Password must contain at least 10 characters")
      .regex(/[A-Z]/, "Password must contain at least one uppercase letter")
      .regex(/[a-z]/, "Password must contain at least one lowercase letter")
      .regex(/[0-9]/, "Password must contain at least one digit")
      .regex(/[^A-Za-z0-9]/, "Password must contain at least one special character"),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  })

type ResetPasswordValues = z.infer<typeof resetPasswordSchema>

interface ResetPasswordFormProps {
  token: string
}

export function ResetPasswordForm({ token }: ResetPasswordFormProps) {
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)
  const t = useTranslations("auth")
  const tCommon = useTranslations("common")

  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ResetPasswordValues>({
    resolver: zodResolver(resetPasswordSchema),
  })

  if (!token) {
    return (
      <div className="animate-scale-in rounded-2xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-8 shadow-lg text-center space-y-4">
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-red-100 dark:bg-red-500/20">
          <XCircle className="h-7 w-7 text-red-600 dark:text-red-400" />
        </div>
        <h2 className="text-lg font-bold text-gray-900 dark:text-white">{t("invalidLink")}</h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {t("invalidLinkDesc")}
        </p>
        <Link
          href="/forgot-password"
          className="inline-block text-sm font-medium text-rose-500 hover:text-rose-600"
        >
          {tCommon("requestNewLink")}
        </Link>
      </div>
    )
  }

  async function onSubmit(values: ResetPasswordValues) {
    setError(null)
    try {
      await resetPassword(token, values.password)
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
        <h2 className="text-lg font-bold text-gray-900 dark:text-white">{t("resetSuccess")}</h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          {tCommon("canSignIn")}
        </p>
        <Link
          href="/login"
          className="gradient-primary inline-block rounded-xl px-8 py-3 text-sm font-semibold text-white shadow-md transition-all hover:shadow-glow active:scale-[0.98]"
        >
          {tCommon("signIn")}
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
          <label htmlFor="password" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("newPassword")}
          </label>
          <div className="relative">
            <input
              id="password"
              type={showPassword ? "text" : "password"}
              autoComplete="new-password"
              placeholder={t("newPasswordPlaceholder")}
              className="h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 pr-11 text-sm text-gray-900 dark:text-white transition-all placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
              {...registerField("password")}
            />
            <Button variant="ghost" size="auto"
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 transition-colors hover:text-gray-600 dark:hover:text-gray-300"
              aria-label={showPassword ? tCommon("hidePassword") : tCommon("showPassword")}
            >
              {showPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
            </Button>
          </div>
          {errors.password && (
            <p className="text-sm text-red-500 dark:text-red-400 mt-1">{errors.password.message}</p>
          )}
          <p className="text-xs text-gray-400 dark:text-gray-500">
            {tCommon("passwordHintFull")}
          </p>
        </div>

        <div className="space-y-1.5">
          <label htmlFor="confirmPassword" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("confirmPassword")}
          </label>
          <div className="relative">
            <input
              id="confirmPassword"
              type={showConfirm ? "text" : "password"}
              autoComplete="new-password"
              placeholder={t("confirmPasswordPlaceholder")}
              className="h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 pr-11 text-sm text-gray-900 dark:text-white transition-all placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
              {...registerField("confirmPassword")}
            />
            <Button variant="ghost" size="auto"
              type="button"
              onClick={() => setShowConfirm(!showConfirm)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 transition-colors hover:text-gray-600 dark:hover:text-gray-300"
              aria-label={showConfirm ? tCommon("hidePassword") : tCommon("showPassword")}
            >
              {showConfirm ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
            </Button>
          </div>
          {errors.confirmPassword && (
            <p className="text-sm text-red-500 dark:text-red-400 mt-1">{errors.confirmPassword.message}</p>
          )}
        </div>

        <Button variant="primary" size="auto"
          type="submit"
          disabled={isSubmitting}
          className="h-12 w-full rounded-xl font-semibold text-white shadow-md transition-all disabled:opacity-50"
        >
          {isSubmitting ? t("resetting") : t("resetPassword")}
        </Button>
      </form>
    </div>
  )
}
