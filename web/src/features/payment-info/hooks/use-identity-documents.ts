"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  listIdentityDocuments,
  uploadIdentityDocument,
  deleteIdentityDocument,
} from "../api/identity-document-api"

const IDENTITY_DOCS_KEY = ["identity-documents"]

export function useIdentityDocuments() {
  return useQuery({
    queryKey: IDENTITY_DOCS_KEY,
    queryFn: listIdentityDocuments,
    staleTime: 30 * 1000,
    refetchOnWindowFocus: false,
  })
}

export function useUploadIdentityDocument() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: { file: File; category: string; documentType: string; side: string }) =>
      uploadIdentityDocument(input.file, input.category, input.documentType, input.side),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: IDENTITY_DOCS_KEY }),
  })
}

export function useDeleteIdentityDocument() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteIdentityDocument(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: IDENTITY_DOCS_KEY }),
  })
}
