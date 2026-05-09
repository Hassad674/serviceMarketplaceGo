"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useState } from "react"
import { Link } from "@i18n/navigation"
import { Eye, EyeOff, CheckCircle2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { resetPassword } from "@/features/auth/api/auth-api"
import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"

// Schema kept inline (extracting to features/auth/schemas/ is an
// OFF-LIMITS refactor for this UI batch). Validation rules unchanged
// — every literal stays english so existing zod messages keep
// matching the e2e contract.
const resetPasswordSchema = z
  .object({
    password: z
      .string()
      .min(10, "Password must contain at least 10 characters")
      .regex(/[A-Z]/, "Password must contain at least one uppercase letter")
      .regex(/[a-z]/, "Password must contain at least one lowercase letter")
      .regex(/[0-9]/, "Password must contain at least one digit")
      .regex(
        /[^A-Za-z0-9]/,
        "Password must contain at least one special character",
      ),
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
      <div className="space-y-4 text-center" role="alert">
        <p className="text-sm leading-relaxed text-muted-foreground">
          {t("invalidLinkDesc")}
        </p>
        <Link
          href="/forgot-password"
          className="inline-block text-[13px] font-semibold text-primary transition-colors hover:text-primary-deep"
        >
          {tCommon("requestNewLink")}
          <span aria-hidden="true"> →</span>
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
      setError(err instanceof Error ? err.message : tCommon("errorOccurred"))
    }
  }

  if (success) {
    return (
      <div
        className="space-y-4 text-center"
        role="status"
        aria-live="polite"
      >
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-success-soft">
          <CheckCircle2 className="h-7 w-7 text-success" aria-hidden="true" />
        </div>
        <h2 className="font-serif text-[22px] font-medium text-foreground">
          {t("resetSuccess")}
        </h2>
        <p className="text-sm leading-relaxed text-muted-foreground">
          {tCommon("canSignIn")}
        </p>
        <Link
          href="/login"
          className={[
            "mt-2 inline-block rounded-full bg-primary-deep px-7 py-3 text-[14.5px] font-semibold text-white",
            "transition-colors hover:bg-primary",
            "focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/30",
          ].join(" ")}
          style={{ boxShadow: "0 4px 14px rgba(232, 93, 74, 0.3)" }}
        >
          {tCommon("signIn")}
        </Link>
      </div>
    )
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      {error && (
        <div
          className="rounded-xl border border-destructive/30 bg-primary-soft/40 p-3 text-sm text-destructive"
          role="alert"
        >
          {error}
        </div>
      )}

      {/* New password */}
      <div className="space-y-1.5">
        <label
          htmlFor="password"
          className="block text-[13px] font-semibold text-foreground"
        >
          {t("newPassword")}
        </label>
        <div className="relative">
          <Input
            id="password"
            type={showPassword ? "text" : "password"}
            autoComplete="new-password"
            placeholder={t("newPasswordPlaceholder")}
            aria-invalid={errors.password ? true : undefined}
            aria-describedby={
              errors.password ? "password-error" : "password-hint"
            }
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
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={() => setShowPassword(!showPassword)}
            className="absolute right-3 top-1/2 -translate-y-1/2 rounded-md p-1 text-muted-foreground transition-colors hover:text-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
            aria-label={
              showPassword
                ? tCommon("hidePassword")
                : tCommon("showPassword")
            }
          >
            {showPassword ? (
              <EyeOff className="h-4 w-4" />
            ) : (
              <Eye className="h-4 w-4" />
            )}
          </Button>
        </div>
        {errors.password?.message ? (
          <p id="password-error" className="text-xs text-destructive">
            {errors.password.message}
          </p>
        ) : (
          <p id="password-hint" className="text-xs text-muted-foreground">
            {tCommon("passwordHintFull")}
          </p>
        )}
      </div>

      {/* Confirm password */}
      <div className="space-y-1.5">
        <label
          htmlFor="confirmPassword"
          className="block text-[13px] font-semibold text-foreground"
        >
          {t("confirmPassword")}
        </label>
        <div className="relative">
          <Input
            id="confirmPassword"
            type={showConfirm ? "text" : "password"}
            autoComplete="new-password"
            placeholder={t("confirmPasswordPlaceholder")}
            aria-invalid={errors.confirmPassword ? true : undefined}
            aria-describedby={
              errors.confirmPassword ? "confirm-error" : undefined
            }
            className={[
              "block w-full rounded-xl border bg-card px-4 py-[13px] pr-11 text-[14.5px] text-foreground",
              "transition-colors duration-150 placeholder:text-subtle-foreground",
              "focus:border-primary focus:outline-none focus:ring-4 focus:ring-primary/15",
              errors.confirmPassword
                ? "border-destructive focus:ring-destructive/15"
                : "border-border-strong",
            ].join(" ")}
            {...registerField("confirmPassword")}
          />
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={() => setShowConfirm(!showConfirm)}
            className="absolute right-3 top-1/2 -translate-y-1/2 rounded-md p-1 text-muted-foreground transition-colors hover:text-foreground focus:outline-none focus:ring-2 focus:ring-primary/30"
            aria-label={
              showConfirm
                ? tCommon("hidePassword")
                : tCommon("showPassword")
            }
          >
            {showConfirm ? (
              <EyeOff className="h-4 w-4" />
            ) : (
              <Eye className="h-4 w-4" />
            )}
          </Button>
        </div>
        {errors.confirmPassword?.message && (
          <p id="confirm-error" className="text-xs text-destructive">
            {errors.confirmPassword.message}
          </p>
        )}
      </div>

      <Button
        variant="primary"
        size="auto"
        type="submit"
        disabled={isSubmitting}
        className={[
          "mt-2 w-full rounded-full px-4 py-3.5 text-[14.5px] font-semibold",
          "active:scale-[0.99]",
          "focus:outline-none focus:ring-4 focus:ring-primary/30",
          "disabled:cursor-not-allowed disabled:opacity-60",
        ].join(" ")}
        style={{ boxShadow: "0 4px 14px rgba(232, 93, 74, 0.3)" }}
      >
        {isSubmitting ? t("resetting") : t("resetPassword")}
      </Button>
    </form>
  )
}
