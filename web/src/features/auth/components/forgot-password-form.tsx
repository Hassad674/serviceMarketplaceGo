"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useState } from "react"
import { Link } from "@i18n/navigation"
import { CheckCircle2 } from "lucide-react"
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
      <div className="animate-scale-in rounded-2xl border border-gray-100 bg-white p-8 shadow-lg space-y-4 text-center">
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-full bg-emerald-100">
          <CheckCircle2 className="h-7 w-7 text-emerald-600" />
        </div>
        <h2 className="text-lg font-bold text-gray-900">Email sent</h2>
        <p className="text-sm text-gray-500">
          A password reset email has been sent to your address. Check your inbox.
        </p>
        <Link
          href="/login"
          className="inline-block text-sm font-medium text-rose-500 hover:text-rose-600"
        >
          Back to sign in
        </Link>
      </div>
    )
  }

  return (
    <div className="animate-scale-in rounded-2xl border border-gray-100 bg-white p-8 shadow-lg">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
        {error && (
          <div className="rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-600" role="alert">
            {error}
          </div>
        )}

        <div className="space-y-1.5">
          <label htmlFor="email" className="block text-sm font-medium text-gray-700">
            Email
          </label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            placeholder="you@example.com"
            className="h-12 w-full rounded-xl border border-gray-200 bg-white px-4 text-sm transition-all placeholder:text-gray-400 focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
            {...registerField("email")}
          />
          {errors.email && (
            <p className="text-sm text-red-500 mt-1">{errors.email.message}</p>
          )}
        </div>

        <button
          type="submit"
          disabled={isSubmitting}
          className="gradient-primary h-12 w-full rounded-xl font-semibold text-white shadow-md transition-all hover:shadow-glow active:scale-[0.98] disabled:opacity-50"
        >
          {isSubmitting ? "Sending..." : "Send reset link"}
        </button>
      </form>

      <p className="mt-6 text-center text-sm text-gray-500">
        <Link
          href="/login"
          className="font-medium text-rose-500 hover:text-rose-600"
        >
          Back to sign in
        </Link>
      </p>
    </div>
  )
}
