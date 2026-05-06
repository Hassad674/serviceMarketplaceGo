package redis

import (
	"golang.org/x/sync/singleflight"
)

// coalesceWithDoubleCheck encapsulates the V7 V6-1 stampede-protection
// recipe shared by every cache decorator in this package: collapse
// concurrent misses through singleflight AND re-peek the cache from
// inside the singleflight callback so a winner that just populated
// Redis is observed by losing peers that follow.
//
// V8 NEW-4: extracted from the 4 cache adapters that previously
// inlined the identical "outer peek → group.Do → inner peek → load"
// recipe (freelance_profile_cache, expertise_cache, skill_catalog_cache,
// profile_cache). The rule of three was triggered, and a generic
// helper removes ~30 LOC of duplication while making the failure-mode
// fix from V7 V6-1 reviewable in exactly one place.
//
// Callers shape:
//
//	hit, err := coalesceWithDoubleCheck(&c.group, key, c.peek, c.load)
//
// peek MUST return:
//   - (value, true, nil)  → cache hit, return value as-is
//   - (zero, true, err)   → negative-cache hit, return err to caller
//                            (e.g. freelanceprofile.ErrProfileNotFound)
//   - (zero, false, nil)  → cache miss / transient cache failure,
//                            advance to load
//
// load is called exactly once per coalesced burst on a true miss. Its
// return value is what gets cached and returned. peek is called twice
// per request on a miss (once outside, once inside the singleflight
// slot) — both calls are cheap Redis reads relative to the load it
// guards against.
//
// The helper does not handle cache invalidation or the cache write
// itself — callers do that inside their load function so the helper
// stays type-agnostic. Returning a non-nil err from load propagates
// through singleflight unchanged.
func coalesceWithDoubleCheck[T any](
	sf *singleflight.Group,
	key string,
	peek func() (T, bool, error),
	load func() (T, error),
) (T, error) {
	if v, found, sentinelErr := peek(); found {
		return v, sentinelErr
	}

	v, err, _ := sf.Do(key, func() (any, error) {
		// Double-check INSIDE the singleflight slot — see V7 V6-1.
		// Without this, two goroutines that both observed a miss in
		// the outer peek can each invoke load if the second arrives
		// AFTER the first's group.Do returned and forgot the key.
		if v, found, sentinelErr := peek(); found {
			if sentinelErr != nil {
				// Surface the sentinel through the Result.Err channel
				// so the caller's switch-on-err branch fires.
				var zero T
				return zero, sentinelErr
			}
			return v, nil
		}
		return load()
	})
	if err != nil {
		var zero T
		return zero, err
	}
	if v == nil {
		// load returned (nil, nil) — uncommon but possible for pointer
		// or slice T. Hand back the typed zero so the caller does not
		// need to defend against an untyped nil from the any cast.
		var zero T
		return zero, nil
	}
	out, _ := v.(T)
	return out, nil
}
