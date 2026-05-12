import { useTranslations } from "next-intl"
import { ArrowLeft } from "lucide-react"
import { Link } from "@i18n/navigation"

// PaymentCheckoutShell is the focused minimal chrome rendered on the
// client payment page (`/projects/pay/...`). The dashboard sidebar +
// top header are intentionally absent — a checkout flow is a single-
// task page and the dashboard navigation would compete with the
// payment CTA ("deux navbar" bug fixed here).
//
// What stays:
//   - the brand wordmark on the left, linking back home so the user
//     never feels trapped on the page.
//   - a back-to-dashboard secondary link on the right (escape hatch).
//
// What is removed:
//   - the role-aware Sidebar (with its premium chip, KYC nudge,
//     unread badges, etc.).
//   - the top Header (search, notifications, user menu).
//   - the KYC banner (irrelevant during checkout).
//   - the global ChatWidget and CallSlot (the LiveKit + Crisp chunks
//     are simply not mounted — checkout is faster and quieter).
//
// Auth is still enforced upstream by `middleware.ts` — this component
// is presentation-only.
//
// Soleil v2: ivoire background, faint border under the header, Inter
// Tight for the brand wordmark, corail hover for the back link.

export function PaymentCheckoutShell({
  children,
}: {
  children: React.ReactNode
}) {
  const t = useTranslations("proposal")
  return (
    <div className="min-h-screen bg-background">
      <header
        data-testid="payment-checkout-shell-header"
        className="border-b border-border bg-background/95 backdrop-blur"
      >
        <div className="mx-auto flex h-14 max-w-3xl items-center justify-between px-4">
          <Link
            href="/"
            className="text-[15px] font-bold tracking-tight text-foreground transition-colors hover:text-primary"
          >
            {t("proposalFlow_pay_shellBrand")}
          </Link>
          <Link
            href="/dashboard"
            className="inline-flex items-center gap-1.5 text-[13px] font-medium text-muted-foreground transition-colors hover:text-primary"
          >
            <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
            {t("proposalFlow_pay_shellBackLink")}
          </Link>
        </div>
      </header>
      <main className="mx-auto max-w-3xl px-4 py-8">{children}</main>
    </div>
  )
}
