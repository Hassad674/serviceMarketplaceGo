"use client"

import { useQuery } from "@tanstack/react-query"
import { getCountryFields } from "../api/payment-info-api"

export function useCountryFields(country: string, businessType: string) {
  return useQuery({
    queryKey: ["country-fields", country, businessType],
    queryFn: () => getCountryFields(country, businessType),
    enabled: country.length === 2,
    staleTime: 24 * 60 * 60 * 1000,
  })
}
