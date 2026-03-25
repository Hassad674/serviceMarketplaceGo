"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useState } from "react"
import { Eye, EyeOff } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link, useRouter } from "@i18n/navigation"
import { login } from "@/features/auth/api/auth-api"

const loginSchema = z.object({
  email: z.string().email("Invalid email address"),
  password: z.string().min(8, "Password must contain at least 8 characters"),
})

type LoginValues = z.infer<typeof loginSchema>

export function LoginForm() {
  const router = useRouter()
  const [error, setError] = useState<string | null>(null)
  const [showPassword, setShowPassword] = useState(false)
  const t = useTranslations("auth")
  const tCommon = useTranslations("common")

  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
  })

  async function onSubmit(values: LoginValues) {
    setError(null)
    try {
      await login(values.email, values.password)
      router.push("/dashboard")
    } catch (err) {
      setError(
        err instanceof Error ? err.message : tCommon("errorOccurred"),
      )
    }
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

        <div className="space-y-1.5">
          <label htmlFor="password" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("password")}
          </label>
          <div className="relative">
            <input
              id="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              placeholder={t("passwordPlaceholder")}
              className="h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 pr-11 text-sm text-gray-900 dark:text-white transition-all placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
              {...registerField("password")}
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 transition-colors hover:text-gray-600 dark:hover:text-gray-300"
              aria-label={showPassword ? tCommon("hidePassword") : tCommon("showPassword")}
            >
              {showPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
            </button>
          </div>
          {errors.password && (
            <p className="text-sm text-red-500 dark:text-red-400 mt-1">{errors.password.message}</p>
          )}
          <div className="flex justify-end">
            <Link
              href="/forgot-password"
              className="text-sm font-medium text-rose-500 hover:text-rose-600 dark:text-rose-400 dark:hover:text-rose-300"
            >
              {t("forgotPassword")}
            </Link>
          </div>
        </div>

        <button
          type="submit"
          disabled={isSubmitting}
          className="gradient-primary h-12 w-full rounded-xl font-semibold text-white shadow-md transition-all hover:shadow-glow active:scale-[0.98] disabled:opacity-50"
        >
          {isSubmitting ? t("signingIn") : t("loginTitle")}
        </button>
      </form>

      <p className="mt-6 text-center text-sm text-gray-500 dark:text-gray-400">
        {t("noAccount")}{" "}
        <Link
          href="/register"
          className="font-medium text-rose-500 hover:text-rose-600 dark:text-rose-400 dark:hover:text-rose-300"
        >
          {tCommon("createAccount")}
        </Link>
      </p>
    </div>
  )
}
