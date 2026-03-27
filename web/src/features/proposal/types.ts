export type ProposalStatus = "pending" | "accepted" | "declined" | "withdrawn"

export type ProposalFormData = {
  recipientId: string
  conversationId: string
  title: string
  description: string
  amount: string
  deadline: string
  files: File[]
}

export type ProposalMessageMetadata = {
  proposal_id: string
  proposal_title: string
  proposal_amount: number
  proposal_status: ProposalStatus
  proposal_deadline: string | null
  proposal_sender_name: string
  proposal_documents_count: number
}

export function createEmptyProposalForm(): ProposalFormData {
  return {
    recipientId: "",
    conversationId: "",
    title: "",
    description: "",
    amount: "",
    deadline: "",
    files: [],
  }
}
