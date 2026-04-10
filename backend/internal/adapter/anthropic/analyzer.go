// Package anthropic implements the AIAnalyzer port using Claude Haiku 4.5.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	portservice "marketplace-backend/internal/port/service"
)

const (
	apiURL     = "https://api.anthropic.com/v1/messages"
	apiVersion = "2023-06-01"
	model      = "claude-haiku-4-5-20251001"

	// Output caps per call type. Reserved BEFORE filling the input so the
	// AI is guaranteed enough room to deliver a complete answer even if
	// the input had to be truncated to fit.
	maxTokensSummary = 2000 // structured 5-section report
	maxTokensChat    = 800  // concise admin Q&A reply

	// charsPerToken is a rough estimate (1 token ≈ 4 characters in EN/FR).
	// Used for cheap input-budget enforcement before calling the API.
	// Margin of error is ~10% but largely sufficient to bound cost.
	charsPerToken = 4

	// chatMaxRecentMessages caps the number of conversation messages
	// included in chat call prompts. Smaller than the summary call (which
	// includes up to 200) because the chat doesn't need full history,
	// just enough context for the admin's question.
	chatMaxRecentMessages = 50
)

// Analyzer generates dispute summaries using the Anthropic Messages API.
//
// Token budgets are passed per-call (not stored on the struct) because
// they vary with the dispute's tier. The adapter is otherwise stateless.
type Analyzer struct {
	apiKey string
	client *http.Client
}

// NewAnalyzer creates an Analyzer bound to the given Anthropic API key.
func NewAnalyzer(apiKey string) *Analyzer {
	return &Analyzer{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *Analyzer) AnalyzeDispute(ctx context.Context, input portservice.DisputeAnalysisInput, budgetTokens int) (string, portservice.AIUsage, error) {
	// Reserve maxTokensSummary for the response so the input budget never
	// starves Claude of room to write. The remainder is what we have for
	// the prompt itself.
	inputBudget := budgetTokens - maxTokensSummary
	if inputBudget < 1000 {
		inputBudget = 1000 // floor: even tiny budgets get at least the irreducible context
	}
	trimmed := applyInputBudget(input, inputBudget, buildPrompt)
	prompt := buildPrompt(trimmed)

	text, usage, err := a.callMessages(ctx, prompt, maxTokensSummary)
	if err != nil {
		return "", portservice.AIUsage{}, err
	}

	slog.Info("anthropic dispute summary",
		"input_tokens", usage.InputTokens,
		"output_tokens", usage.OutputTokens,
		"messages_kept", len(trimmed.Messages),
		"messages_dropped", len(input.Messages)-len(trimmed.Messages),
		"budget_tokens", budgetTokens,
	)
	return text, usage, nil
}

func (a *Analyzer) ChatAboutDispute(ctx context.Context, input portservice.DisputeAnalysisInput, history []portservice.ChatTurn, question string, budgetTokens int) (string, portservice.AIUsage, error) {
	// Compress the context for chat: drop messages beyond the most recent
	// chatMaxRecentMessages BEFORE applying the budget. This frees room
	// for the chat history without sacrificing the dispute structure.
	compressed := compressForChat(input)

	// Reserve maxTokensChat for the response, then enforce budget on input.
	inputBudget := budgetTokens - maxTokensChat
	if inputBudget < 1000 {
		inputBudget = 1000
	}
	builder := func(in portservice.DisputeAnalysisInput) string {
		return buildChatPrompt(in, history, question)
	}
	trimmed := applyInputBudget(compressed, inputBudget, builder)
	prompt := builder(trimmed)

	text, usage, err := a.callMessages(ctx, prompt, maxTokensChat)
	if err != nil {
		return "", portservice.AIUsage{}, err
	}

	slog.Info("anthropic dispute chat",
		"input_tokens", usage.InputTokens,
		"output_tokens", usage.OutputTokens,
		"history_turns", len(history),
		"messages_kept", len(trimmed.Messages),
		"budget_tokens", budgetTokens,
	)
	return text, usage, nil
}

// callMessages performs the actual HTTP call to the Anthropic Messages API
// and parses the response. Shared between summary and chat methods so the
// retry/timeout/error handling is identical.
func (a *Analyzer) callMessages(ctx context.Context, prompt string, maxOutputTokens int) (string, portservice.AIUsage, error) {
	reqBody := apiRequest{
		Model:     model,
		MaxTokens: maxOutputTokens,
		Messages: []apiMessage{
			{Role: "user", Content: prompt},
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", portservice.AIUsage{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return "", portservice.AIUsage{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", portservice.AIUsage{}, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", portservice.AIUsage{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", portservice.AIUsage{}, fmt.Errorf("anthropic api error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", portservice.AIUsage{}, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(apiResp.Content) == 0 {
		return "", portservice.AIUsage{}, fmt.Errorf("empty response from anthropic")
	}

	return apiResp.Content[0].Text, portservice.AIUsage{
		InputTokens:  apiResp.Usage.InputTokens,
		OutputTokens: apiResp.Usage.OutputTokens,
	}, nil
}

// applyInputBudget enforces a hard cap on the prompt size by dropping the
// oldest conversation messages first. The dispute metadata, description,
// scope, counter-proposals and evidence list are NEVER truncated — they
// are the irreducible context the AI needs to make a decision.
//
// The promptBuilder param lets this helper be reused for both the summary
// prompt builder and the chat prompt builder.
//
// When file content reading lands in Phase 3, this function will be
// extended to also truncate file bodies (largest first) while keeping
// the file metadata list intact.
func applyInputBudget(input portservice.DisputeAnalysisInput, budgetTokens int, promptBuilder func(portservice.DisputeAnalysisInput) string) portservice.DisputeAnalysisInput {
	if budgetTokens <= 0 {
		return input // no budget enforced
	}

	estimate := func(in portservice.DisputeAnalysisInput) int {
		return len(promptBuilder(in)) / charsPerToken
	}

	if estimate(input) <= budgetTokens {
		return input
	}

	// Drop oldest messages one by one until we fit. Messages are already
	// sorted chronologically, so trimming from the front drops the oldest.
	trimmed := input
	for len(trimmed.Messages) > 0 && estimate(trimmed) > budgetTokens {
		trimmed.Messages = trimmed.Messages[1:]
	}

	if estimate(trimmed) > budgetTokens {
		slog.Warn("anthropic: prompt still over budget after dropping all conversation messages",
			"budget_tokens", budgetTokens,
			"estimated_tokens", estimate(trimmed),
		)
	} else if len(trimmed.Messages) < len(input.Messages) {
		slog.Warn("anthropic: input prompt truncated to fit budget",
			"budget_tokens", budgetTokens,
			"messages_kept", len(trimmed.Messages),
			"messages_dropped", len(input.Messages)-len(trimmed.Messages),
		)
	}

	return trimmed
}

// compressForChat returns a copy of the dispute input with the message
// list capped to the most recent chatMaxRecentMessages entries. The chat
// doesn't need the full conversation history — only enough context for
// the current question.
func compressForChat(input portservice.DisputeAnalysisInput) portservice.DisputeAnalysisInput {
	out := input
	if len(out.Messages) > chatMaxRecentMessages {
		out.Messages = out.Messages[len(out.Messages)-chatMaxRecentMessages:]
	}
	return out
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
		b.WriteString("## Conversation messages exchanged after the mission started\n")
		b.WriteString("(Only messages posted after the proposal was paid and the mission became active. Earlier messages — proposal negotiation — are NOT included by default and can be requested separately if needed.)\n\n")
		for _, m := range in.Messages {
			if m.Type == "text" || m.Type == "file" {
				fmt.Fprintf(&b, "[%s] %s (%s): %s\n", m.CreatedAt, m.SenderName, m.SenderRole, m.Content)
			}
		}
		b.WriteString("\n")
	}

	if len(in.Evidence) > 0 {
		b.WriteString("## Evidence files attached to the dispute\n")
		b.WriteString("(File metadata only — you cannot read the file content. Reference these by name when relevant so the admin knows to open them.)\n")
		for i, e := range in.Evidence {
			fmt.Fprintf(&b, "%d. %q (uploaded by %s, %s, %.1f KB)\n",
				i+1, e.Filename, e.UploaderRole, e.MimeType, float64(e.Size)/1024)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Instructions\nProduce a structured report in French with EXACTLY these 5 sections (use the markdown headings as written):\n\n")
	b.WriteString("**Resume**\n3-5 sentences neutral summary of the situation.\n\n")
	b.WriteString("**Position du client**\nWhat the client claims, in their own framing.\n\n")
	b.WriteString("**Position du prestataire**\nWhat the provider claims, in their own framing.\n\n")
	b.WriteString("**Elements factuels**\nKey facts extracted from the conversation, the proposal description and the evidence files (deadlines mentioned, deliverables sent, explicit agreements, missed commitments, etc.). Reference evidence files by their exact filename when relevant.\n\n")
	b.WriteString("**Recommandation finale**\nProvide a clear, executable decision for the admin in the EXACT following format:\n\n")
	fmt.Fprintf(&b, "  > **Repartition recommandee** : Client X%% (%.2f EUR sur %d EUR) / Prestataire Y%% (%.2f EUR sur %d EUR)\n",
		float64(in.ProposalAmount)/100, in.ProposalAmount/100,
		float64(in.ProposalAmount)/100, in.ProposalAmount/100)
	fmt.Fprintf(&b, "  > Where X + Y = 100, computed against the total mission amount of %d centimes (%.2f EUR).\n\n",
		in.ProposalAmount, float64(in.ProposalAmount)/100)
	b.WriteString("Then 2-3 short sentences of justification, based STRICTLY on the documented facts. Do not invent or assume anything that is not in the context above.\n\n")
	b.WriteString("If you genuinely hesitate between two splits, give the primary recommendation in the format above and mention the alternative in one sentence after the justification.\n\n")
	b.WriteString("## Constraints\n- Be neutral and factual. Do not take sides emotionally.\n- Only base your reasoning on the evidence present in the context above.\n- If you do not know something, say \"information non disponible\" rather than guessing.\n- Always reference evidence files by their exact filename (in quotes) so the admin knows where to verify.\n- You operate under a strict token budget. Your final report MUST fit in approximately 1800 tokens. Prioritize delivering all 5 sections in concise form rather than detailed sections that get truncated mid-thought. The \"Recommandation finale\" block is the most critical — it must always be present and complete, never sacrificed for length.")

	return b.String()
}

// buildChatPrompt builds the prompt for an admin chat question. It re-uses
// the dispute context from buildPrompt minus the "produce a structured
// report" instructions, then appends the chat history (if any) and the
// new admin question. Claude is stateless so the full context is resent
// on every chat turn.
func buildChatPrompt(in portservice.DisputeAnalysisInput, history []portservice.ChatTurn, question string) string {
	var b bytes.Buffer

	b.WriteString("You are a dispute mediator assistant for a B2B marketplace. You are answering follow-up questions from the admin mediator about an escalated dispute. Be precise, neutral, and base every claim on the documented context below.\n\n")

	// --- Same context structure as the summary, kept in sync intentionally ---
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
		b.WriteString("## Recent conversation messages (after the mission started)\n")
		for _, m := range in.Messages {
			if m.Type == "text" || m.Type == "file" {
				fmt.Fprintf(&b, "[%s] %s (%s): %s\n", m.CreatedAt, m.SenderName, m.SenderRole, m.Content)
			}
		}
		b.WriteString("\n")
	}

	if len(in.Evidence) > 0 {
		b.WriteString("## Evidence files attached to the dispute\n")
		b.WriteString("(File metadata only — you cannot read the file content. Reference these by name when relevant so the admin knows to open them.)\n")
		for i, e := range in.Evidence {
			fmt.Fprintf(&b, "%d. %q (uploaded by %s, %s, %.1f KB)\n",
				i+1, e.Filename, e.UploaderRole, e.MimeType, float64(e.Size)/1024)
		}
		b.WriteString("\n")
	}

	// --- Chat history (previous Q/A turns from this admin session) ---
	if len(history) > 0 {
		b.WriteString("## Previous chat turns in this admin session\n")
		for _, t := range history {
			label := "Admin"
			if t.Role == "assistant" {
				label = "Assistant"
			}
			fmt.Fprintf(&b, "%s: %s\n\n", label, t.Content)
		}
	}

	// --- The new question + answering instructions ---
	b.WriteString("## Current admin question\n")
	b.WriteString(question)
	b.WriteString("\n\n")

	b.WriteString("## Instructions\n")
	b.WriteString("- Answer concisely and directly. Your response must fit in approximately 600 tokens.\n")
	b.WriteString("- Base every claim strictly on the documented context above. Do not invent facts.\n")
	b.WriteString("- Reference evidence files by their exact filename in quotes when relevant.\n")
	b.WriteString("- If the question requires information not present in the context, say \"information non disponible dans le contexte\" briefly and suggest where the admin should look (which file, which message date, which counter-proposal).\n")
	b.WriteString("- Stay neutral. You are an assistant, not a judge — the admin makes the final decision.\n")
	b.WriteString("- Respond in French.")

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
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
