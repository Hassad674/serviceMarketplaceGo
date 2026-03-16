"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useRouter } from "next/navigation"
import { useState } from "react"
import Link from "next/link"
import { login } from "@/features/auth/api/auth-api"
import { useAuth } from "@/shared/hooks/use-auth"

const loginSchema = z.object({
  email: z.string().email("Adresse email invalide"),
  password: z.string().min(8, "Le mot de passe doit contenir au moins 8 caracteres"),
})

type LoginValues = z.infer<typeof loginSchema>

export function LoginForm() {
  const router = useRouter()
  const { setAuth } = useAuth()
  const [error, setError] = useState<string | null>(null)

  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
  })

  async function onSubmit(values: LoginValues) {
    setError(null)
    try {
      const response = await login(values.email, values.password)
      setAuth(response.user, response.access_token, response.refresh_token)

      const dashboardRoutes: Record<string, string> = {
        agency: "/dashboard/agency",
        enterprise: "/dashboard/enterprise",
        provider: "/dashboard/provider",
      }
      router.push(dashboardRoutes[response.user.role] || "/dashboard/provider")
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Une erreur est survenue",
      )
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="rounded-xl border border-border bg-card p-8 shadow-sm space-y-4">
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
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
          placeholder="vous@exemple.com"
          className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          {...registerField("email")}
        />
        {errors.email && (
          <p className="text-xs text-red-600">{errors.email.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <label htmlFor="password" className="block text-sm font-medium text-foreground">
          Mot de passe
        </label>
        <input
          id="password"
          type="password"
          autoComplete="current-password"
          placeholder="Votre mot de passe"
          className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          {...registerField("password")}
        />
        {errors.password && (
          <p className="text-xs text-red-600">{errors.password.message}</p>
        )}
      </div>

      <button
        type="submit"
        disabled={isSubmitting}
        className="h-11 w-full rounded-md bg-primary font-medium text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
      >
        {isSubmitting ? "Connexion..." : "Se connecter"}
      </button>

      <p className="text-center text-sm text-muted-foreground">
        Pas encore de compte ?{" "}
        <Link href="/register" className="font-medium text-primary underline underline-offset-4 hover:text-primary/80">
          Creer un compte
        </Link>
      </p>
    </form>
  )
}
