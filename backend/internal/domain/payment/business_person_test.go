package payment

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBusinessPerson(t *testing.T) {
	userID := uuid.New()
	dob := time.Date(1990, 1, 15, 0, 0, 0, 0, time.UTC)

	validRoles := []string{"representative", "director", "owner", "executive"}

	for _, role := range validRoles {
		t.Run("valid role "+role, func(t *testing.T) {
			person, err := NewBusinessPerson(NewBusinessPersonInput{
				UserID:      userID,
				Role:        role,
				FirstName:   "Jean",
				LastName:    "Dupont",
				DateOfBirth: dob,
				Email:       "jean@example.com",
				Phone:       "+33612345678",
				Address:     "1 Rue Test",
				City:        "Paris",
				PostalCode:  "75001",
				Title:       "CEO",
			})

			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, person.ID)
			assert.Equal(t, userID, person.UserID)
			assert.Equal(t, PersonRole(role), person.Role)
			assert.Equal(t, "Jean", person.FirstName)
			assert.Equal(t, "Dupont", person.LastName)
			assert.Equal(t, dob, person.DateOfBirth)
			assert.Equal(t, "jean@example.com", person.Email)
			assert.Equal(t, "+33612345678", person.Phone)
			assert.Equal(t, "CEO", person.Title)
		})
	}

	t.Run("invalid role", func(t *testing.T) {
		person, err := NewBusinessPerson(NewBusinessPersonInput{
			UserID:    userID,
			Role:      "ceo",
			FirstName: "Jean",
			LastName:  "Dupont",
		})

		assert.ErrorIs(t, err, ErrInvalidPersonRole)
		assert.Nil(t, person)
	})

	t.Run("empty first name", func(t *testing.T) {
		person, err := NewBusinessPerson(NewBusinessPersonInput{
			UserID:    userID,
			Role:      "representative",
			FirstName: "",
			LastName:  "Dupont",
		})

		assert.ErrorIs(t, err, ErrPersonNameRequired)
		assert.Nil(t, person)
	})

	t.Run("empty last name", func(t *testing.T) {
		person, err := NewBusinessPerson(NewBusinessPersonInput{
			UserID:    userID,
			Role:      "representative",
			FirstName: "Jean",
			LastName:  "",
		})

		assert.ErrorIs(t, err, ErrPersonNameRequired)
		assert.Nil(t, person)
	})

	t.Run("both names empty", func(t *testing.T) {
		person, err := NewBusinessPerson(NewBusinessPersonInput{
			UserID:    userID,
			Role:      "representative",
			FirstName: "",
			LastName:  "",
		})

		assert.ErrorIs(t, err, ErrPersonNameRequired)
		assert.Nil(t, person)
	})

	t.Run("invalid role checked before name", func(t *testing.T) {
		_, err := NewBusinessPerson(NewBusinessPersonInput{
			UserID:    userID,
			Role:      "invalid",
			FirstName: "",
			LastName:  "",
		})

		assert.ErrorIs(t, err, ErrInvalidPersonRole)
	})
}
