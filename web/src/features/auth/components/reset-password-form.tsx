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
      .min(8, "Le mot de passe doit contenir au moins 8 caracteres")
      .regex(/[A-Z]/, "Le mot de passe doit contenir au moins une majuscule")
      .regex(/[a-z]/, "Le mot de passe doit contenir au moins une minuscule")
      .regex(/[0-9]/, "Le mot de passe doit contenir au moins un chiffre")
      .regex(/[^A-Za-z0-9]/, "Le mot de passe doit contenir au moins un caractere special"),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Les mots de passe ne correspondent pas",
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
        <h2 className="text-lg font-semibold text-foreground">Lien invalide</h2>
        <p className="text-sm text-muted-foreground">
          Ce lien de reinitialisation est invalide ou a expire.
        </p>
        <Link
          href="/forgot-password"
          className="inline-block text-sm font-medium text-primary underline underline-offset-4 hover:text-primary/80"
        >
          Demander un nouveau lien
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
        err instanceof Error ? err.message : "Une erreur est survenue",
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
        <h2 className="text-lg font-semibold text-foreground">Mot de passe reinitialise avec succes !</h2>
        <p className="text-sm text-muted-foreground">
          Vous pouvez maintenant vous connecter avec votre nouveau mot de passe.
        </p>
        <Link
          href="/login"
          className="inline-block rounded-md bg-primary px-6 py-2.5 text-sm font-medium text-primary-foreground shadow-sm hover:bg-primary/90"
        >
          Se connecter
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
          Nouveau mot de passe
        </label>
        <input
          id="password"
          type="password"
          autoComplete="new-password"
          placeholder="Minimum 8 caracteres"
          className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          {...registerField("password")}
        />
        {errors.password && (
          <p className="text-xs text-red-600">{errors.password.message}</p>
        )}
        <p className="text-xs text-muted-foreground">
          8 caracteres minimum, une majuscule, une minuscule, un chiffre et un caractere special
        </p>
      </div>

      <div className="space-y-2">
        <label htmlFor="confirmPassword" className="block text-sm font-medium text-foreground">
          Confirmer le mot de passe
        </label>
        <input
          id="confirmPassword"
          type="password"
          autoComplete="new-password"
          placeholder="Confirmez votre mot de passe"
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
        {isSubmitting ? "Reinitialisation..." : "Reinitialiser mon mot de passe"}
      </button>
    </form>
  )
}
