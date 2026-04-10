package response

import (
	"time"

	"marketplace-backend/internal/domain/dispute"
)

type DisputeResponse struct {
	ID              string  `json:"id"`
	ProposalID      string  `json:"proposal_id"`
	ConversationID  string  `json:"conversation_id"`
	InitiatorID     string  `json:"initiator_id"`
	RespondentID    string  `json:"respondent_id"`
	ClientID        string  `json:"client_id"`
	ProviderID      string  `json:"provider_id"`
	Reason          string  `json:"reason"`
	Description     string  `json:"description"`
	RequestedAmount int64   `json:"requested_amount"`
	ProposalAmount  int64   `json:"proposal_amount"`
	Status          string  `json:"status"`
	ResolutionType  *string `json:"resolution_type"`
	ResAmountClient *int64  `json:"resolution_amount_client"`
	ResAmountProv   *int64  `json:"resolution_amount_provider"`
	ResolutionNote  *string `json:"resolution_note"`
	InitiatorRole   string  `json:"initiator_role"`

	Evidence         []EvidenceResponse         `json:"evidence"`
	CounterProposals []CounterProposalResponse   `json:"counter_proposals"`

	CancellationRequestedBy *string `json:"cancellation_requested_by"`
	CancellationRequestedAt *string `json:"cancellation_requested_at"`

	EscalatedAt *string `json:"escalated_at"`
	ResolvedAt  *string `json:"resolved_at"`
	CreatedAt   string  `json:"created_at"`
}

type AdminDisputeResponse struct {
	DisputeResponse
	AISummary      *string                 `json:"ai_summary"`
	AIBudget       AIBudgetSummary         `json:"ai_budget"`
	AIChatHistory  []AIChatMessageResponse `json:"ai_chat_history"`
}

// AIChatMessageResponse is one persisted turn of the admin AI chat.
// Loaded with the dispute detail and rendered chronologically by the
// admin frontend.
type AIChatMessageResponse struct {
	ID           string `json:"id"`
	Role         string `json:"role"` // "user" or "assistant"
	Content      string `json:"content"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	CreatedAt    string `json:"created_at"`
}

// AIBudgetSummary describes the cumulative AI usage and remaining budget
// for a dispute. Computed from the dispute's tier + tracked tokens, sent
// to the admin frontend so it can render the live cost panel.
type AIBudgetSummary struct {
	Tier                string  `json:"tier"`         // "S" | "M" | "L" | "XL"
	BonusTokens         int     `json:"bonus_tokens"` // cumulative bonus from "Augmenter le budget"
	SummaryUsedTokens   int     `json:"summary_used_tokens"`
	SummaryMaxTokens    int     `json:"summary_max_tokens"`
	ChatUsedTokens      int     `json:"chat_used_tokens"`
	ChatMaxTokens       int     `json:"chat_max_tokens"`
	TotalUsedTokens     int     `json:"total_used_tokens"`
	TotalCostEUR        float64 `json:"total_cost_eur"` // 4-decimal precision
}

type EvidenceResponse struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

type CounterProposalResponse struct {
	ID             string  `json:"id"`
	ProposerID     string  `json:"proposer_id"`
	AmountClient   int64   `json:"amount_client"`
	AmountProvider int64   `json:"amount_provider"`
	Message        string  `json:"message"`
	Status         string  `json:"status"`
	RespondedAt    *string `json:"responded_at"`
	CreatedAt      string  `json:"created_at"`
}

type DisputeListResponse struct {
	Data       []DisputeResponse `json:"data"`
	NextCursor string            `json:"next_cursor"`
	HasMore    bool              `json:"has_more"`
}

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

func NewDisputeResponse(d *dispute.Dispute, ev []*dispute.Evidence, cps []*dispute.CounterProposal) DisputeResponse {
	resp := DisputeResponse{
		ID:              d.ID.String(),
		ProposalID:      d.ProposalID.String(),
		ConversationID:  d.ConversationID.String(),
		InitiatorID:     d.InitiatorID.String(),
		RespondentID:    d.RespondentID.String(),
		ClientID:        d.ClientID.String(),
		ProviderID:      d.ProviderID.String(),
		Reason:          string(d.Reason),
		Description:     d.Description,
		RequestedAmount: d.RequestedAmount,
		ProposalAmount:  d.ProposalAmount,
		Status:          string(d.Status),
		ResAmountClient: d.ResolutionAmountClient,
		ResAmountProv:   d.ResolutionAmountProvider,
		ResolutionNote:  d.ResolutionNote,
		InitiatorRole:   d.InitiatorRole(),
		EscalatedAt:     formatTimePtr(d.EscalatedAt),
		ResolvedAt:      formatTimePtr(d.ResolvedAt),
		CreatedAt:       d.CreatedAt.Format(time.RFC3339),
	}
	if d.ResolutionType != nil {
		s := string(*d.ResolutionType)
		resp.ResolutionType = &s
	}
	if d.CancellationRequestedBy != nil {
		s := d.CancellationRequestedBy.String()
		resp.CancellationRequestedBy = &s
	}
	resp.CancellationRequestedAt = formatTimePtr(d.CancellationRequestedAt)

	resp.Evidence = make([]EvidenceResponse, 0, len(ev))
	for _, e := range ev {
		resp.Evidence = append(resp.Evidence, EvidenceResponse{
			ID:       e.ID.String(),
			Filename: e.Filename,
			URL:      e.URL,
			Size:     e.Size,
			MimeType: e.MimeType,
		})
	}

	resp.CounterProposals = make([]CounterProposalResponse, 0, len(cps))
	for _, cp := range cps {
		resp.CounterProposals = append(resp.CounterProposals, CounterProposalResponse{
			ID:             cp.ID.String(),
			ProposerID:     cp.ProposerID.String(),
			AmountClient:   cp.AmountClient,
			AmountProvider: cp.AmountProvider,
			Message:        cp.Message,
			Status:         string(cp.Status),
			RespondedAt:    formatTimePtr(cp.RespondedAt),
			CreatedAt:      cp.CreatedAt.Format(time.RFC3339),
		})
	}

	return resp
}

func NewAdminDisputeResponse(d *dispute.Dispute, ev []*dispute.Evidence, cps []*dispute.CounterProposal, chats []*dispute.ChatMessage) AdminDisputeResponse {
	chatHistory := make([]AIChatMessageResponse, 0, len(chats))
	for _, c := range chats {
		chatHistory = append(chatHistory, AIChatMessageResponse{
			ID:           c.ID.String(),
			Role:         string(c.Role),
			Content:      c.Content,
			InputTokens:  c.InputTokens,
			OutputTokens: c.OutputTokens,
			CreatedAt:    c.CreatedAt.Format(time.RFC3339),
		})
	}
	return AdminDisputeResponse{
		DisputeResponse: NewDisputeResponse(d, ev, cps),
		AISummary:       d.AISummary,
		AIBudget:        buildAIBudgetSummary(d),
		AIChatHistory:   chatHistory,
	}
}

// AI cost constants — Claude Haiku 4.5 pricing as of 2026.
// Input: $1 per million tokens. Output: $5 per million tokens.
// USD→EUR conversion is a fixed estimate; precise enough for an admin
// dashboard display, easy to swap out if pricing changes.
const (
	AnthropicHaikuInputUSDPerMillion  = 1.0
	AnthropicHaikuOutputUSDPerMillion = 5.0
	USDToEURRate                      = 0.92
)

func buildAIBudgetSummary(d *dispute.Dispute) AIBudgetSummary {
	totalInput := d.AISummaryInputTokens + d.AIChatInputTokens
	totalOutput := d.AISummaryOutputTokens + d.AIChatOutputTokens
	costUSD := (float64(totalInput)*AnthropicHaikuInputUSDPerMillion +
		float64(totalOutput)*AnthropicHaikuOutputUSDPerMillion) / 1_000_000
	return AIBudgetSummary{
		Tier:              string(d.AITier()),
		BonusTokens:       d.AIBudgetBonusTokens,
		SummaryUsedTokens: d.AISummaryUsed(),
		SummaryMaxTokens:  d.AIBudgetSummary(),
		ChatUsedTokens:    d.AIChatUsed(),
		ChatMaxTokens:     d.AIBudgetChat(),
		TotalUsedTokens:   totalInput + totalOutput,
		TotalCostEUR:      costUSD * USDToEURRate,
	}
}

func NewDisputeListItem(d *dispute.Dispute) DisputeResponse {
	return NewDisputeResponse(d, nil, nil)
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
