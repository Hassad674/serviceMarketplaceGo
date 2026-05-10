export type BudgetType = "one_shot" | "long_term"
export type ApplicantType = "all" | "freelancers" | "agencies"
export type PaymentFrequency = "weekly" | "monthly"
export type DescriptionType = "text" | "video" | "both"

export type JobFormData = {
  title: string
  description: string
  skills: string[]
  applicantType: ApplicantType
  budgetType: BudgetType
  minBudget: string
  maxBudget: string
  paymentFrequency: PaymentFrequency
  durationWeeks: string
  isIndefinite: boolean
  descriptionType: DescriptionType
  videoUrl: string
  videoFile: File | null
}

export function createDefaultJobFormData(): JobFormData {
  return {
    title: "",
    description: "",
    skills: [],
    applicantType: "all",
    budgetType: "one_shot",
    minBudget: "",
    maxBudget: "",
    paymentFrequency: "monthly",
    durationWeeks: "",
    isIndefinite: false,
    descriptionType: "text",
    videoUrl: "",
    videoFile: null,
  }
}

export type JobResponse = {
  id: string
  creator_id: string
  title: string
  description: string
  skills: string[]
  applicant_type: string
  budget_type: string
  min_budget: number
  max_budget: number
  status: string
  created_at: string
  updated_at: string
  closed_at?: string
  payment_frequency?: string
  duration_weeks?: number
  is_indefinite: boolean
  description_type: string
  video_url?: string
  /**
   * Public count of applications/candidatures on the job. Optional because
   * only the marketplace feed (`/jobs/open`) returns it; single GET / owner
   * list endpoints omit it. Zero is a legitimate value ("be the first to
   * apply" UX) — distinct from undefined ("not exposed by this endpoint").
   */
  total_applicants?: number
}

export type JobListResponse = {
  data: JobResponse[]
  next_cursor: string
  has_more: boolean
}

export type JobWithCountsResponse = JobResponse & {
  total_applicants: number
  new_applicants: number
}

export type JobWithCountsListResponse = {
  data: JobWithCountsResponse[]
  next_cursor: string
  has_more: boolean
}

// --- Job Application types ---

// ApplicantKind is the persona under which an application was filed.
// 'freelance' and 'agency' are the default kinds derived from the
// applicant's role; 'referrer' is set explicitly when a provider with
// referrer_enabled applies as an apporteur d'affaires (broker the deal
// for a commission).
export type ApplicantKind = "freelance" | "agency" | "referrer"

// Since phase R3, a job application is owned by an organization, not
// an individual user — see ApplicationWithProfile.profile.organization_id
// for the canonical owner id consumed by the public profile route + chat
// widget. applicant_id is the applicant USER id (audit/authorship), kept
// for legacy clients but NEVER use it as a route id; it 404s the public
// profile endpoint which expects an organization id.
export type JobApplicationResponse = {
  id: string
  job_id: string
  applicant_id: string
  applicant_kind: ApplicantKind
  message: string
  video_url?: string
  created_at: string
}

// Mirror of backend/internal/handler/dto/response/profile.go. Describes
// the org behind the application, not a user.
export type PublicProfileSummary = {
  organization_id: string
  name: string
  org_type: string
  title: string
  photo_url: string
  referrer_enabled: boolean
  average_rating: number
  review_count: number
}

export type ApplicationWithProfile = {
  application: JobApplicationResponse
  profile: PublicProfileSummary
}

export type ApplicationWithJob = {
  application: JobApplicationResponse
  job: JobResponse
}

export type ApplicationListResponse = {
  data: ApplicationWithProfile[]
  next_cursor: string
  has_more: boolean
}

export type MyApplicationListResponse = {
  data: ApplicationWithJob[]
  next_cursor: string
  has_more: boolean
}

export type OpenJobListFilters = {
  skills?: string[]
  applicant_type?: string
  budget_type?: string
  min_budget?: number
  max_budget?: number
  search?: string
}
