import Link from "next/link"
import { Building2, User, Briefcase, ArrowRight } from "lucide-react"

const roles = [
  {
    id: "agency",
    title: "Agency",
    description: "Manage your missions, your team and your visibility on the marketplace.",
    icon: Building2,
    href: "/register/agency",
    gradient: "from-blue-500 to-blue-600",
    hoverBorder: "hover:border-blue-400",
    hoverShadow: "hover:shadow-blue-100",
  },
  {
    id: "provider",
    title: "Freelance / Business Referrer",
    description: "Find missions, manage your invoices and grow your professional network.",
    icon: User,
    href: "/register/provider",
    gradient: "from-rose-500 to-rose-600",
    hoverBorder: "hover:border-rose-400",
    hoverShadow: "hover:shadow-rose-100",
  },
  {
    id: "enterprise",
    title: "Enterprise",
    description: "Publish projects and find the best service providers for your needs.",
    icon: Briefcase,
    href: "/register/enterprise",
    gradient: "from-emerald-500 to-emerald-600",
    hoverBorder: "hover:border-emerald-400",
    hoverShadow: "hover:shadow-emerald-100",
  },
]

export default function RegisterPage() {
  return (
    <div className="mx-auto w-full max-w-3xl space-y-8">
      <div className="text-center">
        <h1 className="text-2xl font-bold tracking-tight text-gray-900">
          Create your account
        </h1>
        <p className="mt-2 text-sm text-gray-500">
          Choose your professional profile to get started
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        {roles.map((role) => {
          const Icon = role.icon
          return (
            <Link
              key={role.id}
              href={role.href}
              className={`animate-scale-in group flex flex-col items-center rounded-2xl border-2 border-gray-100 bg-white p-8 text-center shadow-sm transition-all duration-200 ${role.hoverBorder} ${role.hoverShadow} hover:shadow-lg`}
            >
              <div className={`mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br ${role.gradient}`}>
                <Icon className="h-7 w-7 text-white" />
              </div>
              <h2 className="text-lg font-bold text-gray-900">
                {role.title}
              </h2>
              <p className="mt-2 text-sm leading-relaxed text-gray-500">
                {role.description}
              </p>
              <div className="mt-5 flex items-center gap-1 text-sm font-medium text-gray-400 transition-colors group-hover:text-gray-900">
                Get started
                <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />
              </div>
            </Link>
          )
        })}
      </div>

      <p className="text-center text-sm text-gray-500">
        Already registered?{" "}
        <Link
          href="/login"
          className="font-medium text-rose-500 hover:text-rose-600"
        >
          Sign In
        </Link>
      </p>
    </div>
  )
}
