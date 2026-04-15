package referral

import (
	"encoding/json"
	"strings"
)

// SnapshotVersion identifies the schema version of an IntroSnapshot payload.
// Bump this constant whenever the IntroSnapshot struct gains or removes a field;
// the postgres adapter persists the version alongside the JSONB blob so older
// rows can be decoded with the legacy shape.
const SnapshotVersion = 1

// IntroSnapshot is the anonymised view of the provider and the client that the
// counter-party sees BEFORE the intro is activated. It is frozen at creation
// time and never mutates; once the intro reaches StatusActive the UI swaps to
// the live profile.
//
// The snapshot is NOT a thin pointer to live profile data on purpose:
//
//  1. It must remain stable even if the underlying profile changes after the
//     intro is sent (so the recipient sees what was promised).
//  2. It must be safe to render to a non-authorised viewer (the client viewing
//     the provider's snapshot has no read access to the provider's full profile
//     until activation).
//
// The referrer chooses which fields to reveal via toggles in the creation
// wizard; unselected fields are left as their zero value here.
type IntroSnapshot struct {
	// Provider-side anonymised fields (visible to the client before activation).
	Provider ProviderSnapshot `json:"provider"`

	// Client-side anonymised fields (visible to the provider before activation).
	Client ClientSnapshot `json:"client"`
}

// ProviderSnapshot holds the safe-to-reveal provider attributes. Names, emails,
// avatars, social links and exact city are intentionally absent.
type ProviderSnapshot struct {
	ExpertiseDomains  []string `json:"expertise_domains,omitempty"`
	YearsExperience   *int     `json:"years_experience,omitempty"`
	AverageRating     *float64 `json:"average_rating,omitempty"`
	ReviewCount       *int     `json:"review_count,omitempty"`
	PricingMinCents   *int64   `json:"pricing_min_cents,omitempty"`
	PricingMaxCents   *int64   `json:"pricing_max_cents,omitempty"`
	PricingCurrency   string   `json:"pricing_currency,omitempty"`
	PricingType       string   `json:"pricing_type,omitempty"` // daily, hourly, project_from, project_range
	Region            string   `json:"region,omitempty"`        // generic region, NOT the city
	Languages         []string `json:"languages,omitempty"`
	AvailabilityState string   `json:"availability_state,omitempty"`
}

// ClientSnapshot holds the safe-to-reveal client attributes. Company name,
// logo, website and exact address are intentionally absent.
type ClientSnapshot struct {
	Industry          string `json:"industry,omitempty"`
	SizeBucket        string `json:"size_bucket,omitempty"` // tpe, pme, eti, ge
	Region            string `json:"region,omitempty"`
	BudgetEstimateMin *int64 `json:"budget_estimate_min_cents,omitempty"`
	BudgetEstimateMax *int64 `json:"budget_estimate_max_cents,omitempty"`
	BudgetCurrency    string `json:"budget_currency,omitempty"`
	NeedSummary       string `json:"need_summary,omitempty"` // free-text written by the referrer
	Timeline          string `json:"timeline,omitempty"`
}

// Validate enforces the minimal invariants on an IntroSnapshot. It is called
// at NewReferral time so a malformed snapshot is rejected before the row is
// persisted.
func (s IntroSnapshot) Validate() error {
	if err := s.Provider.validate(); err != nil {
		return err
	}
	if err := s.Client.validate(); err != nil {
		return err
	}
	return nil
}

const (
	maxSnapshotStringLen      = 240
	maxSnapshotListLen        = 24
	maxSnapshotListItemLen    = 80
	maxSnapshotNeedSummaryLen = 800
)

func (p ProviderSnapshot) validate() error {
	if err := validateStringLen(p.Region, maxSnapshotStringLen); err != nil {
		return err
	}
	if err := validateStringLen(p.PricingCurrency, 8); err != nil {
		return err
	}
	if err := validateStringLen(p.PricingType, 32); err != nil {
		return err
	}
	if err := validateStringLen(p.AvailabilityState, 32); err != nil {
		return err
	}
	if err := validateStringList(p.ExpertiseDomains); err != nil {
		return err
	}
	if err := validateStringList(p.Languages); err != nil {
		return err
	}
	if p.YearsExperience != nil && (*p.YearsExperience < 0 || *p.YearsExperience > 100) {
		return ErrSnapshotInvalid
	}
	if p.AverageRating != nil && (*p.AverageRating < 0 || *p.AverageRating > 5) {
		return ErrSnapshotInvalid
	}
	if p.ReviewCount != nil && *p.ReviewCount < 0 {
		return ErrSnapshotInvalid
	}
	if err := validateRange(p.PricingMinCents, p.PricingMaxCents); err != nil {
		return err
	}
	return nil
}

func (c ClientSnapshot) validate() error {
	if err := validateStringLen(c.Industry, maxSnapshotStringLen); err != nil {
		return err
	}
	if err := validateStringLen(c.SizeBucket, 32); err != nil {
		return err
	}
	if err := validateStringLen(c.Region, maxSnapshotStringLen); err != nil {
		return err
	}
	if err := validateStringLen(c.BudgetCurrency, 8); err != nil {
		return err
	}
	if err := validateStringLen(c.Timeline, maxSnapshotStringLen); err != nil {
		return err
	}
	if l := len([]rune(strings.TrimSpace(c.NeedSummary))); l > maxSnapshotNeedSummaryLen {
		return ErrSnapshotInvalid
	}
	return validateRange(c.BudgetEstimateMin, c.BudgetEstimateMax)
}

func validateStringLen(s string, max int) error {
	if len([]rune(strings.TrimSpace(s))) > max {
		return ErrSnapshotInvalid
	}
	return nil
}

func validateStringList(items []string) error {
	if len(items) > maxSnapshotListLen {
		return ErrSnapshotInvalid
	}
	for _, it := range items {
		if err := validateStringLen(it, maxSnapshotListItemLen); err != nil {
			return err
		}
	}
	return nil
}

func validateRange(min, max *int64) error {
	if min != nil && *min < 0 {
		return ErrSnapshotInvalid
	}
	if max != nil && *max < 0 {
		return ErrSnapshotInvalid
	}
	if min != nil && max != nil && *max < *min {
		return ErrSnapshotInvalid
	}
	return nil
}

// MarshalJSON / UnmarshalJSON satisfy the encoding/json contract through a
// fixed alias type; the postgres adapter relies on the standard library to
// serialise the snapshot before storing it as a JSONB value.

// MarshalSnapshot marshals an IntroSnapshot into its JSONB-friendly bytes.
func MarshalSnapshot(s IntroSnapshot) ([]byte, error) {
	return json.Marshal(s)
}

// UnmarshalSnapshot decodes a JSONB-encoded IntroSnapshot.
func UnmarshalSnapshot(raw []byte) (IntroSnapshot, error) {
	var s IntroSnapshot
	if len(raw) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return IntroSnapshot{}, err
	}
	return s, nil
}
