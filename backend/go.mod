module marketplace-backend

go 1.25

require (
	github.com/go-chi/chi/v5 v5.2.5
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.11.2
	golang.org/x/crypto v0.48.0
)

require github.com/resend/resend-go/v2 v2.28.0

// Run `go mod tidy` after adding application code to resolve indirect dependencies.
// golang-migrate requires github.com/hashicorp/errwrap and github.com/hashicorp/go-multierror
// as indirect deps when using the postgres driver.
