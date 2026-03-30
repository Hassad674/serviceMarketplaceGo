package stripe

import (
	"context"
	"fmt"
	"io"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	stripefile "github.com/stripe/stripe-go/v82/file"
	stripeperson "github.com/stripe/stripe-go/v82/person"
	"github.com/stripe/stripe-go/v82/token"
)

func (s *Service) UploadIdentityFile(_ context.Context, filename string, reader io.Reader, purpose string) (string, error) {
	params := &stripe.FileParams{
		Purpose:    stripe.String(purpose),
		FileReader: reader,
		Filename:   stripe.String(filename),
	}

	f, err := stripefile.New(params)
	if err != nil {
		return "", fmt.Errorf("upload stripe file: %w", err)
	}

	return f.ID, nil
}

func (s *Service) UpdateAccountVerification(_ context.Context, accountID string, frontFileID, backFileID string) error {
	// Check if individual or company account
	acct, err := account.GetByID(accountID, nil)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}

	if acct.BusinessType == stripe.AccountBusinessTypeCompany {
		return s.updateCompanyRepVerification(accountID, frontFileID, backFileID)
	}

	// Individual account — use account token
	tokenParams := &stripe.TokenParams{
		Account: &stripe.TokenAccountParams{
			Individual: &stripe.PersonParams{
				Verification: &stripe.PersonVerificationParams{
					Document: &stripe.PersonVerificationDocumentParams{
						Front: stripe.String(frontFileID),
					},
				},
			},
		},
	}
	if backFileID != "" {
		tokenParams.Account.Individual.Verification.Document.Back = stripe.String(backFileID)
	}

	tok, err := token.New(tokenParams)
	if err != nil {
		return fmt.Errorf("create verification token: %w", err)
	}

	_, err = account.Update(accountID, &stripe.AccountParams{
		AccountToken: stripe.String(tok.ID),
	})
	if err != nil {
		return fmt.Errorf("update account verification: %w", err)
	}
	return nil
}

// updateCompanyRepVerification attaches documents to the representative person.
func (s *Service) updateCompanyRepVerification(accountID, frontFileID, backFileID string) error {
	params := &stripe.PersonListParams{
		Account: stripe.String(accountID),
	}
	iter := stripeperson.List(params)
	for iter.Next() {
		p := iter.Person()
		if p.Relationship != nil && p.Relationship.Representative {
			updateParams := &stripe.PersonParams{
				Account: stripe.String(accountID),
				Verification: &stripe.PersonVerificationParams{
					Document: &stripe.PersonVerificationDocumentParams{
						Front: stripe.String(frontFileID),
					},
				},
			}
			if backFileID != "" {
				updateParams.Verification.Document.Back = stripe.String(backFileID)
			}
			_, err := stripeperson.Update(p.ID, updateParams)
			if err != nil {
				return fmt.Errorf("update representative verification: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("no representative person found")
}
