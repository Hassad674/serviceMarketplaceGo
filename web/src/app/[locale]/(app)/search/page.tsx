"use client"

import { useSearchParams } from "next/navigation"
import { SearchPage } from "@/features/provider/components/search-page"
import type { SearchType } from "@/features/provider/api/search-api"

const VALID_TYPES: SearchType[] = ["freelancer", "agency", "referrer"]

export default function SearchRoutePage() {
  const searchParams = useSearchParams()
  const typeParam = searchParams.get("type")
  const type: SearchType = VALID_TYPES.includes(typeParam as SearchType)
    ? (typeParam as SearchType)
    : "freelancer"

  return <SearchPage type={type} />
}
