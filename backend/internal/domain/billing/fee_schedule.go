// Package billing holds the platform fee schedule for milestone payments.
//
// The fee is a flat amount per released milestone, tiered by the prestataire's
// role (freelance or agency) and the milestone amount. The schedule is
// deliberately kept in code rather than configuration: a fee change is a
// business decision that deserves a code review and a migration of the
// caller wiring, not a silent env var swap. Historical payment_records keep
// the fee amount that was computed at creation time — changes here apply
// only to future milestones.
package billing

// Role identifies the prestataire kind charged for the milestone.
// Enterprise and admin never pay platform fees, so they are not members.
type Role string

const (
	RoleFreelance Role = "freelance"
	RoleAgency    Role = "agency"
)

// IsValid reports whether r is a known prestataire role.
func (r Role) IsValid() bool {
	return r == RoleFreelance || r == RoleAgency
}

// RoleFromUser maps an application-level user role string to the billing
// role used by the fee schedule. An agency user is charged the agency grid;
// every other role (provider, enterprise, admin, unknown) falls back to the
// freelance grid — which is the cheaper side, so an unmapped role can never
// over-charge a prestataire. Enterprise and admin users never reach the fee
// calculation path in practice (they are clients or internal), but the
// fallback keeps the function total so callers never need nil checks.
func RoleFromUser(userRole string) Role {
	if userRole == "agency" {
		return RoleAgency
	}
	return RoleFreelance
}

// Tier is a single bracket in the fee schedule.
// MaxCents is the exclusive upper bound for the bracket, nil for the last
// (open-ended) tier. FeeCents is the flat fee charged when the milestone
// amount falls into this bracket.
type Tier struct {
	Label    string
	MaxCents *int64
	FeeCents int64
}

// Result is the outcome of a fee calculation.
// Tiers is returned alongside the computation so callers (web/mobile) can
// render the full grid with the active tier highlighted without duplicating
// the schedule on the client.
type Result struct {
	AmountCents     int64
	FeeCents        int64
	NetCents        int64
	Role            Role
	ActiveTierIndex int
	Tiers           []Tier
}

var freelanceTiers = []Tier{
	{Label: "Moins de 200 €", MaxCents: ptrCents(20000), FeeCents: 900},
	{Label: "200 € – 1 000 €", MaxCents: ptrCents(100000), FeeCents: 1500},
	{Label: "Plus de 1 000 €", MaxCents: nil, FeeCents: 2500},
}

var agencyTiers = []Tier{
	{Label: "Moins de 500 €", MaxCents: ptrCents(50000), FeeCents: 1900},
	{Label: "500 € – 2 500 €", MaxCents: ptrCents(250000), FeeCents: 3900},
	{Label: "Plus de 2 500 €", MaxCents: nil, FeeCents: 6900},
}

// TiersFor returns the fee tiers for the given role. The returned slice is
// a copy — callers may not mutate the package-level schedule.
func TiersFor(role Role) []Tier {
	src := scheduleFor(role)
	out := make([]Tier, len(src))
	copy(out, src)
	return out
}

func scheduleFor(role Role) []Tier {
	switch role {
	case RoleAgency:
		return agencyTiers
	default:
		return freelanceTiers
	}
}

// Calculate returns the platform fee for a milestone of the given amount and
// prestataire role. Amounts of zero or less yield a zero fee and an active
// tier index of -1 (no bracket applies). Callers should reject non-positive
// milestone amounts at the validation layer; Calculate stays safe by design.
func Calculate(role Role, amountCents int64) Result {
	tiers := scheduleFor(role)
	idx := -1
	var fee int64
	if amountCents > 0 {
		for i, t := range tiers {
			if t.MaxCents == nil || amountCents < *t.MaxCents {
				idx = i
				fee = t.FeeCents
				break
			}
		}
	}
	return Result{
		AmountCents:     amountCents,
		FeeCents:        fee,
		NetCents:        amountCents - fee,
		Role:            role,
		ActiveTierIndex: idx,
		Tiers:           TiersFor(role),
	}
}

func ptrCents(v int64) *int64 { return &v }
