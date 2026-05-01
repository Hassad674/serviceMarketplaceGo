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
    <div className="mx-auto max-w-4xl">
      <h1 className="mb-6 text-2xl font-bold text-slate-900 dark:text-slate-100">
        {t("title")}
      </h1>

      <div className="flex flex-col gap-6 lg:flex-row">
        <aside className="w-full shrink-0 lg:w-56">
          <AccountNav activeSection={section} onSectionChange={handleSectionChange} />
        </aside>
        <div className="min-w-0 flex-1">
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
