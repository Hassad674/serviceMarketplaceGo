package gdpr

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ExportVersion is the schema version stamped into manifest.json.
// Bumped whenever we change the file shape so a future tool that
// imports old exports can branch on the version. Decision 1 of
// the P5 brief.
const ExportVersion = "1.0"

// ErrEmptyExport is returned when an Export is built with no user
// rows — there must always be at least the profile section, even
// for a brand new user, otherwise the ZIP makes no sense.
var ErrEmptyExport = errors.New("gdpr export: missing profile data")

// Export aggregates every JSON file the export ZIP contains. Each
// field is a slice of map[string]any rather than typed entities
// so the GDPR feature stays loosely coupled to the rest of the
// domain — adding a new section is a SQL change in the repository,
// not a wave of new types here.
//
// The repository layer fills these slices with plain rows and the
// service layer serializes them to a ZIP per Decision 1 (one JSON
// file per domain + manifest.json + README.txt).
type Export struct {
	UserID    uuid.UUID
	Email     string
	Timestamp time.Time

	Profile       []map[string]any // user + profile + organization
	Proposals     []map[string]any // proposals authored by or addressed to the user
	Messages      []map[string]any // messages sent or received
	Invoices      []map[string]any // invoices issued to the user / their org
	Reviews       []map[string]any // reviews written + received
	AuditLogs     []map[string]any // audit_logs WHERE user_id = $self
	Notifications []map[string]any // notifications addressed to the user
	Jobs          []map[string]any // jobs published or applied to
	Portfolios    []map[string]any // portfolio items
	Reports       []map[string]any // moderation reports involving the user

	// Locale ("en" / "fr") is set by the service from the user's
	// preferred language and embedded in manifest.json so the
	// README.txt template can be picked correctly at write time.
	Locale string
}

// Validate enforces the minimum invariant: an export without a
// profile section is a programming error (the user always exists
// at the moment we build the export). The repository layer is
// expected to populate at least Profile.
func (e *Export) Validate() error {
	if e == nil {
		return ErrEmptyExport
	}
	if len(e.Profile) == 0 {
		return ErrEmptyExport
	}
	return nil
}

// FileNames lists every JSON file the export contains, in the
// canonical order they appear in manifest.json. Used by the ZIP
// writer + by the README template renderer so the order matches
// the human-readable description.
func (e *Export) FileNames() []string {
	return []string{
		"profile.json",
		"proposals.json",
		"messages.json",
		"invoices.json",
		"reviews.json",
		"notifications.json",
		"jobs.json",
		"portfolios.json",
		"reports.json",
		"audit_logs.json",
	}
}

// SectionFor returns the rows for the named JSON file. Centralized
// here so the writer doesn't have to switch on a magic string in
// two places. Returns nil for unknown names (write writes "[]").
func (e *Export) SectionFor(name string) []map[string]any {
	switch name {
	case "profile.json":
		return e.Profile
	case "proposals.json":
		return e.Proposals
	case "messages.json":
		return e.Messages
	case "invoices.json":
		return e.Invoices
	case "reviews.json":
		return e.Reviews
	case "notifications.json":
		return e.Notifications
	case "jobs.json":
		return e.Jobs
	case "portfolios.json":
		return e.Portfolios
	case "reports.json":
		return e.Reports
	case "audit_logs.json":
		return e.AuditLogs
	}
	return nil
}
