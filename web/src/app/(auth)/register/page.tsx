import Link from "next/link"
import { Building2, User, Briefcase, ArrowRight } from "lucide-react"

const roles = [
  {
    id: "agency",
    title: "Agence",
    description: "Gérez vos missions, votre équipe et votre visibilité.",
    icon: Building2,
    href: "/register/agency",
    accent: "border-blue-200 hover:border-blue-400 hover:bg-blue-50/50",
    iconBg: "bg-blue-50",
    iconColor: "text-blue-500",
  },
  {
    id: "provider",
    title: "Freelance / Apporteur d'affaire",
    description: "Gérez vos missions et développez votre activité.",
    icon: User,
    href: "/register/provider",
    accent: "border-rose-200 hover:border-rose-400 hover:bg-rose-50/50",
    iconBg: "bg-rose-50",
    iconColor: "text-rose-500",
  },
  {
    id: "enterprise",
    title: "Entreprise",
    description: "Trouvez les meilleurs prestataires pour vos projets.",
    icon: Briefcase,
    href: "/register/enterprise",
    accent: "border-emerald-200 hover:border-emerald-400 hover:bg-emerald-50/50",
    iconBg: "bg-emerald-50",
    iconColor: "text-emerald-500",
  },
]

export default function RegisterPage() {
  return (
    <div className="mx-auto w-full max-w-2xl space-y-8">
      <div className="text-center">
        <h1 className="text-2xl font-bold text-gray-900">Créer un compte</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          Choisissez votre profil professionnel
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        {roles.map((role) => {
          const Icon = role.icon
          return (
            <Link
              key={role.id}
              href={role.href}
              className={`group flex flex-col items-center rounded-xl border bg-card p-8 text-center shadow-sm transition-all duration-150 hover:shadow-md hover:scale-[1.02] ${role.accent}`}
            >
              <div className={`mb-4 flex h-14 w-14 items-center justify-center rounded-xl ${role.iconBg}`}>
                <Icon className={`h-7 w-7 ${role.iconColor}`} />
              </div>
              <h2 className="text-sm font-semibold text-gray-900">
                {role.title}
              </h2>
              <p className="mt-2 text-xs text-muted-foreground leading-relaxed">
                {role.description}
              </p>
              <ArrowRight className="mt-4 h-4 w-4 text-muted-foreground transition-transform group-hover:translate-x-1" />
            </Link>
          )
        })}
      </div>

      <p className="text-center text-sm text-muted-foreground">
        Déjà inscrit ?{" "}
        <Link
          href="/login"
          className="font-medium text-primary underline underline-offset-4 hover:text-primary/80"
        >
          Se connecter
        </Link>
      </p>
    </div>
  )
}
