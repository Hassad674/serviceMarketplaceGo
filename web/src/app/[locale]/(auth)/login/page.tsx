import { useTranslations } from "next-intl"
import { LoginForm } from "@/features/auth/components/login-form"
import { AuthPageShell } from "@/features/auth/components/auth-page-shell"

// W-01 · Connexion · Soleil v2.
// Shell extracted to <AuthPageShell> so /forgot-password and
// /reset-password reuse the exact same split layout (W-03/W-04).
//
// Source: design/assets/sources/phase1/soleil-lotE.jsx `SoleilLogin`
// (lines 39-130) + design/assets/pdf/web-desktop.pdf p.3.
export default function LoginPage() {
  const t = useTranslations("auth")

  return (
    <AuthPageShell
      eyebrow={t("loginEyebrow")}
      titlePrefix={t("loginTitlePrefix")}
      titleAccent={t("loginTitleAccent")}
      subtitle={t("loginSubtitleSoleil")}
    >
      <LoginForm />
    </AuthPageShell>
  )
}
