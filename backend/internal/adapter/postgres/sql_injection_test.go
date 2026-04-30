package postgres

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/profile"
	domainskill "marketplace-backend/internal/domain/skill"
	"marketplace-backend/internal/port/repository"
)

// ---------------------------------------------------------------------------
// SEC: SQL injection — gosec G201/G202 suppressed sites
//
// The five sites flagged by gosec on `main` in /tmp/mp-phase-1.5/backend
// (expertise_repository.go:187, profile_skill_repository.go:246,
// profile_repository.go:242, user_repository.go:341, invoicing_repository.go:828)
// all build SQL strings via concatenation. Each site only splices
// numeric placeholder strings (`$1`, `$2`, …) into the SQL — the
// caller's user input flows through `args` to ExecContext / QueryContext
// and is bound by lib/pq's parameterised protocol, never embedded in
// the SQL text.
//
// These tests prove that invariant by driving each function with
// classic injection payloads and verifying:
//   1. The generated SQL string MUST NOT contain the payload as
//      executable text — sqlmock's expectation regex confirms the
//      bound shape;
//   2. The Args slice MUST carry the payload as an opaque bound
//      value — sqlmock's `WithArgs` matcher fails the test if the
//      payload reaches the database via any other path.
// ---------------------------------------------------------------------------

// classicSQLInjectionPayloads is the canonical attack surface used
// across every test below. Adding a new payload here automatically
// raises coverage on every fixture; no per-test plumbing required.
var classicSQLInjectionPayloads = []struct {
	name    string
	payload string
}{
	{"drop_table", "'; DROP TABLE users; --"},
	{"or_1_eq_1", "' OR 1=1 --"},
	{"comment_terminator", "' /* */ --"},
	{"union_select", "' UNION SELECT 1,2,3 --"},
	{"stacked_query", "1; INSERT INTO users (email) VALUES ('hacker') --"},
	{"null_byte", "abc\x00; DROP TABLE users; --"},
	{"hex_encoded", "\\x27; DROP TABLE users; --"},
	{"newline_smuggle", "\n; DROP TABLE users; --"},
	{"backslash_quote", "\\'; DROP TABLE users; --"},
	{"double_quote", `"; DROP TABLE users; --`},
}

// ---------------------------------------------------------------------------
// 1) profile_repository.UpdateAvailability — column-name interpolation
// ---------------------------------------------------------------------------

func TestProfileRepository_UpdateAvailability_BindsPayloadAsArg(t *testing.T) {
	for _, tt := range classicSQLInjectionPayloads {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := NewProfileRepository(db)
			orgID := uuid.New()
			payload := profile.AvailabilityStatus(tt.payload)

			mock.ExpectExec(`UPDATE profiles SET availability_status = \$2 WHERE organization_id = \$1`).
				WithArgs(orgID, tt.payload).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err = repo.UpdateAvailability(context.Background(), orgID, &payload, nil)
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet(),
				"payload %q should be bound as $2, not interpolated into SQL", tt.payload)
		})
	}
}

func TestProfileRepository_UpdateAvailability_BothColumns(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewProfileRepository(db)
	orgID := uuid.New()
	direct := profile.AvailabilityStatus("available")
	ref := profile.AvailabilityStatus("away")

	mock.ExpectExec(`UPDATE profiles SET availability_status = \$2, referrer_availability_status = \$3 WHERE organization_id = \$1`).
		WithArgs(orgID, "available", "away").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateAvailability(context.Background(), orgID, &direct, &ref)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// 2) expertise_repository.insertExpertiseRows — multi-row INSERT
// ---------------------------------------------------------------------------

func TestExpertiseRepository_Replace_BindsPayloadAsArg(t *testing.T) {
	for _, tt := range classicSQLInjectionPayloads {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := NewExpertiseRepository(db)
			orgID := uuid.New()
			keys := []string{"design", tt.payload, "marketing"}

			mock.ExpectBegin()
			mock.ExpectExec(`DELETE FROM organization_expertise_domains WHERE organization_id = \$1`).
				WithArgs(orgID).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec(`INSERT INTO organization_expertise_domains \(organization_id, domain_key, position\) VALUES \(\$1, \$2, 0\), \(\$1, \$3, 1\), \(\$1, \$4, 2\)`).
				WithArgs(orgID, "design", tt.payload, "marketing").
				WillReturnResult(sqlmock.NewResult(0, 3))
			mock.ExpectCommit()

			err = repo.Replace(context.Background(), orgID, keys)
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet(),
				"payload %q should be bound as $3, not interpolated", tt.payload)
		})
	}
}

func TestExpertiseRepository_Replace_EmptyClearsAllRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewExpertiseRepository(db)
	orgID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM organization_expertise_domains WHERE organization_id = \$1`).
		WithArgs(orgID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Replace(context.Background(), orgID, []string{}))
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// 3) profile_skill_repository.insertProfileSkillRows — multi-row INSERT
// ---------------------------------------------------------------------------

func TestProfileSkillRepository_ReplaceForOrg_BindsPayloadAsArg(t *testing.T) {
	for _, tt := range classicSQLInjectionPayloads {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := NewProfileSkillRepository(db)
			orgID := uuid.New()
			skills := []*domainskill.ProfileSkill{
				{SkillText: "Go", Position: 0},
				{SkillText: tt.payload, Position: 1},
			}

			mock.ExpectBegin()
			mock.ExpectExec(`DELETE FROM profile_skills WHERE organization_id = \$1`).
				WithArgs(orgID).
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec(`INSERT INTO profile_skills .* VALUES \(\$1, \$2, \$3, now\(\)\), \(\$1, \$4, \$5, now\(\)\)`).
				WithArgs(orgID, "Go", 0, tt.payload, 1).
				WillReturnResult(sqlmock.NewResult(0, 2))
			mock.ExpectCommit()

			err = repo.ReplaceForOrg(context.Background(), orgID, skills)
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet(),
				"skill_text payload %q should be bound as $4", tt.payload)
		})
	}
}

func TestProfileSkillRepository_ReplaceForOrg_EmptyClears(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewProfileSkillRepository(db)
	orgID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM profile_skills WHERE organization_id = \$1`).
		WithArgs(orgID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.ReplaceForOrg(context.Background(), orgID, []*domainskill.ProfileSkill{}))
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// 4) user_repository.ListAdmin — dynamic WHERE + LIMIT + OFFSET
// ---------------------------------------------------------------------------

func TestUserRepository_ListAdmin_BindsRolePayload(t *testing.T) {
	for _, tt := range classicSQLInjectionPayloads {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := NewUserRepository(db)
			filters := repository.AdminUserFilters{
				Role:  tt.payload,
				Limit: 10,
			}

			rows := sqlmock.NewRows([]string{
				"id", "email", "hashed_password", "first_name", "last_name", "display_name",
				"role", "account_type", "referrer_enabled", "email_notifications_enabled",
				"is_admin", "status", "suspended_at", "suspension_reason", "suspension_expires_at",
				"banned_at", "ban_reason", "organization_id", "linkedin_id", "google_id",
				"email_verified", "created_at", "updated_at",
			})
			mock.ExpectQuery(`SELECT .* FROM users WHERE role = \$1 ORDER BY created_at DESC, id DESC LIMIT \$2`).
				WithArgs(tt.payload, 11).
				WillReturnRows(rows)

			_, _, err = repo.ListAdmin(context.Background(), filters)
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet(),
				"role payload %q should be bound as $1", tt.payload)
		})
	}
}

func TestUserRepository_ListAdmin_BindsSearchPayload(t *testing.T) {
	for _, tt := range classicSQLInjectionPayloads {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			repo := NewUserRepository(db)
			filters := repository.AdminUserFilters{
				Search: tt.payload,
				Limit:  20,
			}

			rows := sqlmock.NewRows([]string{
				"id", "email", "hashed_password", "first_name", "last_name", "display_name",
				"role", "account_type", "referrer_enabled", "email_notifications_enabled",
				"is_admin", "status", "suspended_at", "suspension_reason", "suspension_expires_at",
				"banned_at", "ban_reason", "organization_id", "linkedin_id", "google_id",
				"email_verified", "created_at", "updated_at",
			})
			mock.ExpectQuery(`SELECT .* FROM users WHERE \(first_name ILIKE \$1 OR last_name ILIKE \$1 OR email ILIKE \$1 OR display_name ILIKE \$1\) ORDER BY created_at DESC, id DESC LIMIT \$2`).
				WithArgs("%"+tt.payload+"%", 21).
				WillReturnRows(rows)

			_, _, err = repo.ListAdmin(context.Background(), filters)
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet(),
				"search payload %q should be bound, not interpolated", tt.payload)
		})
	}
}

func TestUserRepository_ListAdmin_OffsetUsesPlaceholder(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	filters := repository.AdminUserFilters{
		Page:  3,
		Limit: 10,
	}

	rows := sqlmock.NewRows([]string{
		"id", "email", "hashed_password", "first_name", "last_name", "display_name",
		"role", "account_type", "referrer_enabled", "email_notifications_enabled",
		"is_admin", "status", "suspended_at", "suspension_reason", "suspension_expires_at",
		"banned_at", "ban_reason", "organization_id", "linkedin_id", "google_id",
		"email_verified", "created_at", "updated_at",
	})
	// The actual emitted query is `LIMIT $2 OFFSET $1` because
	// the offset placeholder is reserved before the LIMIT one in
	// ListAdmin's argIdx counter. Both flow through ARGS, no
	// concatenation of payload text.
	mock.ExpectQuery(`SELECT .* FROM users\s* ORDER BY created_at DESC, id DESC LIMIT \$2 OFFSET \$1`).
		WithArgs(20, 11). // (page-1)*limit, then limit+1
		WillReturnRows(rows)

	_, _, err = repo.ListAdmin(context.Background(), filters)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// 5) invoicing_repository.ListInvoicesAdmin — heaviest filter set
// ---------------------------------------------------------------------------

func TestInvoicingRepository_AdminListBindsSearchPayload(t *testing.T) {
	for _, tt := range classicSQLInjectionPayloads {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			invoiceRepo := NewInvoiceRepository(db)
			filters := repository.AdminInvoiceFilters{
				Search: tt.payload,
			}

			cols := []string{
				"id", "number", "is_credit_note", "recipient_organization_id",
				"recipient_legal_name", "issued_at", "amount_incl_tax_cents",
				"currency", "tax_regime", "status", "pdf_r2_key",
				"original_invoice_id", "source_type",
			}
			mock.ExpectQuery(`SELECT id, number, is_credit_note,.*FROM combined`).
				WithArgs("%"+tt.payload+"%", 11).
				WillReturnRows(sqlmock.NewRows(cols))

			_, _, err = invoiceRepo.ListInvoicesAdmin(context.Background(), filters, "", 10)
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet(),
				"search payload %q should be bound, not interpolated", tt.payload)
		})
	}
}

// TestInvoicingRepository_AdminListUnknownStatus_AndFalse confirms
// the "unknown status" branch falls back to "AND FALSE" — even with
// payloads that look like valid status filters, a bogus value cannot
// drive SQL execution.
func TestInvoicingRepository_AdminListUnknownStatus_AndFalse(t *testing.T) {
	for _, tt := range classicSQLInjectionPayloads {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			invoiceRepo := NewInvoiceRepository(db)
			filters := repository.AdminInvoiceFilters{
				Status: tt.payload,
			}
			cols := []string{
				"id", "number", "is_credit_note", "recipient_organization_id",
				"recipient_legal_name", "issued_at", "amount_incl_tax_cents",
				"currency", "tax_regime", "status", "pdf_r2_key",
				"original_invoice_id", "source_type",
			}
			mock.ExpectQuery(`AND FALSE.*FROM combined`).
				WithArgs(21).
				WillReturnRows(sqlmock.NewRows(cols))

			_, _, err = invoiceRepo.ListInvoicesAdmin(context.Background(), filters, "", 20)
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestInvoicingRepository_AdminListAcceptsValidStatusOnly(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"empty -> no filter", ""},
		{"subscription", "subscription"},
		{"monthly_commission", "monthly_commission"},
		{"credit_note", "credit_note"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			invoiceRepo := NewInvoiceRepository(db)
			filters := repository.AdminInvoiceFilters{Status: tt.status}
			cols := []string{
				"id", "number", "is_credit_note", "recipient_organization_id",
				"recipient_legal_name", "issued_at", "amount_incl_tax_cents",
				"currency", "tax_regime", "status", "pdf_r2_key",
				"original_invoice_id", "source_type",
			}

			expectedArgs := []driver.Value{}
			switch tt.status {
			case "subscription", "monthly_commission":
				expectedArgs = append(expectedArgs, tt.status)
			}
			expectedArgs = append(expectedArgs, 11)

			mock.ExpectQuery(`FROM combined`).
				WithArgs(expectedArgs...).
				WillReturnRows(sqlmock.NewRows(cols))

			_, _, err = invoiceRepo.ListInvoicesAdmin(context.Background(), filters, "", 10)
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
