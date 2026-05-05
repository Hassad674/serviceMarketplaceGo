"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useState } from "react"
import { useRouter } from "@i18n/navigation"
import { Eye, EyeOff } from "lucide-react"
import { useTranslations } from "next-intl"
import { register as registerUser } from "@/features/auth/api/auth-api"

// Step-2 enterprise form — visual port to Soleil v2.
//
// IMPORTANT (locked by the design batch contract):
//   - The list of fields collected (display_name, email, password,
//     confirm_password) is unchanged.
//   - The zod schema and validation messages are unchanged.
//   - The submit handler and redirect target are unchanged.
//
// Only the visual chrome moves — same primitive as login (raw `<input>`,
// rounded-xl, focus ring on primary, corail submit pill).
const enterpriseSchema = z
  .object({
    display_name: z.string().min(2, "Company name is required"),
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

type EnterpriseValues = z.infer<typeof enterpriseSchema>

const inputBaseClasses = [
  "block w-full rounded-xl border bg-card px-4 py-[13px] text-[14.5px] text-foreground",
  "transition-colors duration-150 placeholder:text-subtle-foreground",
  "focus:border-primary focus:outline-none focus:ring-4 focus:ring-primary/15",
].join(" ")

function inputStateClasses(hasError: boolean, padRight?: boolean): string {
  return [
    inputBaseClasses,
    padRight ? "pr-11" : "",
    hasError
      ? "border-destructive focus:ring-destructive/15"
      : "border-border-strong",
  ]
    .filter(Boolean)
    .join(" ")
}

export function EnterpriseRegisterForm() {
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
  } = useForm<EnterpriseValues>({
    resolver: zodResolver(enterpriseSchema),
  })

  async function onSubmit(values: EnterpriseValues) {
    setError(null)
    try {
      await registerUser({
        email: values.email,
        password: values.password,
        first_name: "",
        last_name: "",
        display_name: values.display_name,
        role: "enterprise",
      })
      router.push("/dashboard")
    } catch (err) {
      setError(err instanceof Error ? err.message : tCommon("errorOccurred"))
    }
  }

  return (
    <div className="rounded-2xl border border-border bg-card p-7 shadow-sm sm:p-8">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        {error && (
          <div
            role="alert"
            className="rounded-xl border border-destructive/30 bg-primary-soft/40 p-3 text-sm text-destructive"
          >
            {error}
          </div>
        )}

        <div className="space-y-1.5">
          <label
            htmlFor="display_name"
            className="block text-[13px] font-semibold text-foreground"
          >
            {t("companyName")}
          </label>
          <input
            id="display_name"
            type="text"
            autoComplete="organization"
            placeholder={t("companyNamePlaceholder")}
            aria-invalid={errors.display_name ? true : undefined}
            aria-describedby={
              errors.display_name ? "display_name-error" : "display_name-hint"
            }
            className={inputStateClasses(Boolean(errors.display_name))}
            {...registerField("display_name")}
          />
          {errors.display_name?.message ? (
            <p id="display_name-error" className="text-xs text-destructive">
              {errors.display_name.message}
            </p>
          ) : (
            <p id="display_name-hint" className="text-xs text-subtle-foreground">
              {t("companyNameHint")}
            </p>
          )}
        </div>

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
            placeholder={t("enterpriseEmailPlaceholder")}
            aria-invalid={errors.email ? true : undefined}
            aria-describedby={errors.email ? "email-error" : undefined}
            className={inputStateClasses(Boolean(errors.email))}
            {...registerField("email")}
          />
          {errors.email?.message && (
            <p id="email-error" className="text-xs text-destructive">
              {errors.email.message}
            </p>
          )}
        </div>

        <div className="space-y-1.5">
          <label
            htmlFor="password"
            className="block text-[13px] font-semibold text-foreground"
          >
            {t("password")}
          </label>
          <div className="relative">
            <input
              id="password"
              type={showPassword ? "text" : "password"}
              autoComplete="new-password"
              placeholder={t("passwordPlaceholder")}
              aria-invalid={errors.password ? true : undefined}
              aria-describedby={
                errors.password ? "password-error" : "password-hint"
              }
              className={inputStateClasses(Boolean(errors.password), true)}
              {...registerField("password")}
            />
            <button
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
            </button>
          </div>
          {errors.password?.message ? (
            <p id="password-error" className="text-xs text-destructive">
              {errors.password.message}
            </p>
          ) : (
            <p id="password-hint" className="text-xs text-subtle-foreground">
              {t("passwordHint")}
            </p>
          )}
        </div>

        <div className="space-y-1.5">
          <label
            htmlFor="confirm_password"
            className="block text-[13px] font-semibold text-foreground"
          >
            {t("confirmPassword")}
          </label>
          <div className="relative">
            <input
              id="confirm_password"
              type={showConfirm ? "text" : "password"}
              autoComplete="new-password"
              placeholder={t("confirmPasswordPlaceholder")}
              aria-invalid={errors.confirm_password ? true : undefined}
              aria-describedby={
                errors.confirm_password ? "confirm_password-error" : undefined
              }
              className={inputStateClasses(
                Boolean(errors.confirm_password),
                true,
              )}
              {...registerField("confirm_password")}
            />
            <button
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
            </button>
          </div>
          {errors.confirm_password?.message && (
            <p id="confirm_password-error" className="text-xs text-destructive">
              {errors.confirm_password.message}
            </p>
          )}
        </div>

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
          {isSubmitting ? t("signingUp") : t("createEnterpriseAccount")}
        </button>
      </form>
    </div>
  )
}
