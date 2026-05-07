"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useTranslations } from "next-intl"
import { toast } from "sonner"

import { useUser } from "@/shared/hooks/use-user"
import { Input } from "@/shared/components/ui/input"
import { Button } from "@/shared/components/ui/button"
import { ApiError } from "@/shared/lib/api-client"
import { useChangeEmail } from "../hooks/use-change-email"

/**
 * EmailSettings — Soleil v2 form to update the account email.
 *
 * After a successful change the backend bumps the session version,
 * so we hard-redirect to /login (with a "reconnecte-toi" toast) to
 * destroy any in-memory cache. This mirrors `useLogout()` and avoids
 * the surprising "everything starts 401-ing" failure mode that a
 * silent in-place refresh would create.
 */

function buildSchema(messages: { invalidEmail: string; passwordRequired: string }) {
  return z.object({
    current_password: z.string().min(1, messages.passwordRequired),
    new_email: z.string().email(messages.invalidEmail),
  })
}

type EmailSchemaValues = {
  current_password: string
  new_email: string
}

const ERROR_CODE_KEYS: Record<string, string> = {
  invalid_email: "errors.invalidEmail",
  same_email: "errors.sameEmail",
  email_already_exists: "errors.emailAlreadyExists",
  invalid_credentials: "errors.invalidCredentials",
  session_invalid: "errors.sessionInvalid",
  unauthorized: "errors.sessionInvalid",
}

export function EmailSettings() {
  const t = useTranslations("account")
  const { data: user } = useUser()
  const mutation = useChangeEmail()

  const schema = buildSchema({
    invalidEmail: t("errors.invalidEmail"),
    passwordRequired: t("errors.passwordRequired"),
  })

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<EmailSchemaValues>({
    resolver: zodResolver(schema),
    defaultValues: { current_password: "", new_email: "" },
  })

  const isPending = mutation.isPending || isSubmitting

  function onSubmit(values: EmailSchemaValues) {
    mutation.mutate(values, {
      onSuccess: () => {
        toast.success(t("emailChangedSuccess"))
        reset({ current_password: "", new_email: "" })
        // Backend bumped the session version — force a clean re-login.
        // Hard redirect destroys the in-memory React tree + query cache,
        // matching the shared `useLogout()` behaviour.
        window.location.href = "/login"
      },
      onError: (err) => {
        // Reset the password field whether the error is recoverable
        // (wrong password) or not — never let a credential linger in
        // a controlled input.
        reset(
          { current_password: "", new_email: values.new_email },
          { keepErrors: false },
        )
        if (err instanceof ApiError) {
          const key = ERROR_CODE_KEYS[err.code] ?? "errors.generic"
          const message = t(key)
          // Map the error to the most relevant field for inline display.
          if (err.code === "invalid_credentials") {
            setError("current_password", { message })
          } else if (
            err.code === "invalid_email" ||
            err.code === "same_email" ||
            err.code === "email_already_exists"
          ) {
            setError("new_email", { message })
          } else {
            toast.error(message)
          }
          return
        }
        toast.error(t("errors.generic"))
      },
    })
  }

  return (
    <div className="space-y-6">
      <header>
        <h2 className="font-serif text-[22px] font-semibold tracking-[-0.01em] text-foreground">
          {t("emailTitle")}
        </h2>
        <p className="mt-1.5 text-sm text-muted-foreground">{t("emailDesc")}</p>
      </header>

      <form
        noValidate
        onSubmit={handleSubmit(onSubmit)}
        className="rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)]"
      >
        <div>
          <span className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
            {t("currentEmail")}
          </span>
          <p className="mt-1.5 font-mono text-[15px] text-foreground">
            {user?.email || "—"}
          </p>
        </div>

        <div className="mt-7 space-y-5">
          <div>
            <label
              htmlFor="account-current-password-email"
              className="block text-sm font-medium text-foreground"
            >
              {t("currentPassword")}
            </label>
            <Input
              id="account-current-password-email"
              type="password"
              autoComplete="current-password"
              disabled={isPending}
              aria-invalid={Boolean(errors.current_password) || undefined}
              aria-describedby={
                errors.current_password
                  ? "account-current-password-email-error"
                  : undefined
              }
              className="mt-1.5"
              {...register("current_password")}
            />
            {errors.current_password && (
              <p
                id="account-current-password-email-error"
                role="alert"
                className="mt-1 text-xs text-red-600 dark:text-red-400"
              >
                {errors.current_password.message}
              </p>
            )}
          </div>

          <div>
            <label
              htmlFor="account-new-email"
              className="block text-sm font-medium text-foreground"
            >
              {t("newEmail")}
            </label>
            <Input
              id="account-new-email"
              type="email"
              autoComplete="email"
              disabled={isPending}
              placeholder="new@email.com"
              aria-invalid={Boolean(errors.new_email) || undefined}
              aria-describedby={
                errors.new_email ? "account-new-email-error" : undefined
              }
              className="mt-1.5"
              {...register("new_email")}
            />
            {errors.new_email && (
              <p
                id="account-new-email-error"
                role="alert"
                className="mt-1 text-xs text-red-600 dark:text-red-400"
              >
                {errors.new_email.message}
              </p>
            )}
          </div>
        </div>

        <div className="mt-6 flex justify-end">
          <Button
            type="submit"
            variant="primary"
            size="md"
            disabled={isPending}
          >
            {isPending ? t("saving") : t("changeEmailCta")}
          </Button>
        </div>
      </form>
    </div>
  )
}
