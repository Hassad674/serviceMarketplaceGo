package noop

import (
	"context"

	portservice "marketplace-backend/internal/port/service"
)

// Analyzer is a no-op AI analyzer used when Anthropic API is not configured.
type Analyzer struct{}

func NewAnalyzer() *Analyzer { return &Analyzer{} }

func (a *Analyzer) AnalyzeDispute(_ context.Context, _ portservice.DisputeAnalysisInput) (string, error) {
	return "Analyse IA non disponible (clé API Anthropic non configurée).", nil
}
