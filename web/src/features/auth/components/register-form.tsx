"use client"

import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { useRouter } from "next/navigation"
import { useState } from "react"
import Link from "next/link"
import { register as registerUser } from "@/features/auth/api/auth-api"
import { useAuth } from "@/shared/hooks/use-auth"

const registerSchema = z.object({
  email: z.string().email("Adresse email invalide"),
  password: z
    .string()
    .min(8, "Le mot de passe doit contenir au moins 8 caracteres"),
  first_name: z.string().min(1, "Le prenom est requis"),
  last_name: z.string().min(1, "Le nom est requis"),
  display_name: z.string().min(1, "Le nom d'affichage est requis"),
  role: z.enum(["agency", "enterprise", "provider"], {
    required_error: "Veuillez selectionner un role",
  }),
})

type RegisterValues = z.infer<typeof registerSchema>

const roleLabels: Record<RegisterValues["role"], string> = {
  agency: "Agence",
  enterprise: "Entreprise",
  provider: "Prestataire / Freelance",
}

export function RegisterForm() {
  const router = useRouter()
  const { setAuth } = useAuth()
  const [error, setError] = useState<string | null>(null)

  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<RegisterValues>({
    resolver: zodResolver(registerSchema),
  })

  async function onSubmit(values: RegisterValues) {
    setError(null)
    try {
      const response = await registerUser(values)
      setAuth(response.user, response.access_token, response.refresh_token)

      const dashboardPath = `/dashboard/${response.user.role}`
      router.push(dashboardPath)
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Une erreur est survenue",
      )
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <label htmlFor="first_name" className="block text-sm font-medium text-gray-700">
            Prenom
          </label>
          <input
            id="first_name"
            type="text"
            autoComplete="given-name"
            placeholder="Jean"
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm placeholder:text-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
            {...registerField("first_name")}
          />
          {errors.first_name && (
            <p className="text-xs text-red-600">{errors.first_name.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <label htmlFor="last_name" className="block text-sm font-medium text-gray-700">
            Nom
          </label>
          <input
            id="last_name"
            type="text"
            autoComplete="family-name"
            placeholder="Dupont"
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm placeholder:text-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
            {...registerField("last_name")}
          />
          {errors.last_name && (
            <p className="text-xs text-red-600">{errors.last_name.message}</p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <label htmlFor="display_name" className="block text-sm font-medium text-gray-700">
          Nom d&apos;affichage
        </label>
        <input
          id="display_name"
          type="text"
          placeholder="Jean Dupont ou Mon Agence"
          className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm placeholder:text-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
          {...registerField("display_name")}
        />
        {errors.display_name && (
          <p className="text-xs text-red-600">{errors.display_name.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <label htmlFor="email" className="block text-sm font-medium text-gray-700">
          Email
        </label>
        <input
          id="email"
          type="email"
          autoComplete="email"
          placeholder="vous@exemple.com"
          className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm placeholder:text-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
          {...registerField("email")}
        />
        {errors.email && (
          <p className="text-xs text-red-600">{errors.email.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <label htmlFor="password" className="block text-sm font-medium text-gray-700">
          Mot de passe
        </label>
        <input
          id="password"
          type="password"
          autoComplete="new-password"
          placeholder="Minimum 8 caracteres"
          className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm placeholder:text-gray-400 focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
          {...registerField("password")}
        />
        {errors.password && (
          <p className="text-xs text-red-600">{errors.password.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <label htmlFor="role" className="block text-sm font-medium text-gray-700">
          Vous etes
        </label>
        <select
          id="role"
          className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-gray-500 focus:outline-none focus:ring-1 focus:ring-gray-500"
          {...registerField("role")}
        >
          <option value="">Selectionnez votre role</option>
          {(Object.entries(roleLabels) as [RegisterValues["role"], string][]).map(
            ([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ),
          )}
        </select>
        {errors.role && (
          <p className="text-xs text-red-600">{errors.role.message}</p>
        )}
      </div>

      <button
        type="submit"
        disabled={isSubmitting}
        className="w-full rounded-lg bg-gray-900 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-gray-800 disabled:opacity-50"
      >
        {isSubmitting ? "Inscription..." : "Creer mon compte"}
      </button>

      <p className="text-center text-sm text-gray-500">
        Deja inscrit ?{" "}
        <Link
          href="/login"
          className="font-medium text-gray-900 underline underline-offset-4 hover:text-gray-700"
        >
          Se connecter
        </Link>
      </p>
    </form>
  )
}
