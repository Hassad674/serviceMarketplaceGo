package service

import "context"

// TextModerationLabel represents a single toxicity label returned by text moderation analysis.
type TextModerationLabel struct {
	Name  string  // HATE_SPEECH, INSULT, SEXUAL, VIOLENCE_OR_THREAT, GRAPHIC, HARASSMENT_OR_ABUSE
	Score float64 // 0-1 confidence score
}

// TextModerationResult holds the outcome of text moderation analysis.
type TextModerationResult struct {
	Labels   []TextModerationLabel
	MaxScore float64
	IsSafe   bool
}

// TextModerationService analyzes text content for toxicity and policy violations.
type TextModerationService interface {
	AnalyzeText(ctx context.Context, text string) (*TextModerationResult, error)
}
