"use client"

import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"

export default function MessagesPage() {
  const t = useTranslations("sidebar")

  return (
    <div className="rounded-xl border border-dashed border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 p-12 text-center">
      <MessageSquare className="mx-auto h-10 w-10 text-gray-300 dark:text-gray-600" />
      <h1 className="mt-4 text-lg font-semibold text-gray-900 dark:text-white">
        {t("messages")}
      </h1>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
        Coming soon
      </p>
    </div>
  )
}
