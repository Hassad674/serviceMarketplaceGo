export type ProposalStatus = "pending" | "accepted" | "declined" | "withdrawn"
export type ProposalPaymentType = "escrow" | "invoice"
export type ProposalEscrowStructure = "milestone" | "one-time"
export type ProposalInvoiceBilling = "fixed" | "hourly"
export type ProposalInvoiceFrequency = "weekly" | "bi-weekly" | "monthly"

export type ProposalMilestone = {
  id: string
  title: string
  description: string
  deadline: string
  amount: string
}

export type ProposalFormData = {
  paymentType: ProposalPaymentType
  escrowStructure: ProposalEscrowStructure
  milestones: ProposalMilestone[]
  oneTimeAmount: string
  invoiceBillingType: ProposalInvoiceBilling
  invoiceRate: string
  invoiceFrequency: ProposalInvoiceFrequency
  invoiceAmount: string
  title: string
  description: string
  skills: string[]
  startDate: string
  deadline: string
  isOngoing: boolean
  isNegotiable: boolean
}

export type ProposalMessageMetadata = {
  proposal_id: string
  proposal_title: string
  proposal_total_amount: number
  proposal_payment_type: ProposalPaymentType
  proposal_milestones_count: number
  proposal_status: ProposalStatus
  proposal_sender_name: string
}

export function createEmptyMilestone(): ProposalMilestone {
  return {
    id: crypto.randomUUID(),
    title: "",
    description: "",
    deadline: "",
    amount: "",
  }
}

export function createDefaultProposalForm(): ProposalFormData {
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
    isNegotiable: false,
  }
}

export function computeProposalTotal(data: ProposalFormData): number {
  if (data.paymentType === "escrow") {
    if (data.escrowStructure === "one-time") {
      return Number(data.oneTimeAmount) || 0
    }
    return data.milestones.reduce(
      (sum, m) => sum + (Number(m.amount) || 0),
      0,
    )
  }
  return Number(data.invoiceAmount) || 0
}
