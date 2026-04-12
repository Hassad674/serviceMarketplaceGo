package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/organization"
)

// ok200 is a trivial handler that proves the middleware passed through.
var ok200 = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func reqWithOrgRole(role string) *http.Request {
	ctx := context.WithValue(context.Background(), ContextKeyOrgRole, role)
	return httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
}

func TestRequirePermission_OwnerGetsEverything(t *testing.T) {
	perms := []organization.Permission{
		organization.PermJobsCreate, organization.PermJobsDelete,
		organization.PermMessagingSend, organization.PermWalletWithdraw,
		organization.PermOrgDelete, organization.PermKYCManage,
		organization.PermTeamTransferOwner, organization.PermBillingManage,
	}
	for _, perm := range perms {
		t.Run(string(perm), func(t *testing.T) {
			rec := httptest.NewRecorder()
			RequirePermission(perm)(ok200).ServeHTTP(rec, reqWithOrgRole("owner"))
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestRequirePermission_ViewerReadOnly(t *testing.T) {
	allowed := []organization.Permission{
		organization.PermJobsView, organization.PermProposalsView,
		organization.PermMessagingView, organization.PermWalletView,
		organization.PermTeamView,
	}
	for _, perm := range allowed {
		t.Run("allowed_"+string(perm), func(t *testing.T) {
			rec := httptest.NewRecorder()
			RequirePermission(perm)(ok200).ServeHTTP(rec, reqWithOrgRole("viewer"))
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}

	denied := []organization.Permission{
		organization.PermJobsCreate, organization.PermJobsEdit, organization.PermJobsDelete,
		organization.PermProposalsCreate, organization.PermProposalsRespond,
		organization.PermMessagingSend,
		organization.PermWalletWithdraw,
		organization.PermOrgProfileEdit,
		organization.PermTeamInvite, organization.PermTeamManage,
		organization.PermBillingView, organization.PermBillingManage,
		organization.PermOrgDelete, organization.PermKYCManage,
		organization.PermReviewsRespond,
	}
	for _, perm := range denied {
		t.Run("denied_"+string(perm), func(t *testing.T) {
			rec := httptest.NewRecorder()
			RequirePermission(perm)(ok200).ServeHTTP(rec, reqWithOrgRole("viewer"))
			assert.Equal(t, http.StatusForbidden, rec.Code)
			var body map[string]string
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, "permission_denied", body["error"])
		})
	}
}

func TestRequirePermission_MemberDailyOps(t *testing.T) {
	allowed := []organization.Permission{
		organization.PermJobsCreate, organization.PermJobsEdit,
		organization.PermProposalsCreate, organization.PermProposalsRespond,
		organization.PermMessagingSend, organization.PermWalletView,
	}
	for _, perm := range allowed {
		t.Run("allowed_"+string(perm), func(t *testing.T) {
			rec := httptest.NewRecorder()
			RequirePermission(perm)(ok200).ServeHTTP(rec, reqWithOrgRole("member"))
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}

	denied := []organization.Permission{
		organization.PermJobsDelete, organization.PermWalletWithdraw,
		organization.PermTeamInvite, organization.PermTeamManage,
		organization.PermOrgProfileEdit, organization.PermBillingManage,
		organization.PermOrgDelete, organization.PermKYCManage,
	}
	for _, perm := range denied {
		t.Run("denied_"+string(perm), func(t *testing.T) {
			rec := httptest.NewRecorder()
			RequirePermission(perm)(ok200).ServeHTTP(rec, reqWithOrgRole("member"))
			assert.Equal(t, http.StatusForbidden, rec.Code)
		})
	}
}

func TestRequirePermission_AdminNoFinance(t *testing.T) {
	denied := []organization.Permission{
		organization.PermWalletWithdraw, organization.PermOrgDelete,
		organization.PermTeamTransferOwner, organization.PermKYCManage,
		organization.PermBillingManage,
	}
	for _, perm := range denied {
		t.Run("denied_"+string(perm), func(t *testing.T) {
			rec := httptest.NewRecorder()
			RequirePermission(perm)(ok200).ServeHTTP(rec, reqWithOrgRole("admin"))
			assert.Equal(t, http.StatusForbidden, rec.Code)
		})
	}
}

func TestRequirePermission_NoOrg(t *testing.T) {
	// Empty orgRole — user has no organization
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil) // no ContextKeyOrgRole
	RequirePermission(organization.PermJobsView)(ok200).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "no_organization", body["error"])
}

func TestRequirePermission_UnknownRole(t *testing.T) {
	rec := httptest.NewRecorder()
	RequirePermission(organization.PermJobsView)(ok200).ServeHTTP(rec, reqWithOrgRole("hacker"))

	assert.Equal(t, http.StatusForbidden, rec.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "permission_denied", body["error"])
}

// reqWithOrgRoleAndPermissions builds a request that carries both an
// orgRole and the pre-resolved effective permission set — the shape
// the middleware expects once a session is created under R17.
func reqWithOrgRoleAndPermissions(role string, perms []string) *http.Request {
	ctx := context.WithValue(context.Background(), ContextKeyOrgRole, role)
	ctx = context.WithValue(ctx, ContextKeyPermissions, perms)
	return httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
}

// TestRequirePermission_HonorsSessionPermissions verifies that the
// middleware reads the customized permission set from the context
// instead of falling back to the static HasPermission lookup when
// the session carries a permissions list.
func TestRequirePermission_HonorsSessionPermissions(t *testing.T) {
	// Member does NOT have jobs.delete in the static defaults, but the
	// session carries a customized permissions list that explicitly
	// grants it. The middleware must let the request through.
	customized := []string{
		string(organization.PermJobsView),
		string(organization.PermJobsCreate),
		string(organization.PermJobsEdit),
		string(organization.PermJobsDelete), // override grant
	}

	rec := httptest.NewRecorder()
	req := reqWithOrgRoleAndPermissions("member", customized)
	RequirePermission(organization.PermJobsDelete)(ok200).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code,
		"middleware must honor session-embedded permission grant")
}

// TestRequirePermission_SessionRevocationRespected verifies that when
// the session permissions list REVOKES a permission the role would
// normally have, the middleware denies the request.
func TestRequirePermission_SessionRevocationRespected(t *testing.T) {
	// Member normally has messaging.send by default. The session
	// carries a reduced list that revokes it. The middleware must
	// block the request.
	reducedPerms := []string{
		string(organization.PermJobsView),
		string(organization.PermMessagingView),
		// no messaging.send
	}

	rec := httptest.NewRecorder()
	req := reqWithOrgRoleAndPermissions("member", reducedPerms)
	RequirePermission(organization.PermMessagingSend)(ok200).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code,
		"middleware must honor session-embedded permission revocation")
}

// TestRequirePermission_EmptySessionPermissionsFallsBackToRole verifies
// that when the session has NO permissions list, the middleware uses
// the static role-based lookup (legacy fallback).
func TestRequirePermission_EmptySessionPermissionsFallsBackToRole(t *testing.T) {
	// No ContextKeyPermissions set — middleware should fall back to
	// the role map.
	rec := httptest.NewRecorder()
	req := reqWithOrgRole("member")
	RequirePermission(organization.PermJobsCreate)(ok200).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
