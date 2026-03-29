package payment

import (
	"time"

	"github.com/google/uuid"
)

type PersonRole string

const (
	RoleRepresentative PersonRole = "representative"
	RoleDirector       PersonRole = "director"
	RoleOwner          PersonRole = "owner"
	RoleExecutive      PersonRole = "executive"
)

type BusinessPerson struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Role           PersonRole
	FirstName      string
	LastName       string
	DateOfBirth    time.Time
	Email          string
	Phone          string
	Address        string
	City           string
	PostalCode     string
	Title          string
	StripePersonID string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type NewBusinessPersonInput struct {
	UserID      uuid.UUID
	Role        string
	FirstName   string
	LastName    string
	DateOfBirth time.Time
	Email       string
	Phone       string
	Address     string
	City        string
	PostalCode  string
	Title       string
}

func NewBusinessPerson(input NewBusinessPersonInput) (*BusinessPerson, error) {
	role := PersonRole(input.Role)
	if !isValidPersonRole(role) {
		return nil, ErrInvalidPersonRole
	}
	if input.FirstName == "" || input.LastName == "" {
		return nil, ErrPersonNameRequired
	}

	now := time.Now()
	return &BusinessPerson{
		ID:          uuid.New(),
		UserID:      input.UserID,
		Role:        role,
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		DateOfBirth: input.DateOfBirth,
		Email:       input.Email,
		Phone:       input.Phone,
		Address:     input.Address,
		City:        input.City,
		PostalCode:  input.PostalCode,
		Title:       input.Title,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func isValidPersonRole(r PersonRole) bool {
	switch r {
	case RoleRepresentative, RoleDirector, RoleOwner, RoleExecutive:
		return true
	}
	return false
}
