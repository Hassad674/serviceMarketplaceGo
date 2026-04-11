package dispute

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Dispute statuses
// ---------------------------------------------------------------------------

type Status string

const (
	StatusOpen        Status = "open"
	StatusNegotiation Status = "negotiation"
	StatusEscalated   Status = "escalated"
	StatusResolved    Status = "resolved"
	StatusCancelled   Status = "cancelled"
)

func (s Status) IsTerminal() bool {
	return s == StatusResolved || s == StatusCancelled
}

// ---------------------------------------------------------------------------
// Dispute reasons (role-specific)
// ---------------------------------------------------------------------------

type Reason string

const (
	ReasonWorkNotConforming  Reason = "work_not_conforming"
	ReasonNonDelivery        Reason = "non_delivery"
	ReasonInsufficientQuality Reason = "insufficient_quality"
	ReasonClientGhosting     Reason = "client_ghosting"
	ReasonScopeCreep         Reason = "scope_creep"
	ReasonRefusalToValidate  Reason = "refusal_to_validate"
	ReasonHarassment         Reason = "harassment"
	ReasonOther              Reason = "other"
)

var clientReasons = map[Reason]bool{
	ReasonWorkNotConforming:   true,
	ReasonNonDelivery:         true,
	ReasonInsufficientQuality: true,
	ReasonOther:               true,
}

var providerReasons = map[Reason]bool{
	ReasonClientGhosting:    true,
	ReasonScopeCreep:        true,
	ReasonRefusalToValidate: true,
	ReasonHarassment:        true,
	ReasonOther:             true,
}

// IsValidForRole checks whether the reason is allowed for the given role.
// role is "client" or "provider".
func (r Reason) IsValidForRole(role string) bool {
	if role == "client" {
		return clientReasons[r]
	}
	return providerReasons[r]
}

// ---------------------------------------------------------------------------
// Resolution type
// ---------------------------------------------------------------------------

type ResolutionType string

const (
	ResolutionFullRefund    ResolutionType = "full_refund"
	ResolutionPartialRefund ResolutionType = "partial_refund"
	ResolutionFullRelease   ResolutionType = "full_release"
	ResolutionCustom        ResolutionType = "custom"
)

// ---------------------------------------------------------------------------
// Dispute entity
// ---------------------------------------------------------------------------

type Dispute struct {
	ID             uuid.UUID
	ProposalID     uuid.UUID
	ConversationID uuid.UUID
	InitiatorID    uuid.UUID
	RespondentID   uuid.UUID
	ClientID       uuid.UUID
	ProviderID     uuid.UUID

	// Denormalized org anchors (R3 extended): the client's and
	// provider's current organization at the moment the dispute was
	// opened. Used to scope ListByOrganization so every operator of
	// either org sees the dispute in their list.
	ClientOrganizationID   uuid.UUID
	ProviderOrganizationID uuid.UUID

	Reason          Reason
	Description     string
	RequestedAmount int64
	ProposalAmount  int64

	Status Status

	ResolutionType          *ResolutionType
	ResolutionAmountClient  *int64
	ResolutionAmountProvider *int64
	ResolvedBy              *uuid.UUID
	ResolutionNote          *string
	AISummary               *string

	EscalatedAt            *time.Time
	ResolvedAt             *time.Time
	CancelledAt            *time.Time
	LastActivityAt         time.Time
	RespondentFirstReplyAt *time.Time

	// Cancellation request: set when the initiator asks to cancel after the
	// respondent has already replied. The respondent must accept or refuse.
	CancellationRequestedBy *uuid.UUID
	CancellationRequestedAt *time.Time

	// AI budget tracking — cumulative across the dispute lifetime. Summary
	// and chat tokens are tracked separately so the admin UI can show
	// distinct progress bars per category. AIBudgetBonusTokens grows each
	// time the admin clicks "Augmenter le budget" and is added to BOTH
	// the summary and chat caps (whichever the admin needs more of).
	AISummaryInputTokens  int
	AISummaryOutputTokens int
	AIChatInputTokens     int
	AIChatOutputTokens    int
	AIBudgetBonusTokens   int

	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

type NewDisputeInput struct {
	ProposalID             uuid.UUID
	ConversationID         uuid.UUID
	InitiatorID            uuid.UUID
	RespondentID           uuid.UUID
	ClientID               uuid.UUID
	ProviderID             uuid.UUID
	ClientOrganizationID   uuid.UUID
	ProviderOrganizationID uuid.UUID
	Reason                 Reason
	Description            string
	RequestedAmount        int64
	ProposalAmount         int64
}

func NewDispute(in NewDisputeInput) (*Dispute, error) {
	if len(in.Description) > 5000 {
		return nil, ErrDescriptionTooLong
	}
	if in.RequestedAmount <= 0 || in.RequestedAmount > in.ProposalAmount {
		return nil, ErrInvalidAmount
	}

	role := "provider"
	if in.InitiatorID == in.ClientID {
		role = "client"
	}
	if !in.Reason.IsValidForRole(role) {
		return nil, ErrInvalidReason
	}

	now := time.Now()
	return &Dispute{
		ID:                     uuid.New(),
		ProposalID:             in.ProposalID,
		ConversationID:         in.ConversationID,
		InitiatorID:            in.InitiatorID,
		RespondentID:           in.RespondentID,
		ClientID:               in.ClientID,
		ProviderID:             in.ProviderID,
		ClientOrganizationID:   in.ClientOrganizationID,
		ProviderOrganizationID: in.ProviderOrganizationID,
		Reason:                 in.Reason,
		Description:            in.Description,
		RequestedAmount:        in.RequestedAmount,
		ProposalAmount:         in.ProposalAmount,
		Status:                 StatusOpen,
		LastActivityAt:         now,
		Version:                1,
		CreatedAt:              now,
		UpdatedAt:              now,
	}, nil
}

// ---------------------------------------------------------------------------
// State machine
// ---------------------------------------------------------------------------

func (d *Dispute) MarkNegotiation() error {
	if d.Status != StatusOpen {
		return ErrInvalidStatus
	}
	d.Status = StatusNegotiation
	d.UpdatedAt = time.Now()
	return nil
}

func (d *Dispute) Escalate() error {
	if d.Status != StatusOpen && d.Status != StatusNegotiation {
		return ErrInvalidStatus
	}
	now := time.Now()
	d.Status = StatusEscalated
	d.EscalatedAt = &now
	d.UpdatedAt = now
	return nil
}

func (d *Dispute) Resolve(in ResolveInput) error {
	if d.Status != StatusEscalated && d.Status != StatusOpen && d.Status != StatusNegotiation {
		return ErrInvalidStatus
	}
	if in.AmountClient+in.AmountProvider != d.ProposalAmount {
		return ErrAmountMismatch
	}
	now := time.Now()
	d.Status = StatusResolved
	rt := classifyResolution(in.AmountClient, d.ProposalAmount)
	d.ResolutionType = &rt
	d.ResolutionAmountClient = &in.AmountClient
	d.ResolutionAmountProvider = &in.AmountProvider
	if in.ResolvedBy != uuid.Nil {
		d.ResolvedBy = &in.ResolvedBy
	}
	if in.Note != "" {
		d.ResolutionNote = &in.Note
	}
	d.ResolvedAt = &now
	d.UpdatedAt = now
	return nil
}

func (d *Dispute) AutoResolveForInitiator() error {
	if d.Status != StatusOpen {
		return ErrInvalidStatus
	}
	// Initiator gets what they asked for
	var clientAmt, providerAmt int64
	if d.InitiatorID == d.ClientID {
		clientAmt = d.RequestedAmount
		providerAmt = d.ProposalAmount - d.RequestedAmount
	} else {
		providerAmt = d.RequestedAmount
		clientAmt = d.ProposalAmount - d.RequestedAmount
	}
	return d.Resolve(ResolveInput{
		AmountClient:  clientAmt,
		AmountProvider: providerAmt,
		Note:          "Auto-resolved: respondent did not reply within 7 days.",
	})
}

// Cancel attempts to cancel a dispute on behalf of one of its participants.
//
// The path taken depends on who is asking and whether the respondent has
// already engaged with the dispute:
//
//   - Initiator + respondent has NOT yet replied → direct cancellation.
//     The initiator may freely retract the dispute as long as the other side
//     has not invested any effort.
//
//   - Initiator + respondent HAS replied → creates a cancellation request.
//     The respondent now has a stake and must explicitly consent.
//
//   - Respondent (non-initiator), at any point → ALWAYS creates a
//     cancellation request, never a direct cancellation. The respondent
//     never had the unilateral right to terminate a dispute they did not
//     open; they can only ask the initiator for permission to cancel.
//
// Returns (true, nil) when the dispute was cancelled directly,
// or (false, nil) when a cancellation request was created.
func (d *Dispute) Cancel(userID uuid.UUID) (cancelled bool, err error) {
	// Cancellation is allowed all the way through admin mediation: as long
	// as the admin has not rendered a final decision, the parties can still
	// reach an amicable agreement (whichever comes first wins).
	if d.Status != StatusOpen && d.Status != StatusNegotiation && d.Status != StatusEscalated {
		return false, ErrInvalidStatus
	}
	if !d.IsParticipant(userID) {
		return false, ErrNotParticipant
	}

	// Direct cancellation path — reserved to the initiator and only valid
	// while the respondent has not yet engaged.
	if userID == d.InitiatorID && d.RespondentFirstReplyAt == nil {
		now := time.Now()
		d.Status = StatusCancelled
		d.CancelledAt = &now
		d.UpdatedAt = now
		return true, nil
	}

	// All other cases (initiator after a reply, or respondent at any time)
	// must go through a cancellation request that the OTHER party accepts.
	if d.CancellationRequestedBy != nil {
		return false, ErrCancellationAlreadyRequested
	}
	now := time.Now()
	d.CancellationRequestedBy = &userID
	d.CancellationRequestedAt = &now
	d.UpdatedAt = now
	return false, nil
}

// RespondToCancellationRequest processes the other party's decision on a
// pending cancellation request. Either the initiator or the respondent may
// be the requester (since both can ask), so the only invariant is that the
// requester themselves cannot self-accept their own request — only the
// OTHER participant can.
// If accepted, the dispute is cancelled; if refused, the request is cleared.
func (d *Dispute) RespondToCancellationRequest(userID uuid.UUID, accept bool) error {
	if d.CancellationRequestedBy == nil {
		return ErrNoCancellationPending
	}
	if !d.IsParticipant(userID) {
		return ErrNotParticipant
	}
	if userID == *d.CancellationRequestedBy {
		return ErrNotAuthorized
	}

	now := time.Now()
	if accept {
		d.Status = StatusCancelled
		d.CancelledAt = &now
	} else {
		d.CancellationRequestedBy = nil
		d.CancellationRequestedAt = nil
	}
	d.UpdatedAt = now
	return nil
}

// ClearCancellationRequest removes any pending cancellation request.
// Called implicitly when the dispute state changes (e.g. a counter-proposal
// signals that negotiation is still active).
func (d *Dispute) ClearCancellationRequest() {
	if d.CancellationRequestedBy != nil {
		d.CancellationRequestedBy = nil
		d.CancellationRequestedAt = nil
		d.UpdatedAt = time.Now()
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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
