package payment

import (
	"context"
	"fmt"
	"strings"

	domain "marketplace-backend/internal/domain/payment"
)

// CountryFieldsResponse describes the fields needed for a specific country and business type.
type CountryFieldsResponse struct {
	Country      string         `json:"country"`
	BusinessType string         `json:"business_type"`
	Sections     []FieldSection `json:"sections"`
	DocumentsRequired struct {
		Individual bool `json:"individual"`
		Company    bool `json:"company"`
	} `json:"documents_required"`
	PersonRoles []string `json:"person_roles"`
}

// FieldSection groups related fields.
type FieldSection struct {
	ID     string      `json:"id"`
	Fields []FieldSpec `json:"fields"`
}

// FieldSpec describes a single form field.
type FieldSpec struct {
	Path        string `json:"path"`
	Key         string `json:"key"`
	Type        string `json:"type"`
	LabelKey    string `json:"label_key"`
	Required    bool   `json:"required"`
	IsExtra     bool   `json:"is_extra"`
	Placeholder string `json:"placeholder,omitempty"`
}

// autoFields are handled automatically and should not appear in the form.
var autoFields = map[string]bool{
	"business_type":              true,
	"tos_acceptance.date":        true,
	"tos_acceptance.ip":          true,
	"external_account":           true,
	"business_profile.mcc":       true,
	"business_profile.url":       true,
	"business_profile.product_description": true,
}

// GetCountryFields returns the field requirements for a specific country.
func (s *Service) GetCountryFields(ctx context.Context, country, businessType string) (*CountryFieldsResponse, error) {
	if s.countrySpecs == nil {
		return nil, fmt.Errorf("country spec service not configured")
	}

	spec, err := s.countrySpecs.GetFieldsForCountry(ctx, country)
	if err != nil {
		return nil, fmt.Errorf("get country fields: %w", err)
	}

	fields := pickFieldsByBusinessType(spec, businessType)
	return buildCountryFieldsResponse(spec, fields, country, businessType), nil
}

// pickFieldsByBusinessType returns the minimum fields for the given business type.
func pickFieldsByBusinessType(spec *domain.CountryFieldSpec, businessType string) []string {
	if businessType == "company" {
		return spec.CompanyMinimum
	}
	return spec.IndividualMinimum
}

// buildCountryFieldsResponse creates the response from spec and fields.
func buildCountryFieldsResponse(spec *domain.CountryFieldSpec, fields []string, country, businessType string) *CountryFieldsResponse {
	resp := &CountryFieldsResponse{
		Country:      country,
		BusinessType: businessType,
	}

	sections := map[string][]FieldSpec{
		"personal": {},
		"address":  {},
		"extra":    {},
		"bank":     {},
	}

	for _, path := range fields {
		if isAutoField(path) {
			continue
		}

		if docType, ok := domain.IsDocumentField(path); ok {
			setDocumentRequired(resp, docType)
			continue
		}

		fieldSpec := mapPathToFieldSpec(path)
		section := categorizeField(path)
		sections[section] = append(sections[section], fieldSpec)
	}

	for _, id := range []string{"personal", "address", "extra", "bank"} {
		if len(sections[id]) > 0 {
			resp.Sections = append(resp.Sections, FieldSection{ID: id, Fields: sections[id]})
		}
	}

	if businessType == "company" {
		resp.PersonRoles = domain.RequiredPersonRoles(spec.CompanyMinimum)
	}

	return resp
}

// mapPathToFieldSpec converts a Stripe field path to a FieldSpec.
func mapPathToFieldSpec(path string) FieldSpec {
	dbField, isExtra := domain.MapStripeField(path)
	key := extractKey(path)
	return FieldSpec{
		Path:     path,
		Key:      key,
		Type:     inferFieldType(key),
		LabelKey: key,
		Required: true,
		IsExtra:  isExtra,
		Placeholder: placeholderForField(dbField),
	}
}

func isAutoField(path string) bool {
	for prefix := range autoFields {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func setDocumentRequired(resp *CountryFieldsResponse, docType string) {
	if docType == "individual" {
		resp.DocumentsRequired.Individual = true
	} else if docType == "company" {
		resp.DocumentsRequired.Company = true
	}
}

func categorizeField(path string) string {
	if strings.Contains(path, "address") {
		return "address"
	}
	if strings.Contains(path, "external_account") || strings.Contains(path, "bank") {
		return "bank"
	}
	_, isExtra := domain.MapStripeField(path)
	if isExtra {
		return "extra"
	}
	return "personal"
}

func extractKey(path string) string {
	parts := strings.Split(path, ".")
	return parts[len(parts)-1]
}

func inferFieldType(key string) string {
	switch key {
	case "dob", "day", "month", "year":
		return "date"
	case "political_exposure":
		return "select"
	default:
		return "text"
	}
}

func placeholderForField(dbField string) string {
	placeholders := map[string]string{
		"id_number":  "National ID number",
		"ssn_last_4": "Last 4 digits of SSN",
		"state":      "State / Province",
	}
	if p, ok := placeholders[dbField]; ok {
		return p
	}
	return ""
}
