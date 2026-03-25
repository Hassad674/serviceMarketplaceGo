"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useState } from "react"
import Link from "next/link"
import { resetPassword } from "@/features/auth/api/auth-api"

const resetPasswordSchema = z
  .object({
    password: z
      .string()
      .min(8, "Password must contain at least 8 characters")
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

  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ResetPasswordValues>({
    resolver: zodResolver(resetPasswordSchema),
  })

  if (!token) {
    return (
      <div className="rounded-xl border border-border bg-card p-8 shadow-sm text-center space-y-4">
        <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-red-100">
          <svg className="h-6 w-6 text-red-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </div>
        <h2 className="text-lg font-semibold text-foreground">Invalid link</h2>
        <p className="text-sm text-muted-foreground">
          This reset link is invalid or has expired.
        </p>
        <Link
          href="/forgot-password"
          className="inline-block text-sm font-medium text-primary underline underline-offset-4 hover:text-primary/80"
        >
          Request a new link
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
        err instanceof Error ? err.message : "An error occurred",
      )
    }
  }

  if (success) {
    return (
      <div className="rounded-xl border border-border bg-card p-8 shadow-sm space-y-4 text-center">
        <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-green-100">
          <svg className="h-6 w-6 text-green-600" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" d="M4.5 12.75l6 6 9-13.5" />
          </svg>
        </div>
        <h2 className="text-lg font-semibold text-foreground">Password reset successfully!</h2>
        <p className="text-sm text-muted-foreground">
          You can now sign in with your new password.
        </p>
        <Link
          href="/login"
          className="inline-block rounded-md bg-primary px-6 py-2.5 text-sm font-medium text-primary-foreground shadow-sm hover:bg-primary/90"
        >
          Sign In
        </Link>
      </div>
    )
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="rounded-xl border border-border bg-card p-8 shadow-sm space-y-4">
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700" role="alert">
          {error}
        </div>
      )}

      <div className="space-y-2">
        <label htmlFor="password" className="block text-sm font-medium text-foreground">
          New password
        </label>
        <input
          id="password"
          type="password"
          autoComplete="new-password"
          placeholder="Minimum 8 characters"
          className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          {...registerField("password")}
        />
        {errors.password && (
          <p className="text-xs text-red-600">{errors.password.message}</p>
        )}
        <p className="text-xs text-muted-foreground">
          8 characters minimum, one uppercase, one lowercase, one digit and one special character
        </p>
      </div>

      <div className="space-y-2">
        <label htmlFor="confirmPassword" className="block text-sm font-medium text-foreground">
          Confirm password
        </label>
        <input
          id="confirmPassword"
          type="password"
          autoComplete="new-password"
          placeholder="Confirm your password"
          className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          {...registerField("confirmPassword")}
        />
        {errors.confirmPassword && (
          <p className="text-xs text-red-600">{errors.confirmPassword.message}</p>
        )}
      </div>

      <button
        type="submit"
        disabled={isSubmitting}
        className="h-11 w-full rounded-md bg-primary font-medium text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
      >
        {isSubmitting ? "Resetting..." : "Reset my password"}
      </button>
    </form>
  )
}
