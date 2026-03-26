export type BudgetType = "ongoing" | "one_time"

export type PaymentFrequency = "hourly" | "weekly" | "monthly"

export type ApplicantType = "all" | "freelancers" | "agencies"

export type JobFormData = {
  title: string
  description: string
  skills: string[]
  tools: string[]
  contractorCount: number
  applicantType: ApplicantType
  budgetType: BudgetType
  // ongoing
  paymentFrequency: PaymentFrequency
  minRate: string
  maxRate: string
  maxHoursPerWeek: number
  // one_time
  minBudget: string
  maxBudget: string
  // common
  estimatedDuration: string
  isIndefinite: boolean
}

export function createDefaultJobFormData(): JobFormData {
  return {
    title: "",
    description: "",
    skills: [],
    tools: [],
    contractorCount: 1,
    applicantType: "all",
    budgetType: "ongoing",
    paymentFrequency: "hourly",
    minRate: "",
    maxRate: "",
    maxHoursPerWeek: 40,
    minBudget: "",
    maxBudget: "",
    estimatedDuration: "",
    isIndefinite: false,
  }
}
