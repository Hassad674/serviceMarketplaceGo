"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useTranslations } from "next-intl"
import { toast } from "sonner"

import { Input } from "@/shared/components/ui/input"
import { Button } from "@/shared/components/ui/button"
import { ApiError } from "@/shared/lib/api-client"
import { useChangePassword } from "../hooks/use-change-password"

/**
 * PasswordSettings — Soleil v2 form to rotate the account password.
 *
 * Client-side validation mirrors the backend domain rules:
 *   - ≥ 10 characters
 *   - 1 uppercase, 1 lowercase, 1 digit, 1 special character
 *   - confirmPassword === newPassword
 *
 * After a successful change the backend bumps the session version,
 * so we hard-redirect to /login (with a "reconnecte-toi" toast).
 * Same rationale as EmailSettings — the cleanest way to avoid a
 * stale access token causing silent 401s on the next request.
 */

function buildSchema(messages: {
  passwordRequired: string
  weakPassword: string
  passwordMismatch: string
}) {
  return z
    .object({
      current_password: z.string().min(1, messages.passwordRequired),
      new_password: z
        .string()
        .min(10, messages.weakPassword)
        .regex(/[A-Z]/, messages.weakPassword)
        .regex(/[a-z]/, messages.weakPassword)
        .regex(/[0-9]/, messages.weakPassword)
        .regex(/[^A-Za-z0-9]/, messages.weakPassword),
      confirm_password: z.string(),
    })
    .refine((data) => data.new_password === data.confirm_password, {
      message: messages.passwordMismatch,
      path: ["confirm_password"],
    })
}

type PasswordSchemaValues = {
  current_password: string
  new_password: string
  confirm_password: string
}

const ERROR_CODE_KEYS: Record<string, string> = {
  weak_password: "errors.weakPassword",
  same_password: "errors.samePassword",
  invalid_credentials: "errors.invalidCredentials",
  session_invalid: "errors.sessionInvalid",
  unauthorized: "errors.sessionInvalid",
}

export function PasswordSettings() {
  const t = useTranslations("account")
  const mutation = useChangePassword()

  const schema = buildSchema({
    passwordRequired: t("errors.passwordRequired"),
    weakPassword: t("errors.weakPassword"),
    passwordMismatch: t("errors.passwordMismatch"),
  })

  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<PasswordSchemaValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      current_password: "",
      new_password: "",
      confirm_password: "",
    },
  })

  const isPending = mutation.isPending || isSubmitting

  function clearAllPasswordFields() {
    reset(
      {
        current_password: "",
        new_password: "",
        confirm_password: "",
      },
      { keepErrors: false },
    )
  }

  function onSubmit(values: PasswordSchemaValues) {
    mutation.mutate(
      {
        current_password: values.current_password,
        new_password: values.new_password,
      },
      {
        onSuccess: () => {
          toast.success(t("passwordChangedSuccess"))
          clearAllPasswordFields()
          // Backend bumped session version — clean re-login.
          window.location.href = "/login"
        },
        onError: (err) => {
          // ALWAYS clear the password fields on error too — never let
          // credentials linger in controlled inputs.
          clearAllPasswordFields()
          if (err instanceof ApiError) {
            const key = ERROR_CODE_KEYS[err.code] ?? "errors.generic"
            const message = t(key)
            if (err.code === "invalid_credentials") {
              setError("current_password", { message })
            } else if (
              err.code === "weak_password" ||
              err.code === "same_password"
            ) {
              setError("new_password", { message })
            } else {
              toast.error(message)
            }
            return
          }
          toast.error(t("errors.generic"))
        },
      },
    )
  }

  const fields: Array<{
    id: string
    name: keyof PasswordSchemaValues
    labelKey: string
    autoComplete: string
  }> = [
    {
      id: "account-current-password",
      name: "current_password",
      labelKey: "currentPassword",
      autoComplete: "current-password",
    },
    {
      id: "account-new-password",
      name: "new_password",
      labelKey: "newPassword",
      autoComplete: "new-password",
    },
    {
      id: "account-confirm-password",
      name: "confirm_password",
      labelKey: "confirmPassword",
      autoComplete: "new-password",
    },
  ]

  return (
    <div className="space-y-6">
      <header>
        <h2 className="font-serif text-[22px] font-semibold tracking-[-0.01em] text-foreground">
          {t("passwordTitle")}
        </h2>
        <p className="mt-1.5 text-sm text-muted-foreground">
          {t("passwordDesc")}
        </p>
      </header>

      <form
        noValidate
        onSubmit={handleSubmit(onSubmit)}
        className="rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)]"
      >
        <div className="space-y-5">
          {fields.map((field) => {
            const fieldError = errors[field.name]
            const errorId = `${field.id}-error`
            return (
              <div key={field.id}>
                <label
                  htmlFor={field.id}
                  className="block text-sm font-medium text-foreground"
                >
                  {t(field.labelKey)}
                </label>
                <Input
                  id={field.id}
                  type="password"
                  autoComplete={field.autoComplete}
                  disabled={isPending}
                  placeholder="••••••••"
                  aria-invalid={Boolean(fieldError) || undefined}
                  aria-describedby={fieldError ? errorId : undefined}
                  className="mt-1.5"
                  {...register(field.name)}
                />
                {fieldError && (
                  <p
                    id={errorId}
                    role="alert"
                    className="mt-1 text-xs text-red-600 dark:text-red-400"
                  >
                    {fieldError.message}
                  </p>
                )}
              </div>
            )
          })}
        </div>

        <p className="mt-3 text-xs text-muted-foreground">
          {t("passwordHint")}
        </p>

        <div className="mt-6 flex justify-end">
          <Button
            type="submit"
            variant="primary"
            size="md"
            disabled={isPending}
          >
            {isPending ? t("saving") : t("changePasswordCta")}
          </Button>
        </div>
      </form>
    </div>
  )
}
