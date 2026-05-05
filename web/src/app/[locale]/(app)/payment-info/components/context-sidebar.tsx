"use client"

import { Clock, FileCheck, HelpCircle } from "lucide-react"

const SECTIONS = [
  {
    icon: FileCheck,
    title: "Pourquoi ces informations ?",
    body:
      "La réglementation KYC (Know Your Customer) nous oblige à vérifier votre identité avant de vous permettre de recevoir des paiements. C'est une garantie pour vous et pour les clients.",
  },
  {
    icon: Clock,
    title: "Combien de temps ?",
    body:
      "Validation quasi-immédiate dans 80% des cas. Si un document nécessite une revue manuelle, comptez 1 à 2 jours ouvrés.",
  },
  {
    icon: HelpCircle,
    title: "Besoin d'aide ?",
    body:
      "Notre équipe support est disponible 5j/7 pour vous accompagner. Vous pouvez interrompre et reprendre à tout moment.",
  },
]

export function ContextSidebar() {
  return (
    <aside
      aria-label="Informations complémentaires"
      className="sticky top-6 hidden w-full max-w-[320px] flex-col gap-4 lg:flex"
    >
      {SECTIONS.map((section) => {
        const Icon = section.icon
        return (
          <div
            key={section.title}
            className="rounded-2xl border border-border bg-card/80 p-5 backdrop-blur-sm"
          >
            <div className="mb-2 flex items-center gap-2.5">
              <span
                className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary-soft text-primary"
                aria-hidden
              >
                <Icon className="h-4 w-4" />
              </span>
              <h3 className="font-serif text-[16px] font-medium tracking-[-0.01em] text-foreground">
                {section.title}
              </h3>
            </div>
            <p className="text-[13px] leading-relaxed text-muted-foreground">
              {section.body}
            </p>
          </div>
        )
      })}
    </aside>
  )
}
