package handler

// buildCatalogue returns the curated route → spec map. Densely
// populated for the 146 unique paths the F.3.2 web sweep targets,
// PLUS every other route registered in chi.Walk gets at least a
// generic shape via defaultSpecFor — so no path goes undescribed.
//
// The map is built once at boot time and read concurrently from the
// /api/openapi.json handler.
func buildCatalogue() map[string]routeSpec {
	c := map[string]routeSpec{}

	// Curate the route catalogue in feature buckets so each block is
	// grouped, reviewable, and easy to extend when new endpoints land.
	cataloguePublic(c)
	catalogueAuth(c)
	catalogueTeam(c)
	catalogueProfile(c)
	cataloguePersonaProfiles(c)
	catalogueOrganizationShared(c)
	catalogueSearch(c)
	catalogueMessaging(c)
	catalogueProposal(c)
	catalogueJob(c)
	catalogueReview(c)
	catalogueReport(c)
	catalogueSocialLink(c)
	cataloguePortfolio(c)
	catalogueNotification(c)
	catalogueBilling(c)
	catalogueReferral(c)
	catalogueDispute(c)
	catalogueGDPR(c)
	catalogueSkill(c)
	catalogueUpload(c)
	catalogueCall(c)
	catalogueStripe(c)

	return c
}

func cataloguePublic(c map[string]routeSpec) {
	c["GET /health"] = routeSpec{
		Tags: []string{"health"}, Summary: "Liveness probe",
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /ready"] = routeSpec{
		Tags: []string{"health"}, Summary: "Readiness probe",
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/test/health-check"] = routeSpec{
		Tags: []string{"test"}, Summary: "Backend connectivity check",
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/test/words"] = routeSpec{
		Tags: []string{"test"}, Summary: "List dev fixture words",
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/test/words"] = routeSpec{
		Tags: []string{"test"}, Summary: "Add dev fixture word",
		SuccessKind: successRawJSON, SuccessStatus: "201",
	}
	c["GET /api/v1/ws"] = routeSpec{
		Tags: []string{"websocket"}, Summary: "WebSocket upgrade endpoint",
		SuccessKind: successRawJSON, SuccessStatus: "101",
	}
}

func catalogueAuth(c map[string]routeSpec) {
	c["POST /api/v1/auth/register"] = routeSpec{
		Tags: []string{"auth"}, Summary: "Register a new user",
		RequestBody: jsonRequestBody("RegisterRequest"),
		SuccessKind: successJSONRef, SuccessRef: "AuthResponse", SuccessStatus: "201",
	}
	c["POST /api/v1/auth/login"] = routeSpec{
		Tags: []string{"auth"}, Summary: "Login with email and password",
		RequestBody: jsonRequestBody("LoginRequest"),
		SuccessKind: successJSONRef, SuccessRef: "AuthResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/auth/refresh"] = routeSpec{
		Tags: []string{"auth"}, Summary: "Rotate access + refresh tokens",
		RequestBody: jsonRequestBody("RefreshRequest"),
		SuccessKind: successJSONRef, SuccessRef: "AuthResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/auth/forgot-password"] = routeSpec{
		Tags: []string{"auth"}, Summary: "Request a password reset email",
		RequestBody: rawJSONRequestBody(),
		SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/auth/reset-password"] = routeSpec{
		Tags: []string{"auth"}, Summary: "Apply a password reset",
		RequestBody: rawJSONRequestBody(),
		SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["GET /api/v1/auth/me"] = routeSpec{
		Tags: []string{"auth"}, Summary: "Current user + organization context",
		AuthRequired: true,
		SuccessKind:  successJSONRef, SuccessRef: "MeResponse", SuccessStatus: "200",
	}
	c["GET /api/v1/auth/ws-token"] = routeSpec{
		Tags: []string{"auth"}, Summary: "Fetch a short-lived WebSocket token",
		AuthRequired: true,
		SuccessKind:  successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/auth/logout"] = routeSpec{
		Tags: []string{"auth"}, Summary: "Invalidate refresh token + clear session cookie",
		AuthRequired: true,
		SuccessKind:  successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/auth/web-session"] = routeSpec{
		Tags: []string{"auth"}, Summary: "Bridge a mobile JWT into a web session cookie",
		AuthRequired: true,
		SuccessKind:  successNoContent, SuccessStatus: "204",
	}
	c["PUT /api/v1/auth/referrer-enable"] = routeSpec{
		Tags: []string{"auth"}, Summary: "Toggle the provider's referrer (apporteur) facet",
		AuthRequired: true,
		RequestBody:  rawJSONRequestBody(),
		SuccessKind:  successRawJSON, SuccessStatus: "200",
	}
}

func catalogueTeam(c map[string]routeSpec) {
	c["GET /api/v1/invitations/validate"] = routeSpec{
		Tags: []string{"team"}, Summary: "Validate an invitation token",
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/invitations/accept"] = routeSpec{
		Tags: []string{"team"}, Summary: "Accept an invitation",
		RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/organizations/role-definitions"] = routeSpec{
		Tags: []string{"team"}, Summary: "List default role definitions",
		AuthRequired: true,
		SuccessKind:  successJSONRef, SuccessRef: "RoleDefinitionsPayload", SuccessStatus: "200",
	}
	c["GET /api/v1/organizations/{orgID}/members"] = routeSpec{
		Tags: []string{"team"}, Summary: "List organization members",
		AuthRequired: true,
		SuccessKind:  successJSONRef, SuccessRef: "MemberListResponse", SuccessStatus: "200",
	}
	c["PATCH /api/v1/organizations/{orgID}/members/{userID}"] = routeSpec{
		Tags: []string{"team"}, Summary: "Update a member's role / title",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "MemberResponse", SuccessStatus: "200",
	}
	c["DELETE /api/v1/organizations/{orgID}/members/{userID}"] = routeSpec{
		Tags: []string{"team"}, Summary: "Remove a member from the organization",
		AuthRequired: true,
		SuccessKind:  successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/organizations/{orgID}/leave"] = routeSpec{
		Tags: []string{"team"}, Summary: "Leave the current organization",
		AuthRequired: true,
		SuccessKind:  successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/organizations/{orgID}/transfer"] = routeSpec{
		Tags: []string{"team"}, Summary: "Initiate ownership transfer",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "TransferResponse", SuccessStatus: "200",
	}
	c["DELETE /api/v1/organizations/{orgID}/transfer"] = routeSpec{
		Tags: []string{"team"}, Summary: "Cancel a pending ownership transfer",
		AuthRequired: true,
		SuccessKind:  successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/organizations/{orgID}/transfer/accept"] = routeSpec{
		Tags: []string{"team"}, Summary: "Accept ownership transfer (recipient)",
		AuthRequired: true,
		SuccessKind:  successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/organizations/{orgID}/transfer/decline"] = routeSpec{
		Tags: []string{"team"}, Summary: "Decline ownership transfer (recipient)",
		AuthRequired: true,
		SuccessKind:  successNoContent, SuccessStatus: "204",
	}
	c["GET /api/v1/organizations/{orgID}/role-permissions"] = routeSpec{
		Tags: []string{"team"}, Summary: "Read effective role-permissions matrix",
		AuthRequired: true,
		SuccessKind:  successRawJSON, SuccessStatus: "200",
	}
	c["PATCH /api/v1/organizations/{orgID}/role-permissions"] = routeSpec{
		Tags: []string{"team"}, Summary: "Override role-permissions matrix",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/organizations/{orgID}/invitations/"] = routeSpec{
		Tags: []string{"team"}, Summary: "List organization invitations",
		AuthRequired: true,
		SuccessKind:  successJSONList, SuccessRef: "InvitationResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/organizations/{orgID}/invitations/"] = routeSpec{
		Tags: []string{"team"}, Summary: "Send a new invitation",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "InvitationResponse", SuccessStatus: "201",
	}
	c["POST /api/v1/organizations/{orgID}/invitations/{invID}/resend"] = routeSpec{
		Tags: []string{"team"}, Summary: "Resend invitation email",
		AuthRequired: true,
		SuccessKind:  successNoContent, SuccessStatus: "204",
	}
	c["DELETE /api/v1/organizations/{orgID}/invitations/{invID}"] = routeSpec{
		Tags: []string{"team"}, Summary: "Cancel an invitation",
		AuthRequired: true,
		SuccessKind:  successNoContent, SuccessStatus: "204",
	}
}

func catalogueProfile(c map[string]routeSpec) {
	c["GET /api/v1/profile/"] = routeSpec{
		Tags: []string{"profile"}, Summary: "Get my agency profile",
		AuthRequired: true,
		SuccessKind:  successJSONRef, SuccessRef: "ProfileResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/profile/"] = routeSpec{
		Tags: []string{"profile"}, Summary: "Update my agency profile",
		AuthRequired: true, RequestBody: jsonRequestBody("UpdateProfileRequest"),
		SuccessKind: successJSONRef, SuccessRef: "ProfileResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/profile/availability"] = routeSpec{
		Tags: []string{"profile"}, Summary: "Update availability",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "ProfileResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/profile/expertise"] = routeSpec{
		Tags: []string{"profile"}, Summary: "Update expertise list",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "ProfileResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/profile/languages"] = routeSpec{
		Tags: []string{"profile"}, Summary: "Update languages",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "ProfileResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/profile/location"] = routeSpec{
		Tags: []string{"profile"}, Summary: "Update location",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "ProfileResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/profile/client"] = routeSpec{
		Tags: []string{"client-profile"}, Summary: "Update enterprise (client) profile",
		AuthRequired: true, RequestBody: jsonRequestBody("UpdateClientProfileRequest"),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/profile/skills"] = routeSpec{
		Tags: []string{"profile"}, Summary: "List my profile skills",
		AuthRequired: true,
		SuccessKind:  successJSONList, SuccessRef: "SkillResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/profile/skills"] = routeSpec{
		Tags: []string{"profile"}, Summary: "Replace my profile skills",
		AuthRequired: true, RequestBody: jsonRequestBody("PutProfileSkillsRequest"),
		SuccessKind: successJSONList, SuccessRef: "SkillResponse", SuccessStatus: "200",
	}
	c["GET /api/v1/profile/pricing"] = routeSpec{
		Tags: []string{"profile"}, Summary: "List pricing rows",
		AuthRequired: true,
		SuccessKind:  successRawJSON, SuccessStatus: "200",
	}
	c["PUT /api/v1/profile/pricing"] = routeSpec{
		Tags: []string{"profile"}, Summary: "Upsert pricing row",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["DELETE /api/v1/profile/pricing/{kind}"] = routeSpec{
		Tags: []string{"profile"}, Summary: "Delete a pricing row by kind",
		AuthRequired: true,
		SuccessKind:  successNoContent, SuccessStatus: "204",
	}
	c["GET /api/v1/profiles/{orgId}"] = routeSpec{
		Tags: []string{"profile"}, Summary: "Public profile by org id",
		SuccessKind: successJSONRef, SuccessRef: "ProfileResponse", SuccessStatus: "200",
	}
	c["GET /api/v1/profiles/search"] = routeSpec{
		Tags: []string{"profile"}, Summary: "Legacy SQL profile search (referral picker)",
		AuthRequired: true,
		SuccessKind:  successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/clients/{orgId}"] = routeSpec{
		Tags: []string{"client-profile"}, Summary: "Public client (enterprise) profile",
		SuccessKind: successJSONRef, SuccessRef: "PublicClientProfileResponse", SuccessStatus: "200",
	}
	c["GET /api/v1/profiles/{orgId}/project-history"] = routeSpec{
		Tags: []string{"portfolio"}, Summary: "Read public project history",
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
}

func cataloguePersonaProfiles(c map[string]routeSpec) {
	// Freelance
	c["GET /api/v1/freelance-profile/"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Get my freelance profile",
		AuthRequired: true,
		SuccessKind:  successJSONRef, SuccessRef: "FreelanceProfileResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/freelance-profile/"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Update my freelance profile",
		AuthRequired: true, RequestBody: jsonRequestBody("UpdateFreelanceProfileRequest"),
		SuccessKind: successJSONRef, SuccessRef: "FreelanceProfileResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/freelance-profile/availability"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Update freelance availability",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "FreelanceProfileResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/freelance-profile/expertise"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Update freelance expertise",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "FreelanceProfileResponse", SuccessStatus: "200",
	}
	c["GET /api/v1/freelance-profile/pricing"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Read freelance pricing",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["PUT /api/v1/freelance-profile/pricing"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Upsert freelance pricing",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["DELETE /api/v1/freelance-profile/pricing"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Delete freelance pricing",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["GET /api/v1/freelance-profile/social-links/"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "List freelance social links",
		AuthRequired: true, SuccessKind: successJSONList, SuccessRef: "SocialLinkResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/freelance-profile/social-links/"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Replace freelance social links",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONList, SuccessRef: "SocialLinkResponse", SuccessStatus: "200",
	}
	c["DELETE /api/v1/freelance-profile/social-links/{platform}"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Delete freelance social link",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/freelance-profile/video"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Set freelance intro video URL",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["DELETE /api/v1/freelance-profile/video"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Delete freelance intro video",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["GET /api/v1/freelance-profiles/{orgID}"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Public freelance profile by org id",
		SuccessKind: successJSONRef, SuccessRef: "FreelanceProfileResponse", SuccessStatus: "200",
	}
	c["GET /api/v1/freelance-profiles/{orgId}/social-links"] = routeSpec{
		Tags: []string{"freelance-profile"}, Summary: "Public freelance social links",
		SuccessKind: successJSONList, SuccessRef: "SocialLinkResponse", SuccessStatus: "200",
	}

	// Referrer
	c["GET /api/v1/referrer-profile/"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Get my referrer profile",
		AuthRequired: true,
		SuccessKind:  successRawJSON, SuccessStatus: "200",
	}
	c["PUT /api/v1/referrer-profile/"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Update my referrer profile",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["PUT /api/v1/referrer-profile/availability"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Update referrer availability",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["PUT /api/v1/referrer-profile/expertise"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Update referrer expertise",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/referrer-profile/pricing"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Read referrer pricing",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["PUT /api/v1/referrer-profile/pricing"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Upsert referrer pricing",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["DELETE /api/v1/referrer-profile/pricing"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Delete referrer pricing",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["GET /api/v1/referrer-profile/social-links/"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "List referrer social links",
		AuthRequired: true, SuccessKind: successJSONList, SuccessRef: "SocialLinkResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/referrer-profile/social-links/"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Replace referrer social links",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONList, SuccessRef: "SocialLinkResponse", SuccessStatus: "200",
	}
	c["DELETE /api/v1/referrer-profile/social-links/{platform}"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Delete referrer social link",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/referrer-profile/video"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Set referrer intro video URL",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["DELETE /api/v1/referrer-profile/video"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Delete referrer intro video",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["GET /api/v1/referrer-profiles/{orgID}"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Public referrer profile by org id",
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/referrer-profiles/{orgID}/reputation"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Public referrer reputation aggregate",
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/referrer-profiles/{orgId}/social-links"] = routeSpec{
		Tags: []string{"referrer-profile"}, Summary: "Public referrer social links",
		SuccessKind: successJSONList, SuccessRef: "SocialLinkResponse", SuccessStatus: "200",
	}
}
