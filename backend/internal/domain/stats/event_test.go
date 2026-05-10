package stats_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"marketplace-backend/internal/domain/stats"
)

func TestPersona_IsValid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   stats.Persona
		want bool
	}{
		{"freelance ok", stats.PersonaFreelance, true},
		{"agency ok", stats.PersonaAgency, true},
		{"referrer ok", stats.PersonaReferrer, true},
		{"empty rejected", stats.Persona(""), false},
		{"unknown rejected", stats.Persona("admin"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.in.IsValid())
		})
	}
}

func TestCameFrom_IsValid(t *testing.T) {
	t.Parallel()
	for _, value := range []stats.CameFrom{
		stats.CameFromSearch, stats.CameFromList, stats.CameFromDirect,
		stats.CameFromReferral, stats.CameFromUnknown,
	} {
		assert.True(t, value.IsValid(), "expected %s valid", value)
	}
	assert.False(t, stats.CameFrom("").IsValid())
	assert.False(t, stats.CameFrom("rss").IsValid())
}

func TestNewViewEvent_HappyPath(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	uid := uuid.New()
	q := "go developer"
	pos := 3
	ev, err := stats.NewViewEvent(stats.NewViewEventInput{
		OrganizationID:     orgID,
		Persona:            stats.PersonaFreelance,
		ViewerUserID:       &uid,
		ViewerIPAnonymized: "203.0.113.0/24",
		ViewerUAHash:       "deadbeef",
		CameFrom:           stats.CameFromSearch,
		SearchQuery:        &q,
		SearchPosition:     &pos,
	})
	assert.NoError(t, err)
	assert.NotNil(t, ev)
	assert.NotEqual(t, uuid.Nil, ev.ID)
	assert.False(t, ev.CreatedAt.IsZero())
	assert.Equal(t, orgID, ev.OrganizationID)
	assert.Equal(t, "go developer", *ev.SearchQuery)
	assert.Equal(t, 3, *ev.SearchPosition)
}

func TestNewViewEvent_TrimsAndNullsEmptyOptionals(t *testing.T) {
	t.Parallel()
	emptyQ := "  "
	emptyRef := ""
	ev, err := stats.NewViewEvent(stats.NewViewEventInput{
		OrganizationID:     uuid.New(),
		Persona:            stats.PersonaAgency,
		ViewerIPAnonymized: "10.0.0.0/24",
		ViewerUAHash:       "x",
		CameFrom:           stats.CameFromDirect,
		SearchQuery:        &emptyQ,
		ReferrerURL:        &emptyRef,
	})
	assert.NoError(t, err)
	assert.Nil(t, ev.SearchQuery)
	assert.Nil(t, ev.ReferrerURL)
}

func TestNewViewEvent_RejectsBadInputs(t *testing.T) {
	t.Parallel()
	base := stats.NewViewEventInput{
		OrganizationID:     uuid.New(),
		Persona:            stats.PersonaFreelance,
		ViewerIPAnonymized: "203.0.113.0/24",
		ViewerUAHash:       "deadbeef",
		CameFrom:           stats.CameFromDirect,
	}
	negPos := 0
	cases := []struct {
		name    string
		mutate  func(in *stats.NewViewEventInput)
		wantErr error
	}{
		{
			name:    "missing org id",
			mutate:  func(in *stats.NewViewEventInput) { in.OrganizationID = uuid.Nil },
			wantErr: stats.ErrOrgIDRequired,
		},
		{
			name:    "invalid persona",
			mutate:  func(in *stats.NewViewEventInput) { in.Persona = "admin" },
			wantErr: stats.ErrInvalidPersona,
		},
		{
			name:    "invalid came_from",
			mutate:  func(in *stats.NewViewEventInput) { in.CameFrom = "rss" },
			wantErr: stats.ErrInvalidCameFrom,
		},
		{
			name:    "missing ip",
			mutate:  func(in *stats.NewViewEventInput) { in.ViewerIPAnonymized = " " },
			wantErr: stats.ErrIPRequired,
		},
		{
			name:    "non-parseable ip",
			mutate:  func(in *stats.NewViewEventInput) { in.ViewerIPAnonymized = "not-an-ip" },
			wantErr: stats.ErrIPInvalid,
		},
		{
			name:    "missing ua hash",
			mutate:  func(in *stats.NewViewEventInput) { in.ViewerUAHash = " " },
			wantErr: stats.ErrUAHashRequired,
		},
		{
			name: "search position zero invalid",
			mutate: func(in *stats.NewViewEventInput) {
				in.SearchPosition = &negPos
			},
			wantErr: stats.ErrSearchPosNonNeg,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := base
			tc.mutate(&in)
			ev, err := stats.NewViewEvent(in)
			assert.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, ev)
		})
	}
}

func TestAnonymizeIP(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"ipv4 truncated to /24", "203.0.113.42", "203.0.113.0/24"},
		{"ipv4 zero already", "10.0.0.0", "10.0.0.0/24"},
		{"ipv6 /64", "2001:db8::1234", "2001:db8::/64"},
		{"ipv6 mixed", "2001:db8:abcd:0012:0:0:0:0001", "2001:db8:abcd:12::/64"},
		{"invalid passthrough", "not-an-ip", "not-an-ip"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := stats.AnonymizeIP(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPeriodDays(t *testing.T) {
	t.Parallel()
	assert.True(t, stats.Period7Days.IsValid())
	assert.True(t, stats.Period30Days.IsValid())
	assert.True(t, stats.Period90Days.IsValid())
	assert.False(t, stats.PeriodDays(1).IsValid())
	assert.False(t, stats.PeriodDays(365).IsValid())

	got, err := stats.ParsePeriodDays(7)
	assert.NoError(t, err)
	assert.Equal(t, stats.Period7Days, got)

	got, err = stats.ParsePeriodDays(42)
	assert.ErrorIs(t, err, stats.ErrPeriodInvalid)
	assert.Equal(t, stats.Period30Days, got)
}

func TestClampLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 10, stats.ClampLimit(0))
	assert.Equal(t, 10, stats.ClampLimit(-3))
	assert.Equal(t, 50, stats.ClampLimit(50))
	assert.Equal(t, 100, stats.ClampLimit(500))
	// rule of three lower-bound coverage:
	assert.Equal(t, 1, stats.ClampLimit(1))
}

func TestEventErrorsHaveDistinctMessages(t *testing.T) {
	t.Parallel()
	// Sanity: every sentinel has a non-empty message + the prefix
	// "stats:" so log lines are easy to grep on.
	for _, err := range []error{
		stats.ErrInvalidPersona, stats.ErrInvalidCameFrom, stats.ErrOrgIDRequired,
		stats.ErrIPRequired, stats.ErrIPInvalid, stats.ErrUAHashRequired,
		stats.ErrSearchPosNonNeg, stats.ErrPeriodInvalid, stats.ErrInvalidLimit,
	} {
		assert.NotEmpty(t, err.Error())
		assert.True(t, strings.HasPrefix(err.Error(), "stats:"))
	}
}
