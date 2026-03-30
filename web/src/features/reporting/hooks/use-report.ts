"use client"

import { useMutation } from "@tanstack/react-query"
import { createReport } from "../api/reporting-api"

export function useCreateReport() {
  return useMutation({
    mutationFn: createReport,
  })
}
