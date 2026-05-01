"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useState } from "react"
import { Link, useRouter } from "@i18n/navigation"
import { ArrowLeft, Eye, EyeOff } from "lucide-react"
import { useTranslations } from "next-intl"
import { register as registerUser } from "@/features/auth/api/auth-api"
import { Button } from "@/shared/components/ui/button"

const providerSchema = z
  .object({
    first_name: z.string().min(1, "First name is required"),
    last_name: z.string().min(1, "Last name is required"),
    email: z.string().email("Invalid email address"),
    password: z
      .string()
      .min(10, "Minimum 10 characters")
      .regex(/[A-Z]/, "At least one uppercase letter")
      .regex(/[a-z]/, "At least one lowercase letter")
      .regex(/[0-9]/, "At least one digit")
      .regex(/[^A-Za-z0-9]/, "At least one special character"),
    confirm_password: z.string(),
  })
  .refine((data) => data.password === data.confirm_password, {
    message: "Passwords do not match",
    path: ["confirm_password"],
  })

type ProviderValues = z.infer<typeof providerSchema>

export function ProviderRegisterForm() {
  const router = useRouter()
  const [error, setError] = useState<string | null>(null)
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)
  const t = useTranslations("auth")
  const tCommon = useTranslations("common")

  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ProviderValues>({
    resolver: zodResolver(providerSchema),
  })

  async function onSubmit(values: ProviderValues) {
    setError(null)
    try {
      await registerUser({
        email: values.email,
        password: values.password,
        first_name: values.first_name,
        last_name: values.last_name,
        display_name: "",
        role: "provider",
      })
      router.push("/dashboard")
    } catch (err) {
      setError(err instanceof Error ? err.message : tCommon("errorOccurred"))
    }
  }

  return (
    <div className="animate-scale-in rounded-2xl border border-gray-100 dark:border-gray-800 bg-white dark:bg-gray-900 p-8 shadow-lg">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
        {error && (
          <div role="alert" className="rounded-xl border border-red-200 dark:border-red-500/20 bg-red-50 dark:bg-red-500/10 p-3 text-sm text-red-600 dark:text-red-400">
            {error}
          </div>
        )}

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <label htmlFor="first_name" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {t("firstName")}
            </label>
            <input
              id="first_name"
              type="text"
              autoComplete="given-name"
              placeholder={t("firstNamePlaceholder")}
              className="h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 text-sm text-gray-900 dark:text-white transition-all placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
              {...registerField("first_name")}
            />
            {errors.first_name && (
              <p className="text-sm text-red-500 dark:text-red-400 mt-1">{errors.first_name.message}</p>
            )}
          </div>

          <div className="space-y-1.5">
            <label htmlFor="last_name" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {t("lastName")}
            </label>
            <input
              id="last_name"
              type="text"
              autoComplete="family-name"
              placeholder={t("lastNamePlaceholder")}
              className="h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 text-sm text-gray-900 dark:text-white transition-all placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
              {...registerField("last_name")}
            />
            {errors.last_name && (
              <p className="text-sm text-red-500 dark:text-red-400 mt-1">{errors.last_name.message}</p>
            )}
          </div>
        </div>

        <div className="space-y-1.5">
          <label htmlFor="email" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("email")}
          </label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            placeholder={t("providerEmailPlaceholder")}
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
              autoComplete="new-password"
              placeholder={t("passwordPlaceholder")}
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
          <p className="text-xs text-gray-400 dark:text-gray-500">
            {t("passwordHint")}
          </p>
          {errors.password && (
            <p className="text-sm text-red-500 dark:text-red-400 mt-1">{errors.password.message}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="confirm_password" className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {t("confirmPassword")}
          </label>
          <div className="relative">
            <input
              id="confirm_password"
              type={showConfirm ? "text" : "password"}
              autoComplete="new-password"
              placeholder={t("confirmPasswordPlaceholder")}
              className="h-12 w-full rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-4 pr-11 text-sm text-gray-900 dark:text-white transition-all placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
              {...registerField("confirm_password")}
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
          {errors.confirm_password && (
            <p className="text-sm text-red-500 dark:text-red-400 mt-1">{errors.confirm_password.message}</p>
          )}
        </div>

        <Button variant="primary" size="auto"
          type="submit"
          disabled={isSubmitting}
          className="h-12 w-full rounded-xl font-semibold text-white shadow-md transition-all disabled:opacity-50"
        >
          {isSubmitting ? t("signingUp") : t("createFreelanceAccount")}
        </Button>
      </form>

      <div className="mt-6 flex flex-col items-center gap-3 text-sm">
        <Link
          href="/register"
          className="inline-flex items-center gap-1.5 font-medium text-gray-600 dark:text-gray-400 transition-colors hover:text-gray-900 dark:hover:text-white"
        >
          <ArrowLeft className="h-4 w-4" />
          {t("changeProfile")}
        </Link>
        <p className="text-gray-500 dark:text-gray-400">
          {t("alreadyRegistered")}{" "}
          <Link href="/login" className="font-medium text-rose-500 hover:text-rose-600 dark:text-rose-400 dark:hover:text-rose-300">
            {tCommon("signIn")}
          </Link>
        </p>
      </div>
    </div>
  )
}
