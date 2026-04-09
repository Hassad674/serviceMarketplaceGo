package service

import "context"

// AIAnalyzer generates dispute analysis reports for admin mediators.
type AIAnalyzer interface {
	AnalyzeDispute(ctx context.Context, input DisputeAnalysisInput) (string, error)
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
