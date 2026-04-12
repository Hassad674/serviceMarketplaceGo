package job

import "time"

const (
	WeeklyQuota        = 10
	BonusPerMission    = 5
	MaxTokens          = 50
	MinBonusAmountCent = 3000 // 30 EUR minimum mission amount for bonus eligibility
)

// RefillPeriod is how often the weekly application-credit quota tops an
// organization's pool back up to WeeklyQuota. A pool whose balance is
// already at or above the quota (typically because the proposal fraud
// flow awarded BonusPerMission credits) is left untouched — the refill
// is floor-only, never destructive.
//
// The refill is enforced lazily on every read inside
// JobCreditRepository.GetOrCreate via a single atomic UPDATE, which
// both avoids the need for an external cron and guarantees the system
// self-heals after any downtime.
const RefillPeriod = 7 * 24 * time.Hour
