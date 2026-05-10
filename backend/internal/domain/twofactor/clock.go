package twofactor

import "time"

// nowFn is the package-private clock used by the domain. Tests that
// need deterministic timestamps swap it via SetNowForTests and restore
// the default with RestoreNow. Production code uses time.Now via the
// default assignment below — the indirection has zero cost (one extra
// pointer hop) and avoids threading a clock through every constructor.
var nowFn = time.Now

// SetNowForTests overrides the clock used by domain constructors and
// predicates (IsExpired, IsPending, MarkUsed). Test code calls this in
// a t.Cleanup-paired pattern so a failing test cannot leak its
// override into the next case.
func SetNowForTests(fn func() time.Time) {
	nowFn = fn
}

// RestoreNow resets the clock to time.Now. Pair with SetNowForTests
// via t.Cleanup(twofactor.RestoreNow).
func RestoreNow() {
	nowFn = time.Now
}
