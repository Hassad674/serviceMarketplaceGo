package service

import (
	"context"

	"github.com/google/uuid"
)

// SessionVersionInvalidator drops the cached session_version entry
// for a given user. It is consumed by every code path that mutates
// users.session_version (every BumpSessionVersion call site) so the
// revocation propagates on the very next request instead of waiting
// for the 30s Redis TTL.
//
// QW-HARDENING: this port was added to close the second leak left by
// QW1/QW2 — the cache decorator already exposes Invalidate, but
// nothing called it. The decorator on the user repository
// (postgres.NewInvalidatingUserRepository) now wires every successful
// BumpSessionVersion call to this hook automatically, so no service
// has to remember to invalidate. The interface stays in port/service
// for the rare callers that want to invalidate explicitly (admin
// tools, integration tests). Missing keys are NEVER errors.
type SessionVersionInvalidator interface {
	Invalidate(ctx context.Context, userID uuid.UUID) error
}

// OrgOverridesInvalidator drops the cached role_overrides JSONB entry
// for a given organization. It is consumed by the role-permissions
// editor service after a successful SaveRoleOverrides so the new
// permission matrix takes effect on the very next request.
//
// QW-HARDENING: same rationale as SessionVersionInvalidator — the
// cache decorator already exposes Invalidate, this port wires
// callers to it through the standard port/adapter contract instead
// of crossing the layering boundary by importing
// adapter/redis directly from app/. Missing keys are NEVER errors.
type OrgOverridesInvalidator interface {
	Invalidate(ctx context.Context, orgID uuid.UUID) error
}
