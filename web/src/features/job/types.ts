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
