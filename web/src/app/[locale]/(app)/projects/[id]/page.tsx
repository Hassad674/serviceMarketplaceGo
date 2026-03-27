"use client"

import { use } from "react"
import { ProposalDetailView } from "@/features/proposal/components/proposal-detail-view"

export default function ProjectDetailPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = use(params)
  return <ProposalDetailView proposalId={id} />
}
