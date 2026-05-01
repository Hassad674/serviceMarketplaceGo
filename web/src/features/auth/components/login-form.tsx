"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useState } from "react"
import { Eye, EyeOff, ShieldAlert } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link, useRouter } from "@i18n/navigation"
import { login, AuthApiError } from "@/features/auth/api/auth-api"
import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"

const loginSchema = z.object({
  email: z.string().email("Invalid email address"),
  password: z.string().min(8, "Password must contain at least 8 characters"),
})

type LoginValues = z.infer<typeof loginSchema>

type SuspensionInfo = {
  message: string
  reason: string
}

export function LoginForm() {
  const router = useRouter()
  const [error, setError] = useState<string | null>(null)
  const [suspension, setSuspension] = useState<SuspensionInfo | null>(null)
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
    setSuspension(null)
    try {
      await login(values.email, values.password)
      router.push("/dashboard")
    } catch (err) {
      if (err instanceof AuthApiError && (err.code === "account_suspended" || err.code === "account_banned")) {
        setSuspension({ message: err.message, reason: err.reason || "" })
        return
      }
      setError(
        err instanceof Error ? err.message : tCommon("errorOccurred"),
      )
    }
  }

  return (
    <div className="animate-scale-in rounded-2xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-8 shadow-lg">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
        {suspension && (
          <div className="rounded-xl border border-orange-300 dark:border-orange-500/30 bg-orange-50 dark:bg-orange-500/10 p-4 space-y-2" role="alert">
            <div className="flex items-center gap-2 text-orange-700 dark:text-orange-400 font-semibold text-sm">
              <ShieldAlert className="h-5 w-5 flex-shrink-0" />
              <span>{suspension.message}</span>
            </div>
            {suspension.reason && (
              <p className="text-sm text-orange-600 dark:text-orange-300/80 pl-7">
                {suspension.reason}
              </p>
            )}
          </div>
        )}

        {error && (
          <div className="rounded-xl border border-red-200 dark:border-red-500/20 bg-red-50 dark:bg-red-500/10 p-3 text-sm text-red-600 dark:text-red-400" role="alert">
            {error}
          </div>
        )}

        <Input
          id="email"
          type="email"
          autoComplete="email"
          label={t("email")}
          placeholder={t("emailPlaceholder")}
          size="lg"
          className="h-12 rounded-xl px-4"
          error={errors.email?.message}
          {...registerField("email")}
        />

        <div className="space-y-1.5">
          <label htmlFor="password" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("password")}
          </label>
          <div className="relative">
            <Input
              id="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              placeholder={t("passwordPlaceholder")}
              size="lg"
              className="h-12 rounded-xl px-4 pr-11"
              error={errors.password?.message}
              {...registerField("password")}
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-1 top-1 h-10 w-10 px-0 text-gray-400 hover:bg-transparent hover:text-gray-600 dark:hover:text-gray-300"
              aria-label={showPassword ? tCommon("hidePassword") : tCommon("showPassword")}
            >
              {showPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
            </Button>
          </div>
          <div className="flex justify-end">
            <Link
              href="/forgot-password"
              className="text-sm font-medium text-rose-500 hover:text-rose-600 dark:text-rose-400 dark:hover:text-rose-300"
            >
              {t("forgotPassword")}
            </Link>
          </div>
        </div>

        <Button
          type="submit"
          variant="primary"
          size="lg"
          disabled={isSubmitting}
          className="h-12 w-full rounded-xl text-base font-semibold shadow-md"
        >
          {isSubmitting ? t("signingIn") : t("loginTitle")}
        </Button>
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
