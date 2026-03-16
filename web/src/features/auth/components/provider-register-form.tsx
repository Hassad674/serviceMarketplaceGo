"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useRouter } from "next/navigation"
import { useState } from "react"
import Link from "next/link"
import { ArrowLeft } from "lucide-react"
import { register as registerUser } from "@/features/auth/api/auth-api"
import { useAuth } from "@/shared/hooks/use-auth"

const providerSchema = z
  .object({
    first_name: z.string().min(1, "Le prénom est requis"),
    last_name: z.string().min(1, "Le nom est requis"),
    email: z.string().email("Adresse email invalide"),
    password: z
      .string()
      .min(8, "Minimum 8 caractères")
      .regex(/[A-Z]/, "Au moins une majuscule")
      .regex(/[a-z]/, "Au moins une minuscule")
      .regex(/[0-9]/, "Au moins un chiffre"),
    confirm_password: z.string(),
  })
  .refine((data) => data.password === data.confirm_password, {
    message: "Les mots de passe ne correspondent pas",
    path: ["confirm_password"],
  })

type ProviderValues = z.infer<typeof providerSchema>

export function ProviderRegisterForm() {
  const router = useRouter()
  const { setAuth } = useAuth()
  const [error, setError] = useState<string | null>(null)

  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ProviderValues>({
    resolver: zodResolver(providerSchema),
  })

  async function onSubmit(values: ProviderValues) {
    setError(null)
    try {
      const response = await registerUser({
        email: values.email,
        password: values.password,
        first_name: values.first_name,
        last_name: values.last_name,
        display_name: "",
        role: "provider",
      })
      setAuth(response.user, response.access_token, response.refresh_token)
      router.push("/dashboard/provider")
    } catch (err) {
      setError(err instanceof Error ? err.message : "Une erreur est survenue")
    }
  }

  return (
    <div className="rounded-xl border border-border bg-card p-8 shadow-sm">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        {error && (
          <div role="alert" className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
            {error}
          </div>
        )}

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1.5">
            <label htmlFor="first_name" className="block text-sm font-medium text-foreground">
              Prénom
            </label>
            <input
              id="first_name"
              type="text"
              autoComplete="given-name"
              placeholder="Jean"
              className="h-11 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              {...registerField("first_name")}
            />
            {errors.first_name && (
              <p className="text-sm text-destructive">{errors.first_name.message}</p>
            )}
          </div>

          <div className="space-y-1.5">
            <label htmlFor="last_name" className="block text-sm font-medium text-foreground">
              Nom
            </label>
            <input
              id="last_name"
              type="text"
              autoComplete="family-name"
              placeholder="Dupont"
              className="h-11 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              {...registerField("last_name")}
            />
            {errors.last_name && (
              <p className="text-sm text-destructive">{errors.last_name.message}</p>
            )}
          </div>
        </div>

        <div className="space-y-1.5">
          <label htmlFor="email" className="block text-sm font-medium text-foreground">
            Email
          </label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            placeholder="jean.dupont@email.com"
            className="h-11 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            {...registerField("email")}
          />
          {errors.email && (
            <p className="text-sm text-destructive">{errors.email.message}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="password" className="block text-sm font-medium text-foreground">
            Mot de passe
          </label>
          <input
            id="password"
            type="password"
            autoComplete="new-password"
            placeholder="Votre mot de passe"
            className="h-11 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            {...registerField("password")}
          />
          <p className="text-xs text-muted-foreground">
            Minimum 8 caractères avec majuscule, minuscule et chiffre
          </p>
          {errors.password && (
            <p className="text-sm text-destructive">{errors.password.message}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="confirm_password" className="block text-sm font-medium text-foreground">
            Confirmer le mot de passe
          </label>
          <input
            id="confirm_password"
            type="password"
            autoComplete="new-password"
            placeholder="Confirmez votre mot de passe"
            className="h-11 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            {...registerField("confirm_password")}
          />
          {errors.confirm_password && (
            <p className="text-sm text-destructive">{errors.confirm_password.message}</p>
          )}
        </div>

        <button
          type="submit"
          disabled={isSubmitting}
          className="h-11 w-full rounded-md bg-primary font-medium text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
        >
          {isSubmitting ? "Inscription..." : "Créer mon compte freelance"}
        </button>
      </form>

      <div className="mt-6 flex flex-col items-center gap-3 text-sm text-muted-foreground">
        <Link
          href="/register"
          className="inline-flex items-center gap-1 font-medium text-foreground hover:text-foreground/80"
        >
          <ArrowLeft className="h-4 w-4" />
          Changer de profil
        </Link>
        <p>
          Déjà inscrit ?{" "}
          <Link
            href="/login"
            className="font-medium text-primary underline underline-offset-4 hover:text-primary/80"
          >
            Se connecter
          </Link>
        </p>
      </div>
    </div>
  )
}
