"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useState } from "react"
import Link from "next/link"
import { forgotPassword } from "@/features/auth/api/auth-api"

const forgotPasswordSchema = z.object({
  email: z.string().email("Invalid email address"),
})

type ForgotPasswordValues = z.infer<typeof forgotPasswordSchema>

export function ForgotPasswordForm() {
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ForgotPasswordValues>({
    resolver: zodResolver(forgotPasswordSchema),
  })

  async function onSubmit(values: ForgotPasswordValues) {
    setError(null)
    try {
      await forgotPassword(values.email)
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
        <h2 className="text-lg font-semibold text-foreground">Email sent</h2>
        <p className="text-sm text-muted-foreground">
          A password reset email has been sent to your address. Check your inbox.
        </p>
        <Link
          href="/login"
          className="inline-block text-sm font-medium text-primary underline underline-offset-4 hover:text-primary/80"
        >
          Back to sign in
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
        <label htmlFor="email" className="block text-sm font-medium text-foreground">
          Email
        </label>
        <input
          id="email"
          type="email"
          autoComplete="email"
          placeholder="you@example.com"
          className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          {...registerField("email")}
        />
        {errors.email && (
          <p className="text-xs text-red-600">{errors.email.message}</p>
        )}
      </div>

      <button
        type="submit"
        disabled={isSubmitting}
        className="h-11 w-full rounded-md bg-primary font-medium text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
      >
        {isSubmitting ? "Sending..." : "Send reset link"}
      </button>

      <p className="text-center text-sm text-muted-foreground">
        <Link
          href="/login"
          className="font-medium text-primary underline underline-offset-4 hover:text-primary/80"
        >
          Back to sign in
        </Link>
      </p>
    </form>
  )
}
