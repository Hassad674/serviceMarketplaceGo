package main

import (
	"marketplace-backend/internal/app/security"
	"marketplace-backend/internal/handler"
	"marketplace-backend/internal/port/repository"
)

// securityDeps captures the upstream resources the security activity
// feature reads from. The feature is a thin filter over the
// audit_logs table — only the AuditRepository is required.
type securityDeps struct {
	AuditRepo repository.AuditRepository
}

// securityWiring carries the products of the security feature
// initialisation. Nil-valued fields keep the rest of the backend
// booting on minimal builds (e.g. when the audit log is not wired).
type securityWiring struct {
	Handler *handler.SecurityHandler
}

// wireSecurity brings up the /me/security/activity endpoint.
//
// Returns a nil handler (and so the route stays unmounted) when the
// audit repository is missing — the rest of the API keeps booting and
// the security tab simply does not appear. Production wiring always
// passes a non-nil AuditRepo so the endpoint is live.
func wireSecurity(deps securityDeps) securityWiring {
	if deps.AuditRepo == nil {
		return securityWiring{}
	}
	svc := security.NewService(deps.AuditRepo)
	if svc == nil {
		return securityWiring{}
	}
	return securityWiring{Handler: handler.NewSecurityHandler(svc)}
}
