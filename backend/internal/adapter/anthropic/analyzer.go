// Package anthropic implements the AIAnalyzer port using Claude Haiku 4.5.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	portservice "marketplace-backend/internal/port/service"
)

const (
	apiURL     = "https://api.anthropic.com/v1/messages"
	apiVersion = "2023-06-01"
	model      = "claude-haiku-4-5-20251001"
	maxTokens  = 1024
)

// Analyzer generates dispute summaries using the Anthropic Messages API.
type Analyzer struct {
	apiKey string
	client *http.Client
}

// NewAnalyzer creates an Analyzer with the given API key.
func NewAnalyzer(apiKey string) *Analyzer {
	return &Analyzer{
		apiKey: apiKey,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *Analyzer) AnalyzeDispute(ctx context.Context, input portservice.DisputeAnalysisInput) (string, error) {
	prompt := buildPrompt(input)

	reqBody := apiRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages: []apiMessage{
			{Role: "user", Content: prompt},
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic api error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response from anthropic")
	}

	return apiResp.Content[0].Text, nil
}

// ---------------------------------------------------------------------------
// Prompt builder
// ---------------------------------------------------------------------------

func buildPrompt(in portservice.DisputeAnalysisInput) string {
	var b bytes.Buffer

	b.WriteString("You are a dispute mediator assistant for a B2B marketplace. Analyze this dispute and produce a neutral summary for the admin mediator.\n\n")

	fmt.Fprintf(&b, "## Dispute\n- Reason: %s\n- Opened by: %s\n- Proposal: %s\n- Total amount: %d centimes (%.2f EUR)\n- Requested amount: %d centimes (%.2f EUR)\n\n",
		in.DisputeReason, in.InitiatorRole, in.ProposalTitle,
		in.ProposalAmount, float64(in.ProposalAmount)/100,
		in.RequestedAmount, float64(in.RequestedAmount)/100)

	fmt.Fprintf(&b, "## Initiator's description\n%s\n\n", in.DisputeDescription)

	if in.ProposalDescription != "" {
		fmt.Fprintf(&b, "## Original proposal scope\n%s\n\n", in.ProposalDescription)
	}

	if len(in.CounterProposals) > 0 {
		b.WriteString("## Counter-proposals exchanged\n")
		for i, cp := range in.CounterProposals {
			fmt.Fprintf(&b, "%d. %s proposed: %d EUR to client, %d EUR to provider — %s (status: %s)\n",
				i+1, cp.ProposerRole,
				cp.AmountClient/100, cp.AmountProvider/100,
				cp.Message, cp.Status)
		}
		b.WriteString("\n")
	}

	if len(in.Messages) > 0 {
		b.WriteString("## Recent conversation messages (last 50)\n")
		for _, m := range in.Messages {
			if m.Type == "text" || m.Type == "file" {
				fmt.Fprintf(&b, "[%s] %s (%s): %s\n", m.CreatedAt, m.SenderName, m.SenderRole, m.Content)
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("## Instructions\nProduce a structured report in French with:\n")
	b.WriteString("1. **Resume** — neutral summary of the situation (3-5 sentences)\n")
	b.WriteString("2. **Position du client** — what the client claims\n")
	b.WriteString("3. **Position du prestataire** — what the provider claims\n")
	b.WriteString("4. **Elements factuels** — key facts from the conversation (deadlines, deliverables, agreements)\n")
	b.WriteString("5. **Recommandation** — suggested resolution with amounts and justification\n\n")
	b.WriteString("Be neutral, factual, and concise. Do not take sides. Base your recommendation on the evidence available.")

	return b.String()
}

// ---------------------------------------------------------------------------
// API types
// ---------------------------------------------------------------------------

type apiRequest struct {
	Model     string       `json:"model"`
	MaxTokens int          `json:"max_tokens"`
	Messages  []apiMessage `json:"messages"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}
