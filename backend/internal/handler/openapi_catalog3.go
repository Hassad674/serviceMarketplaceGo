package handler

// Catalogue continuation — billing, referral, dispute, GDPR, skill,
// upload, call, stripe, admin. Final third of the curated map.

func catalogueBilling(c map[string]routeSpec) {
	c["GET /api/v1/billing/fee-preview"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Preview platform fee for a given amount",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/wallet/"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Read wallet balance and pending operations",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/wallet/payout"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Initiate a wallet payout",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/wallet/transfers/{record_id}/retry"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Retry a failed wallet transfer",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/me/invoices"] = routeSpec{
		Tags: []string{"billing"}, Summary: "List my invoices (transactions + commission)",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/me/invoices/{id}/pdf"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Download invoice PDF",
		AuthRequired: true, SuccessKind: successPDF, SuccessStatus: "200",
	}
	c["GET /api/v1/me/invoicing/current-month"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Aggregate of current-month invoicing",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/me/billing-profile/"] = routeSpec{
		Tags: []string{"billing-profile"}, Summary: "Read my billing profile",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["PUT /api/v1/me/billing-profile/"] = routeSpec{
		Tags: []string{"billing-profile"}, Summary: "Update my billing profile",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/me/billing-profile/sync-from-stripe"] = routeSpec{
		Tags: []string{"billing-profile"}, Summary: "Pull billing fields from Stripe",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/me/billing-profile/validate-vat"] = routeSpec{
		Tags: []string{"billing-profile"}, Summary: "Validate the stored VAT number",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/subscriptions/me"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Read my subscription",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/subscriptions/me/cycle-preview"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Preview an upcoming billing cycle",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/subscriptions/me/stats"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Subscription usage stats",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/subscriptions/portal"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Stripe billing portal redirect URL",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/subscriptions/"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Subscribe / change plan",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["PATCH /api/v1/subscriptions/me/auto-renew"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Toggle subscription auto-renew",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["PATCH /api/v1/subscriptions/me/billing-cycle"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Switch monthly / yearly billing cycle",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/payment-info/account-status"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Read Stripe Connect KYC status",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/payment-info/account-session"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Open Stripe Embedded Account Session",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["DELETE /api/v1/payment-info/account-session"] = routeSpec{
		Tags: []string{"billing"}, Summary: "Close Stripe Embedded Account Session",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
}

func catalogueReferral(c map[string]routeSpec) {
	c["POST /api/v1/referrals/"] = routeSpec{
		Tags: []string{"referral"}, Summary: "Create a referral",
		AuthRequired: true, RequestBody: jsonRequestBody("CreateReferralRequest"),
		SuccessKind: successJSONRef, SuccessRef: "ReferralResponse", SuccessStatus: "201",
	}
	c["GET /api/v1/referrals/me"] = routeSpec{
		Tags: []string{"referral"}, Summary: "List referrals I sent (apporteur)",
		AuthRequired: true, SuccessKind: successJSONList, SuccessRef: "ReferralResponse", SuccessStatus: "200",
	}
	c["GET /api/v1/referrals/incoming"] = routeSpec{
		Tags: []string{"referral"}, Summary: "List referrals I received (provider)",
		AuthRequired: true, SuccessKind: successJSONList, SuccessRef: "ReferralResponse", SuccessStatus: "200",
	}
	c["GET /api/v1/referrals/{id}"] = routeSpec{
		Tags: []string{"referral"}, Summary: "Read a referral",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "ReferralResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/referrals/{id}/respond"] = routeSpec{
		Tags: []string{"referral"}, Summary: "Respond to a received referral",
		AuthRequired: true, RequestBody: jsonRequestBody("RespondReferralRequest"),
		SuccessKind: successJSONRef, SuccessRef: "ReferralResponse", SuccessStatus: "200",
	}
	c["GET /api/v1/referrals/{id}/attributions"] = routeSpec{
		Tags: []string{"referral"}, Summary: "List attributed proposals for a referral",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/referrals/{id}/commissions"] = routeSpec{
		Tags: []string{"referral"}, Summary: "List paid commissions for a referral",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/referrals/{id}/negotiations"] = routeSpec{
		Tags: []string{"referral"}, Summary: "List commission negotiation rounds",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
}

func catalogueDispute(c map[string]routeSpec) {
	c["POST /api/v1/disputes/"] = routeSpec{
		Tags: []string{"dispute"}, Summary: "Open a dispute",
		AuthRequired: true, RequestBody: jsonRequestBody("OpenDisputeRequest"),
		SuccessKind: successJSONRef, SuccessRef: "DisputeResponse", SuccessStatus: "201",
	}
	c["GET /api/v1/disputes/mine"] = routeSpec{
		Tags: []string{"dispute"}, Summary: "List my disputes",
		AuthRequired: true, SuccessKind: successJSONList, SuccessRef: "DisputeResponse", SuccessStatus: "200",
	}
	c["GET /api/v1/disputes/{id}"] = routeSpec{
		Tags: []string{"dispute"}, Summary: "Read a dispute",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "DisputeResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/disputes/{id}/cancel"] = routeSpec{
		Tags: []string{"dispute"}, Summary: "Request cancellation of a dispute",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "DisputeResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/disputes/{id}/cancellation/respond"] = routeSpec{
		Tags: []string{"dispute"}, Summary: "Respond to a cancellation request",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "DisputeResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/disputes/{id}/counter-propose"] = routeSpec{
		Tags: []string{"dispute"}, Summary: "Submit a counter-proposal",
		AuthRequired: true, RequestBody: jsonRequestBody("CounterProposeRequest"),
		SuccessKind: successJSONRef, SuccessRef: "DisputeResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/disputes/{id}/counter-proposals/{cpId}/respond"] = routeSpec{
		Tags: []string{"dispute"}, Summary: "Respond to a counter-proposal",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "DisputeResponse", SuccessStatus: "200",
	}
}

func catalogueGDPR(c map[string]routeSpec) {
	c["POST /api/v1/me/account/request-deletion"] = routeSpec{
		Tags: []string{"gdpr"}, Summary: "Request account deletion (RGPD)",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "202",
	}
	c["GET /api/v1/me/account/confirm-deletion"] = routeSpec{
		Tags: []string{"gdpr"}, Summary: "Confirm account deletion via emailed token",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/me/account/cancel-deletion"] = routeSpec{
		Tags: []string{"gdpr"}, Summary: "Cancel a pending account deletion",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/me/export"] = routeSpec{
		Tags: []string{"gdpr"}, Summary: "Export my personal data (RGPD right of access)",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
}

func catalogueSkill(c map[string]routeSpec) {
	c["GET /api/v1/skills/catalog"] = routeSpec{
		Tags: []string{"skill"}, Summary: "List the curated skill catalog",
		SuccessKind: successJSONList, SuccessRef: "SkillResponse", SuccessStatus: "200",
	}
	c["GET /api/v1/skills/autocomplete"] = routeSpec{
		Tags: []string{"skill"}, Summary: "Autocomplete skill names",
		SuccessKind: successJSONList, SuccessRef: "SkillResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/skills"] = routeSpec{
		Tags: []string{"skill"}, Summary: "Suggest a new skill (auto-curates after threshold)",
		AuthRequired: true, RequestBody: jsonRequestBody("CreateSkillRequest"),
		SuccessKind: successJSONRef, SuccessRef: "SkillResponse", SuccessStatus: "201",
	}
}

func catalogueUpload(c map[string]routeSpec) {
	for _, ep := range []struct {
		method, route, summary string
	}{
		{"POST", "/api/v1/upload/photo", "Upload a profile photo"},
		{"POST", "/api/v1/upload/video", "Upload an intro video"},
		{"POST", "/api/v1/upload/portfolio-image", "Upload a portfolio image"},
		{"POST", "/api/v1/upload/portfolio-video", "Upload a portfolio video"},
		{"POST", "/api/v1/upload/referrer-video", "Upload referrer intro video"},
		{"POST", "/api/v1/upload/review-video", "Upload a review video"},
	} {
		c[ep.method+" "+ep.route] = routeSpec{
			Tags: []string{"upload"}, Summary: ep.summary,
			AuthRequired: true, RequestBody: multipartFormRequestBody(),
			SuccessKind: successRawJSON, SuccessStatus: "200",
		}
	}
	c["DELETE /api/v1/upload/video"] = routeSpec{
		Tags: []string{"upload"}, Summary: "Delete uploaded intro video",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["DELETE /api/v1/upload/referrer-video"] = routeSpec{
		Tags: []string{"upload"}, Summary: "Delete referrer intro video upload",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
}

func catalogueCall(c map[string]routeSpec) {
	c["POST /api/v1/calls/initiate"] = routeSpec{
		Tags: []string{"call"}, Summary: "Initiate a LiveKit call",
		AuthRequired: true, RequestBody: jsonRequestBody("InitiateCallRequest"),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/calls/{id}/accept"] = routeSpec{
		Tags: []string{"call"}, Summary: "Accept a call",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/calls/{id}/decline"] = routeSpec{
		Tags: []string{"call"}, Summary: "Decline a call",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/calls/{id}/end"] = routeSpec{
		Tags: []string{"call"}, Summary: "End a call",
		AuthRequired: true, RequestBody: jsonRequestBody("EndCallRequest"),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
}

func catalogueStripe(c map[string]routeSpec) {
	c["GET /api/v1/stripe/config"] = routeSpec{
		Tags: []string{"stripe"}, Summary: "Read frontend Stripe configuration",
		AuthRequired: true,
		SuccessKind:  successJSONRef, SuccessRef: "StripeConfigResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/stripe/webhook"] = routeSpec{
		Tags: []string{"stripe"}, Summary: "Stripe webhook ingestion",
		RequestBody: rawJSONRequestBody(),
		SuccessKind: successNoContent, SuccessStatus: "204",
	}
}
