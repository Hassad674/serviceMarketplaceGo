package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	portservice "marketplace-backend/internal/port/service"
)

// RequirementsInfo holds formatted field sections for Stripe requirements.
type RequirementsInfo struct {
	HasRequirements     bool                      `json:"has_requirements"`
	Sections            []FieldSection             `json:"sections"`
	CurrentDeadline     *int64                     `json:"current_deadline,omitempty"`
	PendingVerification []string                   `json:"pending_verification,omitempty"`
	Errors              []domain.RequirementError  `json:"errors,omitempty"`
}

// GetRequirements returns Stripe requirements as formatted FieldSections.
// The sections use the same format as GetCountryFields, so the frontend
// can render them with the same DynamicSection component.
func (s *Service) GetRequirements(ctx context.Context, userID uuid.UUID) (*RequirementsInfo, error) {
	info, err := s.payments.GetByUserID(ctx, userID)
	if err != nil || info.StripeAccountID == "" {
		return &RequirementsInfo{}, nil
	}

	if s.stripe == nil {
		return &RequirementsInfo{}, nil
	}

	reqs, err := s.stripe.GetAccountRequirements(ctx, info.StripeAccountID)
	if err != nil {
		return nil, fmt.Errorf("get requirements: %w", err)
	}

	if len(reqs.CurrentlyDue) == 0 && len(reqs.EventuallyDue) == 0 && len(reqs.PastDue) == 0 {
		return &RequirementsInfo{Sections: []FieldSection{}}, nil
	}

	sections := buildRequirementSections(reqs, info.Country)
	result := &RequirementsInfo{
		HasRequirements: len(sections) > 0,
		Sections:        sections,
	}
	if reqs.CurrentDeadline != 0 {
		result.CurrentDeadline = &reqs.CurrentDeadline
	}
	if len(reqs.PendingVerification) > 0 {
		result.PendingVerification = reqs.PendingVerification
	}
	if len(reqs.Errors) > 0 {
		result.Errors = reqs.Errors
	}
	return result, nil
}

// urgencyPriority defines the priority of each urgency level (higher = more urgent).
var urgencyPriority = map[string]int{
	"past_due":       3,
	"currently_due":  2,
	"eventually_due": 1,
}

// buildRequirementSections converts Stripe requirement paths into FieldSections
// with urgency tagging and deduplication (highest urgency wins).
func buildRequirementSections(reqs *domain.AccountRequirements, country string) []FieldSection {
	// fieldUrgency tracks the highest urgency seen for each unique field path.
	fieldUrgency := make(map[string]string)
	tagUrgency := func(paths []string, urgency string) {
		for _, path := range paths {
			existing, ok := fieldUrgency[path]
			if !ok || urgencyPriority[urgency] > urgencyPriority[existing] {
				fieldUrgency[path] = urgency
			}
		}
	}
	tagUrgency(reqs.EventuallyDue, "eventually_due")
	tagUrgency(reqs.CurrentlyDue, "currently_due")
	tagUrgency(reqs.PastDue, "past_due")

	sectionMap := make(map[string][]FieldSpec)
	seen := make(dateSeen)
	hasBankRequirement := false

	for path, urgency := range fieldUrgency {
		if isExternalAccountPath(path) {
			hasBankRequirement = true
			continue
		}
		if domain.IsAutoHandled(path) {
			continue
		}
		processRequirementFieldWithUrgency(path, urgency, sectionMap, seen)
	}

	sections := buildSections(sectionMap)

	if hasBankRequirement {
		sections = appendBankSectionToSlice(sections, country)
	}

	return sections
}

// processRequirementFieldWithUrgency handles a single Stripe requirement path
// and tags the resulting FieldSpec with the given urgency level.
func processRequirementFieldWithUrgency(
	path, urgency string, sectionMap map[string][]FieldSpec, seen dateSeen,
) {
	entity := domain.EntityFromPath(path)

	// Document upload fields
	if domain.IsDocumentUploadField(path) {
		sectionMap[entity] = append(sectionMap[entity], FieldSpec{
			Path:     path,
			Key:      path,
			Type:     "document_upload",
			LabelKey: domain.FieldLabelKey(path),
			Required: true,
			IsExtra:  false,
			Urgency:  urgency,
		})
		return
	}

	// Collapse dob components into a single date field per entity
	if domain.IsDOBComponent(path) {
		key := entity + ".dob"
		if seen[key] {
			return
		}
		seen[key] = true
		_, isExtra := domain.MapStripeField(path)
		sectionMap[entity] = append(sectionMap[entity], FieldSpec{
			Path:     key,
			Key:      key,
			Type:     "date",
			LabelKey: "dateOfBirth",
			Required: true,
			IsExtra:  isExtra,
			Urgency:  urgency,
		})
		return
	}

	// Collapse registration_date components into single date field
	if domain.IsRegistrationDateComponent(path) {
		key := entity + ".registration_date"
		if seen[key] {
			return
		}
		seen[key] = true
		sectionMap[entity] = append(sectionMap[entity], FieldSpec{
			Path:     key,
			Key:      key,
			Type:     "date",
			LabelKey: "registrationDate",
			Required: true,
			IsExtra:  true,
			Urgency:  urgency,
		})
		return
	}

	_, isExtra := domain.MapStripeField(path)
	sectionMap[entity] = append(sectionMap[entity], FieldSpec{
		Path:        path,
		Key:         path,
		Type:        domain.FieldInputType(path),
		LabelKey:    domain.FieldLabelKey(path),
		Required:    true,
		IsExtra:     isExtra,
		Placeholder: domain.FieldPlaceholder(path),
		Urgency:     urgency,
	})
}

// isExternalAccountPath returns true for external_account requirement paths.
func isExternalAccountPath(path string) bool {
	return path == "external_account" || strings.HasPrefix(path, "external_account.")
}

// appendBankSectionToSlice adds a bank section to the given sections slice.
func appendBankSectionToSlice(sections []FieldSection, country string) []FieldSection {
	isIBAN := ibanCountries[strings.ToUpper(country)]
	var bankFields []FieldSpec

	if isIBAN {
		bankFields = []FieldSpec{
			{Path: "bank.iban", Key: "bank.iban", Type: "text", LabelKey: "iban", Required: true},
			{Path: "bank.bic", Key: "bank.bic", Type: "text", LabelKey: "bic", Required: false},
		}
	} else {
		bankFields = []FieldSpec{
			{Path: "bank.account_number", Key: "bank.account_number", Type: "text", LabelKey: "accountNumber", Required: true},
			{Path: "bank.routing_number", Key: "bank.routing_number", Type: "text", LabelKey: "routingNumber", Required: true},
		}
	}

	bankFields = append(bankFields,
		FieldSpec{Path: "bank.account_holder", Key: "bank.account_holder", Type: "text", LabelKey: "accountHolder", Required: true},
		FieldSpec{Path: "bank.bank_country", Key: "bank.bank_country", Type: "select", LabelKey: "bankCountry", Required: true},
	)

	return append(sections, FieldSection{
		ID:       "bank",
		TitleKey: "bankAccount",
		Fields:   bankFields,
	})
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

// notifCooldown prevents sending duplicate notifications within 5 minutes per user.
var notifCooldown sync.Map // map[uuid.UUID]time.Time

// NotifyNewRequirements sends a notification when Stripe requires new information.
// Deduplicates: max 1 notification per user per 5 minutes.
func (s *Service) NotifyNewRequirements(ctx context.Context, userID uuid.UUID, reqs *domain.AccountRequirements) {
	if s.notifications == nil {
		return
	}
	if len(reqs.CurrentlyDue) == 0 && len(reqs.EventuallyDue) == 0 && len(reqs.PastDue) == 0 {
		return
	}

	// Cooldown check: skip if notified within last 5 minutes
	if lastSent, ok := notifCooldown.Load(userID); ok {
		if time.Since(lastSent.(time.Time)) < 5*time.Minute {
			return
		}
	}
	notifCooldown.Store(userID, time.Now())

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
