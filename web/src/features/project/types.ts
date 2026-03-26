export type PaymentType = "escrow" | "invoice"

export type EscrowStructure = "milestone" | "one-time"

export type InvoiceBillingType = "fixed" | "hourly"

export type InvoiceFrequency = "weekly" | "bi-weekly" | "monthly"

export type ApplicantType = "all" | "freelancers" | "agencies"

export type Milestone = {
  id: string
  title: string
  description: string
  deadline: string
  amount: string
}

export type ProjectFormData = {
  paymentType: PaymentType
  escrowStructure: EscrowStructure
  milestones: Milestone[]
  oneTimeAmount: string
  invoiceBillingType: InvoiceBillingType
  invoiceRate: string
  invoiceFrequency: InvoiceFrequency
  invoiceAmount: string
  title: string
  description: string
  skills: string[]
  startDate: string
  deadline: string
  isOngoing: boolean
  applicantType: ApplicantType
  isNegotiable: boolean
}

export function createEmptyMilestone(): Milestone {
  return {
    id: crypto.randomUUID(),
    title: "",
    description: "",
    deadline: "",
    amount: "",
  }
}

export function createDefaultFormData(): ProjectFormData {
  return {
    paymentType: "escrow",
    escrowStructure: "milestone",
    milestones: [createEmptyMilestone(), createEmptyMilestone()],
    oneTimeAmount: "",
    invoiceBillingType: "fixed",
    invoiceRate: "",
    invoiceFrequency: "weekly",
    invoiceAmount: "",
    title: "",
    description: "",
    skills: [],
    startDate: "",
    deadline: "",
    isOngoing: false,
    applicantType: "all",
    isNegotiable: false,
  }
}
