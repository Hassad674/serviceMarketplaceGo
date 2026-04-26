package openai

import (
	"context"
	"fmt"

	portservice "marketplace-backend/internal/port/service"
)

// defaultModel is the OpenAI content-safety model we call. omni-moderation
// superseded text-moderation-latest in Sept 2024 with two key wins for
// this project: (1) native multilingual support (French coverage is on
// par with English, crucial given our main user base) and (2) 13 fine-
// grained categories — notably sexual/minors, harassment/threatening and
// hate/threatening — that let domain/moderation apply zero-tolerance
// rules the old model could not express.
const defaultModel = "omni-moderation-latest"

// maxInputChars is the per-request text cap enforced by OpenAI for the
// moderation endpoint. We truncate client-side so a long review or
// message never trips a 400 error — the start of the text is usually
// enough for toxicity signals; a truncated novel with a buried threat
// is an edge case we accept trading away.
const maxInputChars = 32_000

// TextModerationService implements port/service.TextModerationService
// using the OpenAI /v1/moderations endpoint. It is the default text
// moderation backend since migration away from AWS Comprehend — see
// config.TextModerationProvider for how to switch providers.
type TextModerationService struct {
	client *Client
	model  string
}

// NewTextModerationService wires a real client pointing at OpenAI. Pass
// the project's OPENAI_API_KEY. Tests build the Service directly so
// they can inject an httptest.Server — see text_moderation_test.go.
func NewTextModerationService(apiKey string) *TextModerationService {
	return &TextModerationService{
		client: NewClient(apiKey, ""),
		model:  defaultModel,
	}
}

// moderationRequest matches the /v1/moderations payload. input is a
// single string here; the endpoint also accepts an array of up to 32
// strings for batching, which we'll add when volume requires it.
type moderationRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// moderationResponse is a trimmed version of OpenAI's response schema.
// We only need per-category scores — flagged booleans are ignored
// because domain/moderation applies its own thresholds.
type moderationResponse struct {
	Results []struct {
		CategoryScores map[string]float64 `json:"category_scores"`
	} `json:"results"`
}

// AnalyzeText sends text to OpenAI and maps the response into the
// generic TextModerationResult that domain/moderation.DecideStatus
// consumes. Empty input short-circuits with a clean result — calling
// the API for nothing is pointless and the endpoint rejects empty
// strings anyway.
func (s *TextModerationService) AnalyzeText(
	ctx context.Context,
	text string,
) (*portservice.TextModerationResult, error) {
	if text == "" {
		return &portservice.TextModerationResult{IsSafe: true}, nil
	}
	if len(text) > maxInputChars {
		text = text[:maxInputChars]
	}

	var resp moderationResponse
	err := s.client.postJSON(ctx, "/v1/moderations", moderationRequest{
		Model: s.model,
		Input: text,
	}, &resp)
	if err != nil {
		return nil, fmt.Errorf("openai: moderate text: %w", err)
	}

	return mapModerationResponse(&resp), nil
}

// mapModerationResponse flattens the nested response into the simpler
// port-level result. OpenAI returns one Results entry per input; with
// a single input we only look at Results[0]. MaxScore is the highest
// per-category score, which preserves enough signal for DecideStatus
// to act on the zero-tolerance matrix AND the global hide/flag
// thresholds.
func mapModerationResponse(resp *moderationResponse) *portservice.TextModerationResult {
	if resp == nil || len(resp.Results) == 0 {
		return &portservice.TextModerationResult{IsSafe: true}
	}

	scores := resp.Results[0].CategoryScores
	labels := make([]portservice.TextModerationLabel, 0, len(scores))
	var maxScore float64
	for name, score := range scores {
		labels = append(labels, portservice.TextModerationLabel{Name: name, Score: score})
		if score > maxScore {
			maxScore = score
		}
	}

	return &portservice.TextModerationResult{
		Labels:   labels,
		MaxScore: maxScore,
		IsSafe:   maxScore < 0.5,
	}
}
