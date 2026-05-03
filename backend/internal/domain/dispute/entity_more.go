package dispute

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

func (d *Dispute) RecordActivity() {
	d.LastActivityAt = time.Now()
	d.UpdatedAt = d.LastActivityAt
}

func (d *Dispute) RecordRespondentReply() {
	if d.RespondentFirstReplyAt == nil {
		now := time.Now()
		d.RespondentFirstReplyAt = &now
	}
}

func (d *Dispute) SetAISummary(summary string) {
	d.AISummary = &summary
	d.UpdatedAt = time.Now()
}

// ---------------------------------------------------------------------------
// AI budget tier and tracking
// ---------------------------------------------------------------------------

// AITier indicates the AI budget tier of a dispute, derived from the
// proposal amount. Higher tiers grant a larger AI budget so high-stakes
// disputes can fund a deeper investigation.
type AITier string

const (
	AITierS  AITier = "S"  // < 500 EUR
	AITierM  AITier = "M"  // 500 - 5 000 EUR
	AITierL  AITier = "L"  // 5 000 - 20 000 EUR
	AITierXL AITier = "XL" // >= 20 000 EUR
)

// Per-tier budgets in cumulative tokens (input + output combined). The
// values are intentionally generous so most disputes never hit the cap;
// the cap exists to bound the worst-case cost per dispute.
const (
	aiSummaryBudgetS  = 30000
	aiSummaryBudgetM  = 40000
	aiSummaryBudgetL  = 60000
	aiSummaryBudgetXL = 90000

	aiChatBudgetS  = 20000
	aiChatBudgetM  = 25000
	aiChatBudgetL  = 40000
	aiChatBudgetXL = 60000
)

// AITier returns the budget tier this dispute belongs to.
func (d *Dispute) AITier() AITier {
	eur := d.ProposalAmount / 100
	switch {
	case eur < 500:
		return AITierS
	case eur < 5000:
		return AITierM
	case eur < 20000:
		return AITierL
	default:
		return AITierXL
	}
}

// AIBudgetSummary returns the maximum cumulative tokens (input+output)
// the AI may consume on summary generation for this dispute, including
// any manual bonus granted by an admin.
func (d *Dispute) AIBudgetSummary() int {
	base := 0
	switch d.AITier() {
	case AITierS:
		base = aiSummaryBudgetS
	case AITierM:
		base = aiSummaryBudgetM
	case AITierL:
		base = aiSummaryBudgetL
	case AITierXL:
		base = aiSummaryBudgetXL
	}
	return base + d.AIBudgetBonusTokens
}

// AIBudgetChat returns the maximum cumulative tokens (input+output) the
// AI may consume on admin chat questions for this dispute, including any
// manual bonus.
func (d *Dispute) AIBudgetChat() int {
	base := 0
	switch d.AITier() {
	case AITierS:
		base = aiChatBudgetS
	case AITierM:
		base = aiChatBudgetM
	case AITierL:
		base = aiChatBudgetL
	case AITierXL:
		base = aiChatBudgetXL
	}
	return base + d.AIBudgetBonusTokens
}

// AISummaryUsed returns the cumulative tokens consumed by AI summary calls.
func (d *Dispute) AISummaryUsed() int {
	return d.AISummaryInputTokens + d.AISummaryOutputTokens
}

// AIChatUsed returns the cumulative tokens consumed by AI chat calls.
func (d *Dispute) AIChatUsed() int {
	return d.AIChatInputTokens + d.AIChatOutputTokens
}

// AISummaryRemaining returns the tokens still available for summary calls.
// May be negative when the budget has been overshot within the +10% tolerance.
func (d *Dispute) AISummaryRemaining() int {
	return d.AIBudgetSummary() - d.AISummaryUsed()
}

// AIChatRemaining returns the tokens still available for chat calls.
func (d *Dispute) AIChatRemaining() int {
	return d.AIBudgetChat() - d.AIChatUsed()
}

// RecordAISummaryUsage adds tokens consumed by an AI summary call.
// Called after each successful Anthropic response with the actual usage
// from the API (not the estimate).
func (d *Dispute) RecordAISummaryUsage(input, output int) {
	if input < 0 {
		input = 0
	}
	if output < 0 {
		output = 0
	}
	d.AISummaryInputTokens += input
	d.AISummaryOutputTokens += output
	d.UpdatedAt = time.Now()
}

// RecordAIChatUsage adds tokens consumed by an AI chat call.
func (d *Dispute) RecordAIChatUsage(input, output int) {
	if input < 0 {
		input = 0
	}
	if output < 0 {
		output = 0
	}
	d.AIChatInputTokens += input
	d.AIChatOutputTokens += output
	d.UpdatedAt = time.Now()
}

// AddAIBudgetBonus grants extra AI budget on this dispute. Used by the
// admin "Augmenter le budget" button. The bonus applies to BOTH the
// summary and chat caps so the admin can use it wherever they need it.
func (d *Dispute) AddAIBudgetBonus(amount int) {
	if amount <= 0 {
		return
	}
	d.AIBudgetBonusTokens += amount
	d.UpdatedAt = time.Now()
}

func (d *Dispute) IsParticipant(userID uuid.UUID) bool {
	return userID == d.InitiatorID || userID == d.RespondentID
}

func (d *Dispute) CanBeCancelledBy(userID uuid.UUID) bool {
	if d.Status.IsTerminal() {
		return false
	}
	return userID == d.InitiatorID && d.RespondentFirstReplyAt == nil
}

func (d *Dispute) InitiatorRole() string {
	if d.InitiatorID == d.ClientID {
		return "client"
	}
	return "provider"
}

func classifyResolution(clientAmount, proposalAmount int64) ResolutionType {
	if clientAmount == proposalAmount {
		return ResolutionFullRefund
	}
	if clientAmount == 0 {
		return ResolutionFullRelease
	}
	return ResolutionCustom
}

// ---------------------------------------------------------------------------
// ResolveInput
// ---------------------------------------------------------------------------

type ResolveInput struct {
	ResolvedBy     uuid.UUID
	AmountClient   int64
	AmountProvider int64
	Note           string
}

// ---------------------------------------------------------------------------
// Evidence
// ---------------------------------------------------------------------------

type Evidence struct {
	ID                uuid.UUID
	DisputeID         uuid.UUID
	CounterProposalID *uuid.UUID // nil = attached to dispute opening, set = attached to a counter-proposal
	UploaderID        uuid.UUID
	Filename          string
	URL               string
	Size              int64
	MimeType          string
	CreatedAt         time.Time
}

// ---------------------------------------------------------------------------
// Counter-proposal
// ---------------------------------------------------------------------------

type CounterProposalStatus string

const (
	CPStatusPending    CounterProposalStatus = "pending"
	CPStatusAccepted   CounterProposalStatus = "accepted"
	CPStatusRejected   CounterProposalStatus = "rejected"
	CPStatusSuperseded CounterProposalStatus = "superseded"
)

type CounterProposal struct {
	ID             uuid.UUID
	DisputeID      uuid.UUID
	ProposerID     uuid.UUID
	AmountClient   int64
	AmountProvider int64
	Message        string
	Status         CounterProposalStatus
	RespondedAt    *time.Time
	CreatedAt      time.Time
}

type NewCounterProposalInput struct {
	DisputeID      uuid.UUID
	ProposerID     uuid.UUID
	AmountClient   int64
	AmountProvider int64
	ProposalAmount int64
	Message        string
}

func NewCounterProposal(in NewCounterProposalInput) (*CounterProposal, error) {
	if in.AmountClient < 0 || in.AmountProvider < 0 {
		return nil, ErrInvalidAmount
	}
	if in.AmountClient+in.AmountProvider != in.ProposalAmount {
		return nil, ErrAmountMismatch
	}
	return &CounterProposal{
		ID:             uuid.New(),
		DisputeID:      in.DisputeID,
		ProposerID:     in.ProposerID,
		AmountClient:   in.AmountClient,
		AmountProvider: in.AmountProvider,
		Message:        in.Message,
		Status:         CPStatusPending,
		CreatedAt:      time.Now(),
	}, nil
}

func (cp *CounterProposal) Accept(userID uuid.UUID) error {
	if cp.Status != CPStatusPending {
		return ErrCounterProposalNotPending
	}
	if userID == cp.ProposerID {
		return ErrCannotRespondToOwnProposal
	}
	now := time.Now()
	cp.Status = CPStatusAccepted
	cp.RespondedAt = &now
	return nil
}

func (cp *CounterProposal) Reject(userID uuid.UUID) error {
	if cp.Status != CPStatusPending {
		return ErrCounterProposalNotPending
	}
	if userID == cp.ProposerID {
		return ErrCannotRespondToOwnProposal
	}
	now := time.Now()
	cp.Status = CPStatusRejected
	cp.RespondedAt = &now
	return nil
}

func (cp *CounterProposal) Supersede() {
	if cp.Status == CPStatusPending {
		cp.Status = CPStatusSuperseded
	}
}

// MustJSON marshals v to json.RawMessage, ignoring errors (for metadata).
func MustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// ---------------------------------------------------------------------------
// AI chat message (admin Q/A history persisted per dispute)
// ---------------------------------------------------------------------------

// ChatMessageRole identifies who authored a chat turn.
type ChatMessageRole string

const (
	ChatMessageRoleUser      ChatMessageRole = "user"      // admin question
	ChatMessageRoleAssistant ChatMessageRole = "assistant" // AI answer
)

// ChatMessage represents one persisted exchange in the admin AI chat.
// Stored append-only in dispute_ai_chat_messages, read in chronological
// order to rebuild the full conversation when an admin opens the dispute.
//
// InputTokens / OutputTokens are populated only on assistant turns and
// reflect the actual API usage reported by Anthropic for that specific
// answer. They are zero on user turns (the question itself doesn't
// consume any API budget — only the request that includes it does).
type ChatMessage struct {
	ID           uuid.UUID
	DisputeID    uuid.UUID
	Role         ChatMessageRole
	Content      string
	InputTokens  int
	OutputTokens int
	CreatedAt    time.Time
}

// NewChatMessage builds a new chat message with validation. Used by the
// service layer to construct entries before persisting them.
func NewChatMessage(disputeID uuid.UUID, role ChatMessageRole, content string, inputTokens, outputTokens int) *ChatMessage {
	if inputTokens < 0 {
		inputTokens = 0
	}
	if outputTokens < 0 {
		outputTokens = 0
	}
	return &ChatMessage{
		ID:           uuid.New(),
		DisputeID:    disputeID,
		Role:         role,
		Content:      content,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		CreatedAt:    time.Now(),
	}
}
