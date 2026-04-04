export type AdminJobAuthor = {
  id: string
  display_name: string
  email: string
  role: string
}

export type AdminJob = {
  id: string
  title: string
  description: string
  skills: string[]
  applicant_type: string
  budget_type: string
  min_budget: number
  max_budget: number
  status: "open" | "closed"
  created_at: string
  updated_at: string
  closed_at?: string
  payment_frequency?: string
  duration_weeks?: number
  is_indefinite: boolean
  description_type: string
  video_url?: string
  application_count: number
  author: AdminJobAuthor
}

export type AdminJobApplicationCandidate = {
  id: string
  display_name: string
  email: string
  role: string
}

export type AdminJobApplicationJob = {
  id: string
  title: string
  status: string
}

export type AdminJobApplication = {
  id: string
  message: string
  video_url?: string
  created_at: string
  updated_at: string
  candidate: AdminJobApplicationCandidate
  job: AdminJobApplicationJob
}

export type AdminJobListResponse = {
  data: AdminJob[]
  next_cursor: string
  has_more: boolean
  total: number
  page: number
  total_pages: number
}

export type AdminJobDetailResponse = {
  data: AdminJob
}

export type AdminJobApplicationListResponse = {
  data: AdminJobApplication[]
  next_cursor: string
  has_more: boolean
  total: number
  page: number
  total_pages: number
}

export type JobFilters = {
  status: string
  search: string
  sort: string
  page: number
}

export type ApplicationFilters = {
  job_id: string
  search: string
  sort: string
  page: number
}
