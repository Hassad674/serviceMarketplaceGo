import type { Metadata } from "next"
import { OpportunityDetail } from "@/features/job/components/opportunity-detail"
import { fetchJobForMetadata } from "@/features/job/api/job-server"
import { safeJsonLd } from "@/shared/lib/json-ld"
import type { JobResponse } from "@/features/job/types"

type Props = {
  params: Promise<{ id: string; locale: string }>
}

// generateMetadata + JobPosting JSON-LD (PERF-W-06). Google for Jobs
// reads the schema directly to enrich the SERP — without this block,
// our opportunities never surface in Google's job-search vertical.
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id } = await params
  const job = await fetchJobForMetadata(id)

  if (!job) {
    return {
      title: "Opportunity | Marketplace Service",
      alternates: { canonical: `/opportunities/${id}` },
    }
  }

  const title = `${job.title} | Marketplace Service`
  const description = (job.description || "").slice(0, 160)

  return {
    title,
    description,
    alternates: { canonical: `/opportunities/${id}` },
    openGraph: {
      type: "article",
      title,
      description,
    },
    twitter: {
      card: "summary",
      title,
      description,
    },
  }
}

export default async function OpportunityDetailPage({ params }: Props) {
  const { id } = await params
  const job = await fetchJobForMetadata(id)
  return (
    <>
      {job ? <JobPostingJsonLd job={job} jobId={id} /> : null}
      <OpportunityDetail jobId={id} />
    </>
  )
}

interface JobPostingJsonLdProps {
  job: JobResponse
  jobId: string
}

function JobPostingJsonLd({ job, jobId }: JobPostingJsonLdProps) {
  // Schema.org JobPosting — see https://schema.org/JobPosting and
  // Google's structured-data guidelines:
  // https://developers.google.com/search/docs/appearance/structured-data/job-posting
  //
  // Required fields (Google): title, description, datePosted,
  // hiringOrganization. We provide the optional baseSalary and
  // jobLocation when available because Google ranks complete
  // postings higher.
  const employmentType = job.budget_type === "long_term" ? "CONTRACT" : "TEMPORARY"
  const payload: Record<string, unknown> = {
    "@context": "https://schema.org",
    "@type": "JobPosting",
    "@id": `/opportunities/${jobId}`,
    title: job.title,
    description: job.description,
    datePosted: job.created_at,
    employmentType,
    hiringOrganization: {
      "@type": "Organization",
      name: "Marketplace Service",
    },
    jobLocation: {
      "@type": "Place",
      address: {
        "@type": "PostalAddress",
        addressCountry: "FR",
      },
    },
    skills: job.skills?.length ? job.skills.join(", ") : undefined,
    baseSalary:
      job.min_budget > 0 || job.max_budget > 0
        ? {
            "@type": "MonetaryAmount",
            currency: "EUR",
            value: {
              "@type": "QuantitativeValue",
              minValue: job.min_budget,
              maxValue: job.max_budget,
              unitText: job.budget_type === "long_term" ? "MONTH" : "PROJECT",
            },
          }
        : undefined,
    validThrough: job.closed_at,
  }
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{ __html: safeJsonLd(payload) }}
    />
  )
}
