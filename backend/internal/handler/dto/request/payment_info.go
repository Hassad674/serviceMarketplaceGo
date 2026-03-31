package request

// CreateAccountSessionRequest holds the data needed to create an account session.
type CreateAccountSessionRequest struct {
	Email string `json:"email"`
}
