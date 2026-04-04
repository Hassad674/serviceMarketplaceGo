package payment

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/payment"
)

// --- buildRequirementSections tests ---

func TestBuildRequirementSections_CurrentlyDue(t *testing.T) {
	reqs := &domain.AccountRequirements{
		CurrentlyDue: []string{"individual.first_name"},
	}

	sections := buildRequirementSections(reqs, "FR")

	require.NotEmpty(t, sections)
	found := findFieldByPath(sections, "individual.first_name")
	require.NotNil(t, found, "field individual.first_name should be present")
	assert.Equal(t, "currently_due", found.Urgency)
}

func TestBuildRequirementSections_EventuallyDue(t *testing.T) {
	reqs := &domain.AccountRequirements{
		EventuallyDue: []string{"individual.last_name"},
	}

	sections := buildRequirementSections(reqs, "FR")

	require.NotEmpty(t, sections)
	found := findFieldByPath(sections, "individual.last_name")
	require.NotNil(t, found, "field individual.last_name should be present")
	assert.Equal(t, "eventually_due", found.Urgency)
}

func TestBuildRequirementSections_PastDue(t *testing.T) {
	reqs := &domain.AccountRequirements{
		PastDue: []string{"individual.address.line1"},
	}

	sections := buildRequirementSections(reqs, "FR")

	require.NotEmpty(t, sections)
	found := findFieldByPath(sections, "individual.address.line1")
	require.NotNil(t, found, "field individual.address.line1 should be present")
	assert.Equal(t, "past_due", found.Urgency)
}

func TestBuildRequirementSections_Deduplication(t *testing.T) {
	reqs := &domain.AccountRequirements{
		CurrentlyDue:  []string{"individual.phone"},
		EventuallyDue: []string{"individual.phone"},
	}

	sections := buildRequirementSections(reqs, "FR")

	// Field should appear exactly once
	count := countFieldByPath(sections, "individual.phone")
	assert.Equal(t, 1, count, "deduplicated field should appear exactly once")

	// Highest urgency wins: currently_due > eventually_due
	found := findFieldByPath(sections, "individual.phone")
	require.NotNil(t, found)
	assert.Equal(t, "currently_due", found.Urgency, "highest urgency should win")
}

func TestBuildRequirementSections_Empty(t *testing.T) {
	reqs := &domain.AccountRequirements{}

	sections := buildRequirementSections(reqs, "FR")

	assert.Empty(t, sections)
}

func TestBuildRequirementSections_MergesAllLists(t *testing.T) {
	reqs := &domain.AccountRequirements{
		CurrentlyDue:  []string{"individual.first_name"},
		EventuallyDue: []string{"individual.last_name"},
		PastDue:       []string{"individual.address.city"},
	}

	sections := buildRequirementSections(reqs, "FR")

	firstName := findFieldByPath(sections, "individual.first_name")
	lastName := findFieldByPath(sections, "individual.last_name")
	city := findFieldByPath(sections, "individual.address.city")

	require.NotNil(t, firstName, "first_name from currently_due should be present")
	require.NotNil(t, lastName, "last_name from eventually_due should be present")
	require.NotNil(t, city, "city from past_due should be present")

	assert.Equal(t, "currently_due", firstName.Urgency)
	assert.Equal(t, "eventually_due", lastName.Urgency)
	assert.Equal(t, "past_due", city.Urgency)
}

func TestBuildRequirementSections_PastDueBeatsCurrentlyDue(t *testing.T) {
	reqs := &domain.AccountRequirements{
		CurrentlyDue: []string{"individual.email"},
		PastDue:      []string{"individual.email"},
	}

	sections := buildRequirementSections(reqs, "FR")

	found := findFieldByPath(sections, "individual.email")
	require.NotNil(t, found)
	assert.Equal(t, "past_due", found.Urgency, "past_due should beat currently_due")
}

func TestBuildRequirementSections_ExternalAccountAddsBankSection(t *testing.T) {
	reqs := &domain.AccountRequirements{
		CurrentlyDue: []string{"external_account"},
	}

	sections := buildRequirementSections(reqs, "FR")

	var bankSection *FieldSection
	for i := range sections {
		if sections[i].ID == "bank" {
			bankSection = &sections[i]
			break
		}
	}
	require.NotNil(t, bankSection, "bank section should be present for external_account requirement")
	assert.Equal(t, "bankAccount", bankSection.TitleKey)
}

func TestBuildRequirementSections_AutoHandledFieldsSkipped(t *testing.T) {
	reqs := &domain.AccountRequirements{
		CurrentlyDue: []string{
			"tos_acceptance.date",
			"business_type",
			"individual.first_name",
		},
	}

	sections := buildRequirementSections(reqs, "FR")

	// Auto-handled fields should not appear
	assert.Nil(t, findFieldByPath(sections, "tos_acceptance.date"))
	assert.Nil(t, findFieldByPath(sections, "business_type"))
	// Regular field should be present
	assert.NotNil(t, findFieldByPath(sections, "individual.first_name"))
}

// --- NotifyNewRequirements tests ---

func TestNotifyNewRequirements_SendsWhenRequirementsExist(t *testing.T) {
	notifier := &mockNotificationSender{}
	svc := NewService(ServiceDeps{
		Payments:      &mockPaymentInfoRepo{},
		Records:       &mockPaymentRecordRepo{},
		Documents:     &mockIdentityDocRepo{},
		Persons:       &mockBusinessPersonRepo{},
		Storage:       &mockStorageService{},
		Notifications: notifier,
	})

	userID := uuid.New()
	reqs := &domain.AccountRequirements{
		CurrentlyDue: []string{"individual.first_name"},
	}

	svc.NotifyNewRequirements(context.Background(), userID, reqs)

	require.Len(t, notifier.calls, 1)
	assert.Equal(t, userID, notifier.calls[0].UserID)
	assert.Equal(t, "stripe_requirements", notifier.calls[0].Type)
	assert.Contains(t, notifier.calls[0].Title, "Stripe")
}

func TestNotifyNewRequirements_SkipsWhenEmpty(t *testing.T) {
	notifier := &mockNotificationSender{}
	svc := NewService(ServiceDeps{
		Payments:      &mockPaymentInfoRepo{},
		Records:       &mockPaymentRecordRepo{},
		Documents:     &mockIdentityDocRepo{},
		Persons:       &mockBusinessPersonRepo{},
		Storage:       &mockStorageService{},
		Notifications: notifier,
	})

	svc.NotifyNewRequirements(context.Background(), uuid.New(), &domain.AccountRequirements{})

	assert.Empty(t, notifier.calls, "no notification when requirements are empty")
}

func TestNotifyNewRequirements_SkipsWhenNotificationsNil(t *testing.T) {
	svc := NewService(ServiceDeps{
		Payments:  &mockPaymentInfoRepo{},
		Records:   &mockPaymentRecordRepo{},
		Documents: &mockIdentityDocRepo{},
		Persons:   &mockBusinessPersonRepo{},
		Storage:   &mockStorageService{},
		// Notifications is nil
	})

	// Should not panic
	svc.NotifyNewRequirements(context.Background(), uuid.New(), &domain.AccountRequirements{
		CurrentlyDue: []string{"individual.first_name"},
	})
}

func TestNotifyNewRequirements_SendsForEventuallyDueOnly(t *testing.T) {
	notifier := &mockNotificationSender{}
	svc := NewService(ServiceDeps{
		Payments:      &mockPaymentInfoRepo{},
		Records:       &mockPaymentRecordRepo{},
		Documents:     &mockIdentityDocRepo{},
		Persons:       &mockBusinessPersonRepo{},
		Storage:       &mockStorageService{},
		Notifications: notifier,
	})

	reqs := &domain.AccountRequirements{
		EventuallyDue: []string{"individual.last_name"},
	}
	svc.NotifyNewRequirements(context.Background(), uuid.New(), reqs)

	require.Len(t, notifier.calls, 1)
}

func TestNotifyNewRequirements_SendsForPastDueOnly(t *testing.T) {
	notifier := &mockNotificationSender{}
	svc := NewService(ServiceDeps{
		Payments:      &mockPaymentInfoRepo{},
		Records:       &mockPaymentRecordRepo{},
		Documents:     &mockIdentityDocRepo{},
		Persons:       &mockBusinessPersonRepo{},
		Storage:       &mockStorageService{},
		Notifications: notifier,
	})

	reqs := &domain.AccountRequirements{
		PastDue: []string{"individual.address.line1"},
	}
	svc.NotifyNewRequirements(context.Background(), uuid.New(), reqs)

	require.Len(t, notifier.calls, 1)
}

// --- notifyAccountStatusChange tests ---

func TestNotifyAccountStatusChange_Activated(t *testing.T) {
	notifier := &mockNotificationSender{}
	svc := NewService(ServiceDeps{
		Payments:      &mockPaymentInfoRepo{},
		Records:       &mockPaymentRecordRepo{},
		Documents:     &mockIdentityDocRepo{},
		Persons:       &mockBusinessPersonRepo{},
		Storage:       &mockStorageService{},
		Notifications: notifier,
	})

	userID := uuid.New()
	svc.notifyAccountStatusChange(context.Background(), userID, true, true)

	require.Len(t, notifier.calls, 1)
	assert.Equal(t, userID, notifier.calls[0].UserID)
	assert.Equal(t, "stripe_account_status", notifier.calls[0].Type)
	assert.Contains(t, notifier.calls[0].Title, "activ\u00e9")
	assert.Contains(t, notifier.calls[0].Body, "actif")
}

func TestNotifyAccountStatusChange_PayoutsSuspended(t *testing.T) {
	notifier := &mockNotificationSender{}
	svc := NewService(ServiceDeps{
		Payments:      &mockPaymentInfoRepo{},
		Records:       &mockPaymentRecordRepo{},
		Documents:     &mockIdentityDocRepo{},
		Persons:       &mockBusinessPersonRepo{},
		Storage:       &mockStorageService{},
		Notifications: notifier,
	})

	svc.notifyAccountStatusChange(context.Background(), uuid.New(), true, false)

	require.Len(t, notifier.calls, 1)
	assert.Contains(t, notifier.calls[0].Title, "suspendus")
	assert.Contains(t, notifier.calls[0].Title, "Virements")
}

func TestNotifyAccountStatusChange_ChargesSuspended(t *testing.T) {
	notifier := &mockNotificationSender{}
	svc := NewService(ServiceDeps{
		Payments:      &mockPaymentInfoRepo{},
		Records:       &mockPaymentRecordRepo{},
		Documents:     &mockIdentityDocRepo{},
		Persons:       &mockBusinessPersonRepo{},
		Storage:       &mockStorageService{},
		Notifications: notifier,
	})

	svc.notifyAccountStatusChange(context.Background(), uuid.New(), false, true)

	require.Len(t, notifier.calls, 1)
	assert.Contains(t, notifier.calls[0].Title, "suspendus")
	assert.Contains(t, notifier.calls[0].Title, "Paiements")
}

func TestNotifyAccountStatusChange_NilNotifications(t *testing.T) {
	svc := NewService(ServiceDeps{
		Payments:  &mockPaymentInfoRepo{},
		Records:   &mockPaymentRecordRepo{},
		Documents: &mockIdentityDocRepo{},
		Persons:   &mockBusinessPersonRepo{},
		Storage:   &mockStorageService{},
		// Notifications is nil
	})

	// Should not panic
	svc.notifyAccountStatusChange(context.Background(), uuid.New(), true, true)
}

// --- helpers ---

func findFieldByPath(sections []FieldSection, path string) *FieldSpec {
	for _, s := range sections {
		for i := range s.Fields {
			if s.Fields[i].Path == path {
				return &s.Fields[i]
			}
		}
	}
	return nil
}

func countFieldByPath(sections []FieldSection, path string) int {
	count := 0
	for _, s := range sections {
		for _, f := range s.Fields {
			if f.Path == path {
				count++
			}
		}
	}
	return count
}
