import { getTranslations } from "next-intl/server"
import { ResetPasswordForm } from "@/features/auth/components/reset-password-form"
import { AuthPageShell } from "@/features/auth/components/auth-page-shell"

// W-04 · Réinitialiser mot de passe · Soleil v2.
// Reuses the AuthPageShell so the split layout matches login and
// forgot-password to the pixel. The page reads the recovery token
// from the URL and forwards it to <ResetPasswordForm>; the form
// renders an "invalid link" empty state when the token is missing
// or expired (text comes from auth.resetPassword.invalid*).
//
// The page eyebrow / H1 stays "Choisir un nouveau mot de passe"
// regardless of token validity — the token-empty state is rendered
// inside the form (single hero, two form variants).
//
// Source: design/assets/sources/phase1/soleil-lotE.jsx (W-01 layout)
// adapted with reset-specific eyebrow + H1.
export default async function ResetPasswordPage({
  searchParams,
}: {
  searchParams: Promise<{ token?: string }>
}) {
  const { token } = await searchParams
  const t = await getTranslations("auth")

  const hasToken = Boolean(token)

  return (
    <AuthPageShell
      eyebrow={
        hasToken ? t("resetShell.eyebrow") : t("resetShell.invalidEyebrow")
      }
      titlePrefix={
        hasToken
          ? t("resetShell.titlePrefix")
          : t("resetShell.invalidTitlePrefix")
      }
      titleAccent={
        hasToken
          ? t("resetShell.titleAccent")
          : t("resetShell.invalidTitleAccent")
      }
      subtitle={
        hasToken
          ? t("resetShell.subtitle")
          : t("resetShell.invalidSubtitle")
      }
    >
      <ResetPasswordForm token={token || ""} />
    </AuthPageShell>
  )
}
