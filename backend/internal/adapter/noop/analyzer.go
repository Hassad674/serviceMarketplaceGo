package noop

import (
	"context"

	portservice "marketplace-backend/internal/port/service"
)

// Analyzer is a no-op AI analyzer used when Anthropic API is not configured.
// Both methods return a placeholder text and zero token usage so callers
// (and the cost tracker) handle the disabled case identically.
type Analyzer struct{}

func NewAnalyzer() *Analyzer { return &Analyzer{} }

func (a *Analyzer) AnalyzeDispute(_ context.Context, _ portservice.DisputeAnalysisInput, _ int) (string, portservice.AIUsage, error) {
	return "Analyse IA non disponible (clé API Anthropic non configurée).", portservice.AIUsage{}, nil
}

func (a *Analyzer) ChatAboutDispute(_ context.Context, _ portservice.DisputeAnalysisInput, _ []portservice.ChatTurn, _ string, _ int) (string, portservice.AIUsage, error) {
	return "Assistant IA non disponible (clé API Anthropic non configurée).", portservice.AIUsage{}, nil
}
