package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	portservice "marketplace-backend/internal/port/service"
)

type RequirementsInfo struct {
	HasRequirements bool               `json:"has_requirements"`
	CurrentlyDue    []string           `json:"currently_due"`
	Labels          []RequirementLabel `json:"labels"`
}

type RequirementLabel struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// Requirement code to human-readable label mapping
var requirementLabels = map[string]map[string]string{
	"company.verification.document":                {"en": "Company document", "fr": "Document de la société"},
	"individual.verification.document":             {"en": "Identity document", "fr": "Pièce d'identité"},
	"individual.verification.additional_document":  {"en": "Additional identity document", "fr": "Justificatif d'identité supplémentaire"},
	"representative.verification.document":         {"en": "Representative identity document", "fr": "Pièce d'identité du représentant"},
	"representative.verification.additional_document": {"en": "Representative additional document", "fr": "Justificatif supplémentaire du représentant"},
}

func translateRequirement(code, lang string) string {
	if labels, ok := requirementLabels[code]; ok {
		if label, ok := labels[lang]; ok {
			return label
		}
		return labels["en"]
	}
	if lang == "fr" {
		return "Information complémentaire requise"
	}
	return "Additional information required"
}

// GetRequirements returns Stripe requirements for the user's connected account.
func (s *Service) GetRequirements(ctx context.Context, userID uuid.UUID, lang string) (*RequirementsInfo, error) {
	info, err := s.payments.GetByUserID(ctx, userID)
	if err != nil || info.StripeAccountID == "" {
		return &RequirementsInfo{}, nil
	}

	if s.stripe == nil {
		return &RequirementsInfo{}, nil
	}

	due, err := s.stripe.GetAccountRequirements(ctx, info.StripeAccountID)
	if err != nil {
		return nil, fmt.Errorf("get requirements: %w", err)
	}

	result := &RequirementsInfo{
		HasRequirements: len(due) > 0,
		CurrentlyDue:    due,
	}

	for _, code := range due {
		result.Labels = append(result.Labels, RequirementLabel{
			Code:  code,
			Label: translateRequirement(code, lang),
		})
	}

	return result, nil
}

// CreateAccountLink generates a Stripe-hosted link for the provider to complete requirements.
func (s *Service) CreateAccountLink(ctx context.Context, userID uuid.UUID) (string, error) {
	info, err := s.payments.GetByUserID(ctx, userID)
	if err != nil || info.StripeAccountID == "" {
		return "", fmt.Errorf("no stripe account")
	}

	if s.stripe == nil {
		return "", fmt.Errorf("stripe not configured")
	}

	returnURL := s.frontendURL + "/payment-info?stripe_return=true"
	refreshURL := s.frontendURL + "/payment-info?stripe_refresh=true"

	return s.stripe.CreateAccountLink(ctx, info.StripeAccountID, returnURL, refreshURL)
}

// NotifyNewRequirements sends a notification when Stripe requires new information.
func (s *Service) NotifyNewRequirements(ctx context.Context, userID uuid.UUID, requirements []string) {
	if s.notifications == nil || len(requirements) == 0 {
		return
	}

	data, _ := json.Marshal(map[string]string{
		"type": "stripe_requirements",
		"url":  "/payment-info",
	})

	if err := s.notifications.Send(ctx, portservice.NotificationInput{
		UserID: userID,
		Type:   "stripe_requirements",
		Title:  "Action requise — Stripe",
		Body:   "Stripe demande des informations complémentaires pour activer votre compte.",
		Data:   data,
	}); err != nil {
		slog.Error("failed to send stripe requirements notification", "user_id", userID, "error", err)
	}
}
