package auth

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/port/service"
)

// B.8 — refresh-token chain hardening constants.
//
// These limits cap the cost of a successful refresh-token theft AND
// of accidental abuse (a runaway client that loops on /auth/refresh).
//
// Numbers chosen for the marketplace's typical session shape:
//
//   - MaxRefreshChainDepth = 1000. With a 7-day refresh TTL and the
//     mobile client refreshing at most once per access-token TTL
//     (15 min), a single device produces ~672 rotations per chain
//     before any natural relog. 1000 leaves comfortable headroom for
//     proactive refreshes (background sync, app warm-up) without
//     allowing an attacker to use the chain as an effectively
//     unbounded persistence mechanism.
//
//   - MaxFamilyAge = 30 * 24h. Even with rotation, the absolute
//     session lifetime is capped at 30 days from initial login. This
//     forces a real password re-authentication on a regular cadence
//     so leaked credentials cannot grant indefinite access. The cap
//     is independent of per-token TTL: even if every rotation
//     refreshes the per-token expiry, the family root's IAT does not
//     move and is the upper bound checked here.
const (
	MaxRefreshChainDepth = 1000
	MaxFamilyAge         = 30 * 24 * time.Hour
)

// chainLimitDecision is the outcome of evaluating a refresh token's
// lineage against the chain caps.
type chainLimitDecision struct {
	// rejected is true when the token must be refused (force re-login).
	rejected bool

	// reason is the snake_case reason code recorded in the audit row
	// when rejected; empty on accept.
	reason string

	// chainDepth is the depth read from the token (echoed for the
	// audit row when rejected, ignored when accepted).
	chainDepth int

	// familyAge is the elapsed time since the family root was issued
	// (echoed for the audit row when rejected). Zero when the token
	// has no family-root IAT recorded (legacy).
	familyAge time.Duration
}

// evaluateChainLimits compares the validated refresh token's lineage
// against MaxRefreshChainDepth and MaxFamilyAge. Returns a decision
// that the caller turns into an audit emission + 401.
//
// Tokens minted before B.8 (no FamilyRootIAT) bypass the family-age
// check — we cannot retroactively know when their chain started.
// They are still subject to the depth check (which defaults to 0
// for legacy tokens, so it never triggers in practice). On the
// NEXT rotation, those tokens are re-rooted by the JWT adapter
// (FamilyRootJTI = own JTI, FamilyRootIAT = now), so the absolute
// lifetime cap kicks in from that point forward.
func evaluateChainLimits(claims *service.TokenClaims, now time.Time) chainLimitDecision {
	if claims == nil {
		return chainLimitDecision{}
	}
	if claims.ChainDepth >= MaxRefreshChainDepth {
		return chainLimitDecision{
			rejected:   true,
			reason:     "chain_depth_exceeded",
			chainDepth: claims.ChainDepth,
		}
	}
	if !claims.FamilyRootIAT.IsZero() {
		age := now.Sub(claims.FamilyRootIAT)
		if age >= MaxFamilyAge {
			return chainLimitDecision{
				rejected:   true,
				reason:     "family_age_exceeded",
				chainDepth: claims.ChainDepth,
				familyAge:  age,
			}
		}
	}
	return chainLimitDecision{
		chainDepth: claims.ChainDepth,
		familyAge:  ageOrZero(claims.FamilyRootIAT, now),
	}
}

// ageOrZero returns now-iat for a non-zero iat, zero otherwise.
func ageOrZero(iat time.Time, now time.Time) time.Duration {
	if iat.IsZero() {
		return 0
	}
	return now.Sub(iat)
}

// invalidateFamily blacklists every JTI currently recorded under the
// family root, then deletes the family set. Returns the number of
// descendant JTIs that were blacklisted (zero on missing family or
// Redis failure). Always best-effort — callers should NOT depend on
// the return value for correctness; the recordTokenReuse audit row
// is the authoritative SOC trail.
//
// Adopts a generous blacklist TTL (MaxFamilyAge) for every member
// regardless of its individual remaining expiry — a rotated token
// could have a TTL nearly identical to the family root, and an
// attacker holding a subset would otherwise see entries evict before
// the legitimate user's session even ends.
func (s *Service) invalidateFamily(ctx context.Context, familyRootJTI string) int {
	if s.refreshBlacklist == nil || familyRootJTI == "" {
		return 0
	}
	members, err := s.refreshBlacklist.FamilyMembers(ctx, familyRootJTI)
	if err != nil {
		slog.Warn("refresh family read failed",
			"family_root_jti", familyRootJTI, "error", err)
		return 0
	}
	count := 0
	for _, jti := range members {
		if jti == "" {
			continue
		}
		if err := s.refreshBlacklist.Add(ctx, jti, MaxFamilyAge); err != nil {
			slog.Warn("refresh family blacklist propagate failed",
				"family_root_jti", familyRootJTI, "jti", jti, "error", err)
			continue
		}
		count++
	}
	// Also blacklist the root itself — it may not be a "member" if
	// the chain has only ever rotated and never re-presented the root.
	if err := s.refreshBlacklist.Add(ctx, familyRootJTI, MaxFamilyAge); err != nil {
		slog.Warn("refresh family blacklist root failed",
			"family_root_jti", familyRootJTI, "error", err)
	}
	if err := s.refreshBlacklist.DeleteFamily(ctx, familyRootJTI); err != nil {
		slog.Warn("refresh family delete failed",
			"family_root_jti", familyRootJTI, "error", err)
	}
	return count
}

// recordChainLimitRejected writes an audit row capturing a chain-limit
// rejection (depth or absolute-age cap hit). Distinct from a reuse
// detection — chain limits trip on a still-legitimate token that has
// simply outlived its allotted lifetime / depth, so the action is
// `auth.token_refresh` failure (we keep token_reuse_detected
// reserved for actual replay forensics).
func (s *Service) recordChainLimitRejected(ctx context.Context, userID uuid.UUID, jti string, decision chainLimitDecision) {
	if s.audits == nil {
		return
	}
	uid := userID
	entry, err := audit.NewEntry(audit.NewEntryInput{
		UserID:       &uid,
		Action:       audit.ActionTokenRefresh,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &uid,
		Metadata: map[string]any{
			"jti":                jti,
			"reason":             decision.reason,
			"chain_depth":        decision.chainDepth,
			"family_age_seconds": int(decision.familyAge.Seconds()),
			"outcome":            "rejected",
		},
	})
	if err != nil {
		slog.Warn("audit: build chain_limit_rejected entry failed", "error", err)
		return
	}
	if err := s.audits.Log(ctx, entry); err != nil {
		slog.Warn("audit: insert chain_limit_rejected failed", "error", err)
	}
}

// recordTokenReuseWithFamily extends recordTokenReuse with the family
// metadata required by B.8 (family_root_jti, descendants_invalidated_count,
// chain_depth, family_age_seconds). Falls back to no-op when the audits
// repo is unwired.
func (s *Service) recordTokenReuseWithFamily(
	ctx context.Context,
	userID uuid.UUID,
	claims *service.TokenClaims,
	descendantsInvalidated int,
) {
	if s.audits == nil {
		return
	}
	uid := userID
	familyAge := time.Duration(0)
	if !claims.FamilyRootIAT.IsZero() {
		familyAge = time.Since(claims.FamilyRootIAT)
	}
	entry, err := audit.NewEntry(audit.NewEntryInput{
		UserID:       &uid,
		Action:       audit.ActionTokenReuseDetected,
		ResourceType: audit.ResourceTypeUser,
		ResourceID:   &uid,
		Metadata: map[string]any{
			"jti":                            claims.JTI,
			"compromised_jti":                claims.JTI,
			"family_root_jti":                claims.FamilyRootJTI,
			"chain_depth":                    claims.ChainDepth,
			"family_age_seconds":             int(familyAge.Seconds()),
			"descendants_invalidated_count":  descendantsInvalidated,
		},
	})
	if err != nil {
		slog.Warn("audit: build token_reuse_detected entry failed", "error", err)
		return
	}
	if err := s.audits.Log(ctx, entry); err != nil {
		slog.Warn("audit: insert token_reuse_detected failed", "error", err)
	}
}
