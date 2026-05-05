"use client"

import { useSearchParams } from "next/navigation"
import { useRouter } from "@i18n/navigation"
import { useTranslations } from "next-intl"
import { AccountNav } from "./account-nav"
import { NotificationSettings } from "./notification-settings"
import { EmailSettings } from "./email-settings"
import { PasswordSettings } from "./password-settings"
import { DeleteAccountCard } from "./delete-account-card"
import { useUser } from "@/shared/hooks/use-user"
import { DEFAULT_SECTION, VALID_SECTIONS } from "../types"
import type { AccountSection } from "../types"

/**
 * AccountSettingsPage — Soleil v2 frame for the account preferences area.
 * Layout matches `SoleilAccount` from soleil-lotE.jsx: editorial title,
 * a 240px sidebar of pill tabs + a flexible content column.
 */
export function AccountSettingsPage() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const t = useTranslations("account")
  const { data: user } = useUser()

  const rawSection = searchParams.get("section") || DEFAULT_SECTION
  const section: AccountSection = VALID_SECTIONS.includes(rawSection as AccountSection)
    ? (rawSection as AccountSection)
    : DEFAULT_SECTION

  function handleSectionChange(newSection: AccountSection) {
    router.replace(`/account?section=${newSection}`)
  }

  return (
    <div className="mx-auto max-w-[1100px]">
      <h1 className="mb-7 font-serif text-3xl font-medium tracking-[-0.02em] text-foreground sm:text-[32px]">
        {t("title")}
      </h1>

      <div className="flex flex-col gap-7 lg:grid lg:grid-cols-[240px_1fr] lg:items-start">
        <aside className="w-full shrink-0">
          <AccountNav activeSection={section} onSectionChange={handleSectionChange} />
        </aside>
        <div className="min-w-0">
          {section === "notifications" && <NotificationSettings />}
          {section === "email" && <EmailSettings />}
          {section === "password" && <PasswordSettings />}
          {section === "data-and-deletion" && (
            <DeleteAccountCard
              pendingDeletionAt={user?.deleted_at ?? null}
              hardDeleteAt={user?.hard_delete_at ?? null}
            />
          )}
        </div>
      </div>
    </div>
  )
}
