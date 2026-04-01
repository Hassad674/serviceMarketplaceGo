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

export type JobApplicationResponse = {
  id: string
  job_id: string
  applicant_id: string
  message: string
  video_url?: string
  created_at: string
}

export type PublicProfileSummary = {
  user_id: string
  display_name: string
  first_name: string
  last_name: string
  role: string
  title: string
  photo_url: string
  referrer_enabled: boolean
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
