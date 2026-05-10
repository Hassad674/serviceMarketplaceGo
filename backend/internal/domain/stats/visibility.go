package stats

import (
	"errors"
	"time"
)

// PeriodDays is the rolling window applied to every stats query. We
// pin a small set so the underlying indexes are predictable and the
// frontend's cache key surface stays bounded.
type PeriodDays int

const (
	Period7Days  PeriodDays = 7
	Period30Days PeriodDays = 30
	Period90Days PeriodDays = 90
)

// IsValid returns true when the value matches one of the three
// supported window sizes.
func (p PeriodDays) IsValid() bool {
	switch p {
	case Period7Days, Period30Days, Period90Days:
		return true
	default:
		return false
	}
}

// ParsePeriodDays validates the int form (used by the handler) and
// returns the matching PeriodDays + a nil error. Unknown values
// resolve to the 30-day default with a non-nil error so callers can
// choose to surface a 400 instead.
func ParsePeriodDays(in int) (PeriodDays, error) {
	v := PeriodDays(in)
	if !v.IsValid() {
		return Period30Days, ErrPeriodInvalid
	}
	return v, nil
}

// DailyBucket is one day in a time series. Date is the UTC midnight
// the events fall into; Count is the number of rows in that bucket.
type DailyBucket struct {
	Date  time.Time
	Count int
}

// Visibility aggregates the per-org view metrics for a fixed window.
// Total counts every recorded view; Unique counts distinct
// (ip_anonymized, ua_hash) pairs. SearchAppearances tracks the rows
// where came_from='search'; AvgSearchPosition is the mean
// search_position across those rows (rounded to 2 decimals at the DB
// level).
type Visibility struct {
	OrganizationID    string
	PeriodDays        PeriodDays
	TotalViews        int
	UniqueViewers     int
	SearchAppearances int
	AvgSearchPosition float64 // 0 when SearchAppearances == 0
	Series            []DailyBucket
}

// ApplicationsTimeSeries is the analogue of DailyBucket-list for the
// enterprise dashboard's job-applications chart.
type ApplicationsTimeSeries struct {
	OrganizationID string
	PeriodDays     PeriodDays
	TotalCount     int
	Series         []DailyBucket
}

// ErrInvalidLimit is returned when keyword limits are out of range.
var ErrInvalidLimit = errors.New("stats: limit must be between 1 and 100")
