package stripe

import (
	"context"
	"fmt"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	stripeperson "github.com/stripe/stripe-go/v82/person"
	"github.com/stripe/stripe-go/v82/token"

	portservice "marketplace-backend/internal/port/service"
)

func (s *Service) CreatePerson(ctx context.Context, accountID string, input portservice.CreatePersonInput) (string, error) {
	params := &stripe.PersonParams{
		Account:   stripe.String(accountID),
		FirstName: stripe.String(input.FirstName),
		LastName:  stripe.String(input.LastName),
	}

	if input.Email != "" {
		params.Email = stripe.String(input.Email)
	}
	if input.Phone != "" {
		params.Phone = stripe.String(input.Phone)
	}
	if input.Title != "" {
		params.Relationship = &stripe.PersonRelationshipParams{
			Title: stripe.String(input.Title),
		}
	} else {
		params.Relationship = &stripe.PersonRelationshipParams{}
	}

	// Set relationship flags
	if input.IsRepresentative {
		params.Relationship.Representative = stripe.Bool(true)
	}
	if input.IsDirector {
		params.Relationship.Director = stripe.Bool(true)
	}
	if input.IsOwner {
		params.Relationship.Owner = stripe.Bool(true)
	}
	if input.IsExecutive {
		params.Relationship.Executive = stripe.Bool(true)
	}

	if !input.DOB.IsZero() {
		params.DOB = &stripe.PersonDOBParams{
			Day:   stripe.Int64(int64(input.DOB.Day())),
			Month: stripe.Int64(int64(input.DOB.Month())),
			Year:  stripe.Int64(int64(input.DOB.Year())),
		}
	}

	if input.Address != "" {
		params.Address = &stripe.AddressParams{
			Line1:      stripe.String(input.Address),
			City:       stripe.String(input.City),
			PostalCode: stripe.String(input.PostalCode),
		}
	}

	p, err := stripeperson.New(params)
	if err != nil {
		return "", fmt.Errorf("create stripe person: %w", err)
	}

	return p.ID, nil
}

func (s *Service) UpdateCompanyFlags(ctx context.Context, accountID string, directorsProvided, executivesProvided, ownersProvided bool) error {
	// Must use account token for FR platforms
	tokenParams := &stripe.TokenParams{
		Account: &stripe.TokenAccountParams{
			Company: &stripe.AccountCompanyParams{
				DirectorsProvided:  stripe.Bool(directorsProvided),
				ExecutivesProvided: stripe.Bool(executivesProvided),
				OwnersProvided:     stripe.Bool(ownersProvided),
			},
		},
	}
	tok, err := token.New(tokenParams)
	if err != nil {
		return fmt.Errorf("create company flags token: %w", err)
	}

	_, err = account.Update(accountID, &stripe.AccountParams{
		AccountToken: stripe.String(tok.ID),
	})
	if err != nil {
		return fmt.Errorf("update company flags: %w", err)
	}
	return nil
}
