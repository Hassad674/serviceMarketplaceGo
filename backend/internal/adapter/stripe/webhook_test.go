package stripe

import (
	"encoding/json"
	"testing"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests validate that buildAccountSnapshot extracts the full
// requirements picture from a Stripe Account struct. They use realistic
// payloads mimicking what Stripe sends via the account.updated webhook.
//
// The goal: prove that when Stripe says "this account needs X document",
// our snapshot carries that fact downstream to the Notifier so the user
// gets a contextual notification.

func TestBuildAccountSnapshot_NilRequirements_ReturnsBaseFields(t *testing.T) {
	acct := &stripe.Account{
		ID:               "acct_1",
		Country:          "FR",
		ChargesEnabled:   true,
		PayoutsEnabled:   true,
		DetailsSubmitted: true,
	}
	snap := buildAccountSnapshot(acct)

	require.NotNil(t, snap)
	assert.Equal(t, "acct_1", snap.AccountID)
	assert.Equal(t, "FR", snap.Country)
	assert.True(t, snap.ChargesEnabled)
	assert.True(t, snap.PayoutsEnabled)
	assert.True(t, snap.DetailsSubmitted)
	assert.Empty(t, snap.CurrentlyDue)
	assert.Empty(t, snap.RequirementErrors)
}

func TestBuildAccountSnapshot_WithBusinessType_PopulatesField(t *testing.T) {
	acct := &stripe.Account{
		ID:           "acct_1",
		Country:      "FR",
		BusinessType: stripe.AccountBusinessTypeCompany,
	}
	snap := buildAccountSnapshot(acct)
	assert.Equal(t, "company", snap.BusinessType)
}

func TestBuildAccountSnapshot_AllRequirementPartitions(t *testing.T) {
	acct := &stripe.Account{
		ID:             "acct_2",
		Country:        "FR",
		ChargesEnabled: true,
		Requirements: &stripe.AccountRequirements{
			CurrentlyDue:        []string{"individual.verification.document", "individual.phone"},
			EventuallyDue:       []string{"individual.address.city"},
			PastDue:             []string{"external_account"},
			PendingVerification: []string{"individual.dob.day"},
			DisabledReason:      stripe.AccountRequirementsDisabledReasonRequirementsPastDue,
		},
	}
	snap := buildAccountSnapshot(acct)

	assert.Equal(t, []string{"individual.verification.document", "individual.phone"}, snap.CurrentlyDue)
	assert.Equal(t, []string{"individual.address.city"}, snap.EventuallyDue)
	assert.Equal(t, []string{"external_account"}, snap.PastDue)
	assert.Equal(t, []string{"individual.dob.day"}, snap.PendingVerification)
	assert.Equal(t, "requirements.past_due", snap.DisabledReason)
}

func TestBuildAccountSnapshot_DocumentRejectionError(t *testing.T) {
	acct := &stripe.Account{
		ID:      "acct_3",
		Country: "FR",
		Requirements: &stripe.AccountRequirements{
			CurrentlyDue: []string{"individual.verification.document"},
			Errors: []*stripe.AccountRequirementsError{
				{
					Requirement: "individual.verification.document",
					Code:        "verification_document_expired",
					Reason:      "The document has expired.",
				},
			},
		},
	}
	snap := buildAccountSnapshot(acct)

	require.Len(t, snap.RequirementErrors, 1)
	assert.Equal(t, "individual.verification.document", snap.RequirementErrors[0].Requirement)
	assert.Equal(t, "verification_document_expired", snap.RequirementErrors[0].Code)
	assert.Equal(t, "The document has expired.", snap.RequirementErrors[0].Reason)
}

func TestBuildAccountSnapshot_MultipleErrors_AllCaptured(t *testing.T) {
	acct := &stripe.Account{
		ID: "acct_4",
		Requirements: &stripe.AccountRequirements{
			Errors: []*stripe.AccountRequirementsError{
				{Requirement: "individual.verification.document", Code: "verification_document_expired", Reason: "expired"},
				{Requirement: "individual.verification.additional_document", Code: "verification_document_too_blurry", Reason: "blurry"},
				{Requirement: "external_account", Code: "invalid_value_other", Reason: "invalid IBAN"},
			},
		},
	}
	snap := buildAccountSnapshot(acct)
	assert.Len(t, snap.RequirementErrors, 3)
}

func TestBuildAccountSnapshot_FromRealisticJSON_AccountUpdated(t *testing.T) {
	// Payload shape mirrors what Stripe sends in an account.updated webhook
	// for a French company account that just had a document rejected.
	payload := []byte(`{
		"id": "acct_1TIsgNPyy7y81FsB",
		"object": "account",
		"country": "FR",
		"business_type": "company",
		"charges_enabled": true,
		"payouts_enabled": true,
		"details_submitted": true,
		"requirements": {
			"currently_due": [
				"person_1NzR.verification.document"
			],
			"eventually_due": [
				"person_1NzR.address.line1"
			],
			"past_due": [],
			"pending_verification": [],
			"disabled_reason": null,
			"errors": [
				{
					"requirement": "person_1NzR.verification.document",
					"code": "verification_document_expired",
					"reason": "The document has expired. Please upload a current one."
				}
			]
		}
	}`)

	var acct stripe.Account
	require.NoError(t, json.Unmarshal(payload, &acct))

	snap := buildAccountSnapshot(&acct)

	assert.Equal(t, "acct_1TIsgNPyy7y81FsB", snap.AccountID)
	assert.Equal(t, "FR", snap.Country)
	assert.Equal(t, "company", snap.BusinessType)
	assert.True(t, snap.ChargesEnabled)
	assert.True(t, snap.PayoutsEnabled)
	assert.True(t, snap.DetailsSubmitted)
	assert.Len(t, snap.CurrentlyDue, 1)
	assert.Contains(t, snap.CurrentlyDue[0], "verification.document")
	assert.Len(t, snap.EventuallyDue, 1)
	require.Len(t, snap.RequirementErrors, 1)
	assert.Equal(t, "verification_document_expired", snap.RequirementErrors[0].Code)
}

func TestBuildAccountSnapshot_FromRealisticJSON_AccountSuspended(t *testing.T) {
	payload := []byte(`{
		"id": "acct_99",
		"object": "account",
		"country": "FR",
		"business_type": "individual",
		"charges_enabled": false,
		"payouts_enabled": false,
		"details_submitted": true,
		"requirements": {
			"currently_due": [],
			"eventually_due": [],
			"past_due": ["individual.verification.document"],
			"pending_verification": [],
			"disabled_reason": "requirements.past_due",
			"errors": []
		}
	}`)

	var acct stripe.Account
	require.NoError(t, json.Unmarshal(payload, &acct))
	snap := buildAccountSnapshot(&acct)

	assert.False(t, snap.ChargesEnabled)
	assert.False(t, snap.PayoutsEnabled)
	assert.Equal(t, "requirements.past_due", snap.DisabledReason)
	assert.Contains(t, snap.PastDue, "individual.verification.document")
}

func TestBuildAccountSnapshot_FromRealisticJSON_FreshActivation(t *testing.T) {
	payload := []byte(`{
		"id": "acct_new",
		"object": "account",
		"country": "DE",
		"business_type": "individual",
		"charges_enabled": true,
		"payouts_enabled": true,
		"details_submitted": true,
		"requirements": {
			"currently_due": [],
			"eventually_due": [],
			"past_due": [],
			"pending_verification": [],
			"disabled_reason": null,
			"errors": []
		}
	}`)

	var acct stripe.Account
	require.NoError(t, json.Unmarshal(payload, &acct))
	snap := buildAccountSnapshot(&acct)

	assert.True(t, snap.ChargesEnabled)
	assert.True(t, snap.PayoutsEnabled)
	assert.Empty(t, snap.CurrentlyDue)
	assert.Empty(t, snap.RequirementErrors)
	assert.Equal(t, "", snap.DisabledReason)
}

func TestBuildAccountSnapshot_FromRealisticJSON_EventuallyDueFirstTime(t *testing.T) {
	// Scenario: Stripe has just informed us that the account will need
	// a document in the future. currently_due is still empty (account
	// remains fully active) but eventually_due shows what's coming.
	payload := []byte(`{
		"id": "acct_warn",
		"object": "account",
		"country": "US",
		"business_type": "individual",
		"charges_enabled": true,
		"payouts_enabled": true,
		"details_submitted": true,
		"requirements": {
			"currently_due": [],
			"eventually_due": ["individual.verification.additional_document"],
			"past_due": [],
			"pending_verification": [],
			"disabled_reason": null,
			"errors": []
		}
	}`)

	var acct stripe.Account
	require.NoError(t, json.Unmarshal(payload, &acct))
	snap := buildAccountSnapshot(&acct)

	assert.True(t, snap.ChargesEnabled) // account still works
	assert.Len(t, snap.EventuallyDue, 1)
	assert.Contains(t, snap.EventuallyDue[0], "additional_document")
}

func TestBuildAccountSnapshot_PendingVerification_InProgress(t *testing.T) {
	// Scenario: user just uploaded a document, Stripe is reviewing it.
	// Field moves from currently_due → pending_verification.
	payload := []byte(`{
		"id": "acct_pending",
		"object": "account",
		"country": "FR",
		"business_type": "individual",
		"charges_enabled": true,
		"payouts_enabled": true,
		"details_submitted": true,
		"requirements": {
			"currently_due": [],
			"eventually_due": [],
			"past_due": [],
			"pending_verification": ["individual.verification.document"],
			"disabled_reason": null,
			"errors": []
		}
	}`)

	var acct stripe.Account
	require.NoError(t, json.Unmarshal(payload, &acct))
	snap := buildAccountSnapshot(&acct)

	assert.Empty(t, snap.CurrentlyDue)
	assert.Len(t, snap.PendingVerification, 1)
}

func TestBuildAccountSnapshot_CapabilityDisabled_Scenario(t *testing.T) {
	// Scenario: Stripe disabled card_payments capability — account must
	// resolve requirements to get it back. charges_enabled becomes false.
	payload := []byte(`{
		"id": "acct_cap",
		"object": "account",
		"country": "FR",
		"business_type": "company",
		"charges_enabled": false,
		"payouts_enabled": true,
		"details_submitted": true,
		"requirements": {
			"currently_due": ["company.verification.document"],
			"eventually_due": [],
			"past_due": [],
			"pending_verification": [],
			"disabled_reason": "requirements.past_due",
			"errors": [
				{
					"requirement": "company.verification.document",
					"code": "verification_document_not_readable",
					"reason": "We could not read the document."
				}
			]
		}
	}`)

	var acct stripe.Account
	require.NoError(t, json.Unmarshal(payload, &acct))
	snap := buildAccountSnapshot(&acct)

	assert.False(t, snap.ChargesEnabled)
	assert.True(t, snap.PayoutsEnabled) // payouts still work
	assert.Equal(t, "requirements.past_due", snap.DisabledReason)
	require.Len(t, snap.RequirementErrors, 1)
	assert.Equal(t, "verification_document_not_readable", snap.RequirementErrors[0].Code)
}

func TestBuildAccountSnapshot_NonFRCountry_ExtractsCorrectly(t *testing.T) {
	payload := []byte(`{
		"id": "acct_us",
		"object": "account",
		"country": "US",
		"business_type": "individual",
		"charges_enabled": true,
		"payouts_enabled": true,
		"details_submitted": true,
		"requirements": {"currently_due": [], "eventually_due": [], "past_due": [], "pending_verification": [], "errors": []}
	}`)
	var acct stripe.Account
	require.NoError(t, json.Unmarshal(payload, &acct))
	snap := buildAccountSnapshot(&acct)
	assert.Equal(t, "US", snap.Country)
}

func TestBuildAccountSnapshot_EmptyAccount_DoesNotPanic(t *testing.T) {
	snap := buildAccountSnapshot(&stripe.Account{})
	assert.NotNil(t, snap)
	assert.Equal(t, "", snap.AccountID)
	assert.Empty(t, snap.CurrentlyDue)
}
