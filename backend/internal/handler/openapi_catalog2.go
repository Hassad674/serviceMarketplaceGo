package handler

// Catalogue continuation — second-half features. Split from
// openapi_catalog.go so each file stays comfortably under the
// 600-line ceiling.

func catalogueOrganizationShared(c map[string]routeSpec) {
	c["GET /api/v1/organization/shared"] = routeSpec{
		Tags: []string{"organization-shared"}, Summary: "Read shared org-level fields (location, languages, photo)",
		AuthRequired: true,
		SuccessKind:  successJSONRef, SuccessRef: "OrganizationSharedProfileResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/organization/location"] = routeSpec{
		Tags: []string{"organization-shared"}, Summary: "Update org location",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["PUT /api/v1/organization/languages"] = routeSpec{
		Tags: []string{"organization-shared"}, Summary: "Update org languages",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["PUT /api/v1/organization/photo"] = routeSpec{
		Tags: []string{"organization-shared"}, Summary: "Update org photo URL",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
}

func catalogueSearch(c map[string]routeSpec) {
	c["GET /api/v1/search"] = routeSpec{
		Tags: []string{"search"}, Summary: "Hybrid Typesense search",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/search/key"] = routeSpec{
		Tags: []string{"search"}, Summary: "Scoped Typesense API key",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/search/track"] = routeSpec{
		Tags: []string{"search"}, Summary: "Track search click-through",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
}

func catalogueMessaging(c map[string]routeSpec) {
	c["GET /api/v1/messaging/conversations"] = routeSpec{
		Tags: []string{"messaging"}, Summary: "List my conversations",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/messaging/conversations"] = routeSpec{
		Tags: []string{"messaging"}, Summary: "Open a new conversation",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "ConversationResponse", SuccessStatus: "201",
	}
	c["GET /api/v1/messaging/conversations/{id}/messages"] = routeSpec{
		Tags: []string{"messaging"}, Summary: "Read messages in a conversation",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/messaging/conversations/{id}/messages"] = routeSpec{
		Tags: []string{"messaging"}, Summary: "Send a message",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "MessageResponse", SuccessStatus: "201",
	}
	c["POST /api/v1/messaging/conversations/{id}/read"] = routeSpec{
		Tags: []string{"messaging"}, Summary: "Mark conversation as read",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/messaging/upload-url"] = routeSpec{
		Tags: []string{"messaging"}, Summary: "Get a presigned upload URL for an attachment",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/messaging/unread-count"] = routeSpec{
		Tags: []string{"messaging"}, Summary: "Get total unread message count",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["PUT /api/v1/messaging/messages/{id}"] = routeSpec{
		Tags: []string{"messaging"}, Summary: "Edit a message",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "MessageResponse", SuccessStatus: "200",
	}
	c["DELETE /api/v1/messaging/messages/{id}"] = routeSpec{
		Tags: []string{"messaging"}, Summary: "Delete a message",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
}

func catalogueProposal(c map[string]routeSpec) {
	c["GET /api/v1/projects/"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "List my projects (paid proposals view)",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Create a proposal",
		AuthRequired: true, RequestBody: jsonRequestBody("CreateProposalRequest"),
		SuccessKind: successJSONRef, SuccessRef: "ProposalResponse", SuccessStatus: "201",
	}
	c["GET /api/v1/proposals/{id}"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Read a proposal",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "ProposalResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/accept"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Accept a proposal",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "ProposalResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/decline"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Decline a proposal",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "ProposalResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/cancel"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Cancel a proposal",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "ProposalResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/modify"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Modify proposal terms",
		AuthRequired: true, RequestBody: jsonRequestBody("ModifyProposalRequest"),
		SuccessKind: successJSONRef, SuccessRef: "ProposalResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/pay"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Pay for a proposal (single-payment)",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/confirm-payment"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Confirm payment intent succeeded",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/complete"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Mark proposal as completed (deprecated single-action)",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "ProposalResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/request-completion"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Request completion (provider)",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "ProposalResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/reject-completion"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Reject completion (client)",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "ProposalResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/milestones/{mid}/fund"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Fund a milestone",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/milestones/{mid}/submit"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Submit milestone for approval",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "ProposalResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/milestones/{mid}/approve"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Approve a submitted milestone",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "ProposalResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/proposals/{id}/milestones/{mid}/reject"] = routeSpec{
		Tags: []string{"proposal"}, Summary: "Reject a submitted milestone",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "ProposalResponse", SuccessStatus: "200",
	}
}

func catalogueJob(c map[string]routeSpec) {
	c["GET /api/v1/jobs/open"] = routeSpec{
		Tags: []string{"job"}, Summary: "List open jobs (public marketplace feed)",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/jobs/mine"] = routeSpec{
		Tags: []string{"job"}, Summary: "List my published jobs",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/jobs/credits"] = routeSpec{
		Tags: []string{"job"}, Summary: "Read remaining job-application credits",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/jobs/"] = routeSpec{
		Tags: []string{"job"}, Summary: "Publish a new job",
		AuthRequired: true, RequestBody: jsonRequestBody("CreateJobRequest"),
		SuccessKind: successJSONRef, SuccessRef: "JobResponse", SuccessStatus: "201",
	}
	c["GET /api/v1/jobs/{id}"] = routeSpec{
		Tags: []string{"job"}, Summary: "Read a job",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "JobResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/jobs/{id}"] = routeSpec{
		Tags: []string{"job"}, Summary: "Update a job",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "JobResponse", SuccessStatus: "200",
	}
	c["DELETE /api/v1/jobs/{id}"] = routeSpec{
		Tags: []string{"job"}, Summary: "Delete a job",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/jobs/{id}/close"] = routeSpec{
		Tags: []string{"job"}, Summary: "Close a job",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "JobResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/jobs/{id}/reopen"] = routeSpec{
		Tags: []string{"job"}, Summary: "Reopen a closed job",
		AuthRequired: true, SuccessKind: successJSONRef, SuccessRef: "JobResponse", SuccessStatus: "200",
	}
	c["POST /api/v1/jobs/{id}/mark-viewed"] = routeSpec{
		Tags: []string{"job"}, Summary: "Mark a job as viewed (telemetry)",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/jobs/{id}/apply"] = routeSpec{
		Tags: []string{"job"}, Summary: "Apply to a job",
		AuthRequired: true, RequestBody: jsonRequestBody("ApplyToJobRequest"),
		SuccessKind: successRawJSON, SuccessStatus: "201",
	}
	c["GET /api/v1/jobs/{id}/has-applied"] = routeSpec{
		Tags: []string{"job"}, Summary: "Check whether the caller has applied",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/jobs/{id}/applications"] = routeSpec{
		Tags: []string{"job"}, Summary: "List applications for a job (owner)",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/jobs/applications/mine"] = routeSpec{
		Tags: []string{"job"}, Summary: "List my submitted applications",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["DELETE /api/v1/jobs/applications/{applicationId}"] = routeSpec{
		Tags: []string{"job"}, Summary: "Withdraw a submitted application",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/jobs/{id}/applications/{applicantId}/contact"] = routeSpec{
		Tags: []string{"job"}, Summary: "Open a conversation with an applicant",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONRef, SuccessRef: "ConversationResponse", SuccessStatus: "200",
	}
}

func catalogueReview(c map[string]routeSpec) {
	c["POST /api/v1/reviews/"] = routeSpec{
		Tags: []string{"review"}, Summary: "Publish a review",
		AuthRequired: true, RequestBody: jsonRequestBody("CreateReviewRequest"),
		SuccessKind: successJSONRef, SuccessRef: "ReviewResponse", SuccessStatus: "201",
	}
	c["GET /api/v1/reviews/can-review/{proposalId}"] = routeSpec{
		Tags: []string{"review"}, Summary: "Check whether a proposal is reviewable",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/reviews/org/{orgId}"] = routeSpec{
		Tags: []string{"review"}, Summary: "List public reviews for an organization",
		SuccessKind: successJSONList, SuccessRef: "ReviewResponse", SuccessStatus: "200",
	}
	c["GET /api/v1/reviews/average/{orgId}"] = routeSpec{
		Tags: []string{"review"}, Summary: "Public review average for an organization",
		SuccessKind: successRawJSON, SuccessStatus: "200",
	}
}

func catalogueReport(c map[string]routeSpec) {
	c["POST /api/v1/reports/"] = routeSpec{
		Tags: []string{"report"}, Summary: "Submit a moderation report",
		AuthRequired: true, RequestBody: jsonRequestBody("CreateReportRequest"),
		SuccessKind: successJSONRef, SuccessRef: "ReportResponse", SuccessStatus: "201",
	}
	c["GET /api/v1/reports/mine"] = routeSpec{
		Tags: []string{"report"}, Summary: "List my submitted reports",
		AuthRequired: true, SuccessKind: successJSONList, SuccessRef: "ReportResponse", SuccessStatus: "200",
	}
}

func catalogueSocialLink(c map[string]routeSpec) {
	c["GET /api/v1/profile/social-links/"] = routeSpec{
		Tags: []string{"social-link"}, Summary: "List my social links",
		AuthRequired: true, SuccessKind: successJSONList, SuccessRef: "SocialLinkResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/profile/social-links/"] = routeSpec{
		Tags: []string{"social-link"}, Summary: "Replace my social links",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successJSONList, SuccessRef: "SocialLinkResponse", SuccessStatus: "200",
	}
	c["DELETE /api/v1/profile/social-links/{platform}"] = routeSpec{
		Tags: []string{"social-link"}, Summary: "Delete a social link",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["GET /api/v1/profiles/{orgId}/social-links"] = routeSpec{
		Tags: []string{"social-link"}, Summary: "Public social links for an organization",
		SuccessKind: successJSONList, SuccessRef: "SocialLinkResponse", SuccessStatus: "200",
	}
}

func cataloguePortfolio(c map[string]routeSpec) {
	c["POST /api/v1/portfolio/"] = routeSpec{
		Tags: []string{"portfolio"}, Summary: "Create a portfolio item",
		AuthRequired: true, RequestBody: jsonRequestBody("CreatePortfolioItemRequest"),
		SuccessKind: successJSONRef, SuccessRef: "PortfolioItemResponse", SuccessStatus: "201",
	}
	c["GET /api/v1/portfolio/{id}"] = routeSpec{
		Tags: []string{"portfolio"}, Summary: "Read a portfolio item",
		SuccessKind: successJSONRef, SuccessRef: "PortfolioItemResponse", SuccessStatus: "200",
	}
	c["PUT /api/v1/portfolio/{id}"] = routeSpec{
		Tags: []string{"portfolio"}, Summary: "Update a portfolio item",
		AuthRequired: true, RequestBody: jsonRequestBody("UpdatePortfolioItemRequest"),
		SuccessKind: successJSONRef, SuccessRef: "PortfolioItemResponse", SuccessStatus: "200",
	}
	c["DELETE /api/v1/portfolio/{id}"] = routeSpec{
		Tags: []string{"portfolio"}, Summary: "Delete a portfolio item",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["PUT /api/v1/portfolio/reorder"] = routeSpec{
		Tags: []string{"portfolio"}, Summary: "Reorder portfolio items",
		AuthRequired: true, RequestBody: jsonRequestBody("ReorderPortfolioRequest"),
		SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["GET /api/v1/portfolio/org/{orgId}"] = routeSpec{
		Tags: []string{"portfolio"}, Summary: "Public portfolio for an organization",
		SuccessKind: successJSONList, SuccessRef: "PortfolioItemResponse", SuccessStatus: "200",
	}
}

func catalogueNotification(c map[string]routeSpec) {
	c["GET /api/v1/notifications/"] = routeSpec{
		Tags: []string{"notification"}, Summary: "List notifications",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["GET /api/v1/notifications/unread-count"] = routeSpec{
		Tags: []string{"notification"}, Summary: "Unread notification count",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["POST /api/v1/notifications/{id}/read"] = routeSpec{
		Tags: []string{"notification"}, Summary: "Mark a notification as read",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/notifications/read-all"] = routeSpec{
		Tags: []string{"notification"}, Summary: "Mark all notifications as read",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["DELETE /api/v1/notifications/{id}"] = routeSpec{
		Tags: []string{"notification"}, Summary: "Delete a notification",
		AuthRequired: true, SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["GET /api/v1/notifications/preferences"] = routeSpec{
		Tags: []string{"notification"}, Summary: "Read notification preferences",
		AuthRequired: true, SuccessKind: successRawJSON, SuccessStatus: "200",
	}
	c["PUT /api/v1/notifications/preferences"] = routeSpec{
		Tags: []string{"notification"}, Summary: "Update notification preferences",
		AuthRequired: true, RequestBody: jsonRequestBody("UpdateNotificationPreferencesRequest"),
		SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["PATCH /api/v1/notifications/preferences/bulk-email"] = routeSpec{
		Tags: []string{"notification"}, Summary: "Bulk-toggle email category preferences",
		AuthRequired: true, RequestBody: rawJSONRequestBody(),
		SuccessKind: successNoContent, SuccessStatus: "204",
	}
	c["POST /api/v1/notifications/device-token"] = routeSpec{
		Tags: []string{"notification"}, Summary: "Register a push device token",
		AuthRequired: true, RequestBody: jsonRequestBody("RegisterDeviceTokenRequest"),
		SuccessKind: successNoContent, SuccessStatus: "204",
	}
}
