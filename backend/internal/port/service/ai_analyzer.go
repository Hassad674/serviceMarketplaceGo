package service

import "context"

// AIAnalyzer generates dispute analysis reports for admin mediators.
//
// Both methods return AIUsage so the caller can record the actual token
// consumption (reported by the upstream API) on the dispute. The estimate
// embedded in budgetTokens is a rough char/4 heuristic for input only;
// the API response gives precise input + output counts.
type AIAnalyzer interface {
	// AnalyzeDispute generates the structured mediation report. Called once
	// per dispute on escalation. budgetTokens is the per-call ceiling
	// for the input prompt (output is capped separately by the adapter).
	AnalyzeDispute(ctx context.Context, input DisputeAnalysisInput, budgetTokens int) (string, AIUsage, error)

	// ChatAboutDispute answers an admin follow-up question. The full
	// dispute context is resent on every call (Claude is stateless), with
	// chat history attached so the model has the prior turns. budgetTokens
	// caps the input prompt for THIS call (cumulative dispute budget is
	// enforced one level up by the service layer).
	ChatAboutDispute(ctx context.Context, input DisputeAnalysisInput, history []ChatTurn, question string, budgetTokens int) (string, AIUsage, error)
}

// AIUsage reports the actual token counts consumed by an AI call,
// extracted from the upstream API response.
type AIUsage struct {
	InputTokens  int
	OutputTokens int
}

// ChatTurn represents one exchange in the admin chat history.
type ChatTurn struct {
	Role    string // "user" (admin question) or "assistant" (AI answer)
	Content string
}

// DisputeAnalysisInput contains all context needed for AI analysis.
type DisputeAnalysisInput struct {
	DisputeReason       string
	DisputeDescription  string
	ProposalTitle       string
	ProposalDescription string
	ProposalAmount      int64
	RequestedAmount     int64
	InitiatorRole       string // "client" or "provider"
	Messages            []ConversationMessage
	CounterProposals    []CounterProposalSummary
	Evidence            []EvidenceSummary
}

// EvidenceSummary describes a file uploaded as dispute evidence. The AI
// only sees metadata (filename, mime, size, uploader) — never the content.
type EvidenceSummary struct {
	Filename     string
	MimeType     string
	Size         int64
	UploaderRole string // "client" or "provider"
}

// ConversationMessage is a simplified message for AI context.
type ConversationMessage struct {
	SenderName string
	SenderRole string
	Content    string
	Type       string
	CreatedAt  string
}

// CounterProposalSummary is a simplified counter-proposal for AI context.
type CounterProposalSummary struct {
	ProposerRole   string
	AmountClient   int64
	AmountProvider int64
	Message        string
	Status         string
}
