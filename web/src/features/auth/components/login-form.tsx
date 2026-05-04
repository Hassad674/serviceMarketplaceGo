"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useState } from "react"
import { Eye, EyeOff, ShieldAlert } from "lucide-react"
import { useTranslations } from "next-intl"
import { Link, useRouter } from "@i18n/navigation"
import { login, AuthApiError } from "@/features/auth/api/auth-api"

// Schema kept inline (intentionally not extracted to features/auth/schemas/
// for this batch — that is OFF-LIMITS work that will happen later as a
// dedicated refactor). Validation rules unchanged.
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
      if (
        err instanceof AuthApiError &&
        (err.code === "account_suspended" || err.code === "account_banned")
      ) {
        setSuspension({ message: err.message, reason: err.reason || "" })
        return
      }
      setError(err instanceof Error ? err.message : tCommon("errorOccurred"))
    }
  }

  return (
    <>
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        {suspension && (
          <div
            className="space-y-2 rounded-xl border border-warning/30 bg-amber-soft/50 p-4 text-sm"
            role="alert"
          >
            <div className="flex items-center gap-2 font-semibold text-warning">
              <ShieldAlert className="h-5 w-5 flex-shrink-0" />
              <span>{suspension.message}</span>
            </div>
            {suspension.reason && (
              <p className="pl-7 text-xs text-muted-foreground">
                {suspension.reason}
              </p>
            )}
          </div>
        )}

        {error && (
          <div
            className="rounded-xl border border-destructive/30 bg-primary-soft/40 p-3 text-sm text-destructive"
            role="alert"
          >
            {error}
          </div>
        )}

        {/* Email */}
        <div className="space-y-1.5">
          <label
            htmlFor="email"
            className="block text-[13px] font-semibold text-foreground"
          >
            {t("email")}
          </label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            placeholder={t("emailPlaceholder")}
            aria-invalid={errors.email ? true : undefined}
            aria-describedby={errors.email ? "email-error" : undefined}
            className={[
              "block w-full rounded-xl border bg-card px-4 py-[13px] text-[14.5px] text-foreground",
              "transition-colors duration-150 placeholder:text-subtle-foreground",
              "focus:border-primary focus:outline-none focus:ring-4 focus:ring-primary/15",
              errors.email
                ? "border-destructive focus:ring-destructive/15"
                : "border-border-strong",
            ].join(" ")}
            {...registerField("email")}
          />
          {errors.email?.message && (
            <p id="email-error" className="text-xs text-destructive">
              {errors.email.message}
            </p>
          )}
        </div>

        {/* Password + forgot link */}
        <div className="space-y-1.5">
          <div className="flex items-baseline justify-between">
            <label
              htmlFor="password"
              className="block text-[13px] font-semibold text-foreground"
            >
              {t("password")}
            </label>
            <Link
              href="/forgot-password"
              className="font-serif text-xs italic text-primary transition-colors hover:text-primary-deep"
            >
              {t("forgotPassword")}
            </Link>
          </div>
          <div className="relative">
            <input
              id="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              placeholder={t("passwordPlaceholder")}
              aria-invalid={errors.password ? true : undefined}
              aria-describedby={errors.password ? "password-error" : undefined}
              className={[
                "block w-full rounded-xl border bg-card px-4 py-[13px] pr-11 text-[14.5px] text-foreground",
                "transition-colors duration-150 placeholder:text-subtle-foreground",
                "focus:border-primary focus:outline-none focus:ring-4 focus:ring-primary/15",
                errors.password
                  ? "border-destructive focus:ring-destructive/15"
                  : "border-border-strong",
              ].join(" ")}
              {...registerField("password")}
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 rounded-md p-1 text-muted-foreground transition-colors hover:text-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
              aria-label={
                showPassword ? tCommon("hidePassword") : tCommon("showPassword")
              }
            >
              {showPassword ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </button>
          </div>
          {errors.password?.message && (
            <p id="password-error" className="text-xs text-destructive">
              {errors.password.message}
            </p>
          )}
        </div>

        {/* Submit */}
        <button
          type="submit"
          disabled={isSubmitting}
          className={[
            "mt-2 w-full rounded-full bg-primary px-4 py-3.5 text-[14.5px] font-semibold text-primary-foreground",
            "transition-all duration-150 hover:bg-primary-deep active:scale-[0.99]",
            "focus:outline-none focus:ring-4 focus:ring-primary/30",
            "disabled:cursor-not-allowed disabled:opacity-60",
          ].join(" ")}
          style={{ boxShadow: "0 4px 14px rgba(232, 93, 74, 0.3)" }}
        >
          {isSubmitting ? t("signingIn") : t("loginTitle")}
        </button>
      </form>

      {/* No account? */}
      <p className="mt-7 text-center text-[13px] text-muted-foreground">
        {t("noAccount")}{" "}
        <Link
          href="/register"
          className="font-semibold text-primary transition-colors hover:text-primary-deep"
        >
          {tCommon("createAccount")}
          <span aria-hidden="true"> →</span>
        </Link>
      </p>
    </>
  )
}
