import { OpportunityDetail } from "@/features/job/components/opportunity-detail"

export default async function OpportunityDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params
  return <OpportunityDetail jobId={id} />
}
