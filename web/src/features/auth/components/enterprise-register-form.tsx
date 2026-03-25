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

const enterpriseSchema = z
  .object({
    display_name: z.string().min(2, "Company name is required"),
    email: z.string().email("Invalid email address"),
    password: z
      .string()
      .min(8, "Minimum 8 characters")
      .regex(/[A-Z]/, "At least one uppercase letter")
      .regex(/[a-z]/, "At least one lowercase letter")
      .regex(/[0-9]/, "At least one digit"),
    confirm_password: z.string(),
  })
  .refine((data) => data.password === data.confirm_password, {
    message: "Passwords do not match",
    path: ["confirm_password"],
  })

type EnterpriseValues = z.infer<typeof enterpriseSchema>

export function EnterpriseRegisterForm() {
  const router = useRouter()
  const { setAuth } = useAuth()
  const [error, setError] = useState<string | null>(null)

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
      const response = await registerUser({
        email: values.email,
        password: values.password,
        first_name: "",
        last_name: "",
        display_name: values.display_name,
        role: "enterprise",
      })
      setAuth(response.user, response.access_token, response.refresh_token)
      router.push("/dashboard/enterprise")
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred")
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

        <div className="space-y-1.5">
          <label htmlFor="display_name" className="block text-sm font-medium text-foreground">
            Company name
          </label>
          <input
            id="display_name"
            type="text"
            autoComplete="organization"
            placeholder="My Company Inc."
            className="h-11 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            {...registerField("display_name")}
          />
          <p className="text-xs text-muted-foreground">Your company name</p>
          {errors.display_name && (
            <p className="text-sm text-destructive">{errors.display_name.message}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="email" className="block text-sm font-medium text-foreground">
            Email
          </label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            placeholder="contact@mycompany.com"
            className="h-11 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            {...registerField("email")}
          />
          {errors.email && (
            <p className="text-sm text-destructive">{errors.email.message}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="password" className="block text-sm font-medium text-foreground">
            Password
          </label>
          <input
            id="password"
            type="password"
            autoComplete="new-password"
            placeholder="Your password"
            className="h-11 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            {...registerField("password")}
          />
          <p className="text-xs text-muted-foreground">
            Minimum 8 characters with uppercase, lowercase and digit
          </p>
          {errors.password && (
            <p className="text-sm text-destructive">{errors.password.message}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="confirm_password" className="block text-sm font-medium text-foreground">
            Confirm password
          </label>
          <input
            id="confirm_password"
            type="password"
            autoComplete="new-password"
            placeholder="Confirm your password"
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
          {isSubmitting ? "Signing up..." : "Create my enterprise account"}
        </button>
      </form>

      <div className="mt-6 flex flex-col items-center gap-3 text-sm text-muted-foreground">
        <Link
          href="/register"
          className="inline-flex items-center gap-1 font-medium text-foreground hover:text-foreground/80"
        >
          <ArrowLeft className="h-4 w-4" />
          Change profile
        </Link>
        <p>
          Already registered?{" "}
          <Link
            href="/login"
            className="font-medium text-primary underline underline-offset-4 hover:text-primary/80"
          >
            Sign In
          </Link>
        </p>
      </div>
    </div>
  )
}
