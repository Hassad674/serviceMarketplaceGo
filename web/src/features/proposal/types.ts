export type ProposalStatus =
  | "pending"
  | "accepted"
  | "declined"
  | "withdrawn"
  | "paid"
  | "active"
  | "completion_requested"
  | "completed"

export type ProposalDocument = {
  id: string
  filename: string
  url: string
  size: number
  mime_type: string
}

export type ProposalResponse = {
  id: string
  conversation_id: string
  sender_id: string
  recipient_id: string
  title: string
  description: string
  amount: number
  deadline: string | null
  status: ProposalStatus
  parent_id: string | null
  version: number
  client_id: string
  provider_id: string
  client_name: string
  provider_name: string
  documents: ProposalDocument[]
  accepted_at: string | null
  paid_at: string | null
  created_at: string
}

export type ProjectListResponse = {
  data: ProposalResponse[]
  next_cursor: string
  has_more: boolean
}

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
  proposal_version: number
  proposal_parent_id: string | null
  proposal_client_id: string
  proposal_provider_id: string
}

export type PaymentIntentResponse = {
  client_secret?: string
  payment_record_id?: string
  amounts?: {
    proposal_amount: number
    stripe_fee: number
    platform_fee: number
    client_total: number
    provider_payout: number
  }
  status?: string // "paid" for simulation mode
}

export type UploadURLResponse = {
  upload_url: string
  file_key: string
  public_url: string
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
