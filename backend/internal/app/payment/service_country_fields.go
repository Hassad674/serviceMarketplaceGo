package payment

import (
	"context"
	"fmt"
	"sort"
	"strings"

	domain "marketplace-backend/internal/domain/payment"
)

// CountryFieldsResponse describes the fields needed for a specific country.
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

// FieldSection groups related fields under an entity.
type FieldSection struct {
	ID       string      `json:"id"`
	TitleKey string      `json:"title_key"`
	Fields   []FieldSpec `json:"fields"`
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
	Urgency     string `json:"urgency,omitempty"`
}

// ibanCountries use IBAN format for bank accounts.
var ibanCountries = map[string]bool{
	"AT": true, "BE": true, "BG": true, "CH": true, "CY": true,
	"CZ": true, "DE": true, "DK": true, "EE": true, "ES": true,
	"FI": true, "FR": true, "GB": true, "GI": true, "GR": true,
	"HR": true, "HU": true, "IE": true, "IT": true, "LI": true,
	"LT": true, "LU": true, "LV": true, "MT": true, "NL": true,
	"NO": true, "PL": true, "PT": true, "RO": true, "SE": true,
	"SI": true, "SK": true,
}

// GetCountryFields returns the field requirements for a specific country.
func (s *Service) GetCountryFields(
	ctx context.Context, country, businessType string,
) (*CountryFieldsResponse, error) {
	if s.countrySpecs == nil {
		return nil, fmt.Errorf("country spec service not configured")
	}

	spec, err := s.countrySpecs.GetFieldsForCountry(ctx, country)
	if err != nil {
		return nil, fmt.Errorf("get country fields: %w", err)
	}

	fields := pickFieldsByBusinessType(spec, businessType)
	return buildResponse(spec, fields, country, businessType), nil
}

func pickFieldsByBusinessType(
	spec *domain.CountryFieldSpec, businessType string,
) []string {
	if businessType == "company" {
		return mergeFields(spec.CompanyMinimum, spec.CompanyAdditional)
	}
	return mergeFields(spec.IndividualMinimum, spec.IndividualAdditional)
}

// mergeFields combines minimum and additional field lists, deduplicating entries.
func mergeFields(minimum, additional []string) []string {
	seen := make(map[string]bool, len(minimum))
	merged := make([]string, 0, len(minimum)+len(additional))
	for _, f := range minimum {
		if !seen[f] {
			seen[f] = true
			merged = append(merged, f)
		}
	}
	for _, f := range additional {
		if !seen[f] {
			seen[f] = true
			merged = append(merged, f)
		}
	}
	return merged
}

// buildResponse creates the response grouped by entity sections.
func buildResponse(
	spec *domain.CountryFieldSpec, fields []string,
	country, businessType string,
) *CountryFieldsResponse {
	resp := &CountryFieldsResponse{
		Country:      country,
		BusinessType: businessType,
	}

	sectionMap := make(map[string][]FieldSpec)
	seen := make(dateSeen)

	for _, path := range fields {
		processField(path, resp, sectionMap, seen)
	}

	resp.Sections = buildSections(sectionMap)
	appendBankSection(resp, country)

	if businessType == "company" {
		resp.PersonRoles = domain.RequiredPersonRoles(fields)
	}

	return resp
}

// dateSeen tracks which entity+dateType combos have been collapsed already.
type dateSeen map[string]bool

// personEntities are entities handled by BusinessPersonsSection on the frontend.
// Their fields are excluded from dynamic sections — only person_roles is returned.
var personEntities = map[string]bool{
	"directors":  true,
	"owners":     true,
	"executives": true,
}

// processField handles a single Stripe field path.
func processField(
	path string, resp *CountryFieldsResponse,
	sectionMap map[string][]FieldSpec, seen dateSeen,
) {
	if domain.IsAutoHandled(path) {
		return
	}

	// Skip person entity fields — handled by BusinessPersonsSection on frontend.
	// These entities contribute only to PersonRoles (computed separately).
	entity := domain.EntityFromPath(path)
	if personEntities[entity] {
		return
	}

	// Document upload fields: add as inline upload zones AND mark docs required
	if domain.IsDocumentUploadField(path) {
		category := domain.DocumentCategoryFromPath(path)
		setDocumentRequired(resp, category)
		sectionMap[entity] = append(sectionMap[entity], FieldSpec{
			Path:     path,
			Key:      path,
			Type:     "document_upload",
			LabelKey: domain.FieldLabelKey(path),
			Required: true,
			IsExtra:  false,
		})
		return
	}

	// Collapse dob.day/month/year into a single date field per entity
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
		})
		return
	}

	// Collapse registration_date.day/month/year into single date field
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
	})
}

// sectionOrder defines the display order of entity sections.
// Note: directors, owners, executives are excluded — handled by
// BusinessPersonsSection on the frontend using person_roles.
var sectionOrder = []string{
	"individual", "representative", "company",
	"authorizer", "documents",
}

// buildSections converts the sectionMap into ordered FieldSections.
func buildSections(sectionMap map[string][]FieldSpec) []FieldSection {
	var sections []FieldSection

	for _, id := range sectionOrder {
		if fields, ok := sectionMap[id]; ok && len(fields) > 0 {
			sections = append(sections, FieldSection{
				ID:       id,
				TitleKey: domain.SectionTitleKey(id),
				Fields:   fields,
			})
			delete(sectionMap, id)
		}
	}

	// Add any remaining unknown sections alphabetically
	remaining := make([]string, 0, len(sectionMap))
	for id := range sectionMap {
		remaining = append(remaining, id)
	}
	sort.Strings(remaining)
	for _, id := range remaining {
		if len(sectionMap[id]) > 0 {
			sections = append(sections, FieldSection{
				ID:       id,
				TitleKey: domain.SectionTitleKey(id),
				Fields:   sectionMap[id],
			})
		}
	}

	return sections
}

// appendBankSection adds the bank account section to the response.
func appendBankSection(resp *CountryFieldsResponse, country string) {
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

	resp.Sections = append(resp.Sections, FieldSection{
		ID:       "bank",
		TitleKey: "bankAccount",
		Fields:   bankFields,
	})
}

func setDocumentRequired(resp *CountryFieldsResponse, docType string) {
	if docType == "individual" {
		resp.DocumentsRequired.Individual = true
	} else if docType == "company" {
		resp.DocumentsRequired.Company = true
	}
}
