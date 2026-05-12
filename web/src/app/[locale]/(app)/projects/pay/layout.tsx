import { PaymentCheckoutShell } from "@/shared/components/layouts/payment-checkout-shell"

// Route-segment layout for `/projects/pay/...`. The parent
// `(app)/layout.tsx` recognises this path as "chromeless" and steps
// out of the way (no DashboardShell wrapping), letting this layout
// be the SOLE chrome on screen.
//
// Why a dedicated layout: the parent's dashboard shell renders a
// sidebar + top header, which on a checkout flow reads as "deux
// navbar" — visually noisy and wrong for a single-task page. Lifting
// the chrome here gives the checkout its own focused identity (logo +
// back link) without globally changing how the (app) group renders.
export default function PaymentCheckoutLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return <PaymentCheckoutShell>{children}</PaymentCheckoutShell>
}
