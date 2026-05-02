"use client"

import { useMutation } from "@tanstack/react-query"
import { createReport } from "@/shared/lib/reporting/reporting-api"

/**
 * Shared `createReport` mutation. The reporting UX (P9) lives in
 * `shared/components/reporting/report-dialog`; the hook lives here so
 * cross-feature callers (messaging, job) do not have to import from
 * the reporting feature.
 */
export function useCreateReport() {
  return useMutation({
    mutationFn: createReport,
  })
}
