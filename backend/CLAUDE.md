# Backend — Go 1.25 + Chi v5 + Hexagonal Architecture

## Module

`marketplace-backend` — the single Go module for the entire backend.

## Structure

```
backend/
├── cmd/
│   ├── api/main.go        -> Entry point. ALL dependency injection happens here.
│   ├── migrate/main.go    -> Run SQL migrations
│   └── seed/main.go       -> Seed initial data (roles, admin user)
│
├── internal/              -> Private application code
│   ├── domain/            -> LAYER 1: Pure business logic
│   ├── port/              -> LAYER 2: Interface contracts
│   ├── app/               -> LAYER 3: Use cases / orchestration
│   ├── adapter/           -> LAYER 4: Concrete implementations
│   ├── handler/           -> LAYER 5: HTTP transport
│   └── config/            -> Configuration from env vars
│
├── pkg/                   -> Public reusable packages
├── migrations/            -> SQL migration files (up/down)
├── mock/                  -> Generated mocks from port interfaces
├── test/                  -> Integration / E2E tests
├── Makefile               -> make run, make test, make migrate
├── go.mod
└── .env.example
```

---

## Dependency rule (absolute, never break this)

```
handler -> app -> domain <- port <- adapter
```

- **domain/** imports NOTHING except Go stdlib
- **port/** imports only domain/
- **app/** imports domain/ and port/ (interfaces only)
- **adapter/** imports domain/, port/, and external libraries
- **handler/** imports app/ and dto/

An adapter NEVER imports another adapter. An app service NEVER imports an adapter directly.

---

## SOLID principles in Go — with code examples

### Single Responsibility

One service = one domain. One file = one concern.

```go
// GOOD: AuthService only handles authentication
type AuthService struct {
    users   repository.UserRepository
    hasher  service.HasherService
    tokens  service.TokenService
}
// Methods: Register, Login, RefreshToken. Nothing else.

// BAD: AuthService also manages profiles, sends emails, generates invoices
type AuthService struct {
    users    repository.UserRepository
    hasher   service.HasherService
    tokens   service.TokenService
    profiles repository.ProfileRepository  // NOT its job
    email    service.EmailService           // NOT its job
    invoices service.InvoiceService         // NOT its job
}
```

**Practical test:** If you cannot describe a service's purpose in one sentence without using "and", it has too many responsibilities. Split it.

### Open/Closed

Port interfaces allow extension without modification of existing code.

```go
// port/service/payment.go — the contract
type PaymentService interface {
    CreateCharge(ctx context.Context, amount int64, currency string, customerID string) (*domain.Charge, error)
    Refund(ctx context.Context, chargeID string) error
}

// adapter/stripe/payment.go — implementation A
type PaymentService struct { client *stripe.Client }
func (s *PaymentService) CreateCharge(ctx context.Context, ...) (*domain.Charge, error) { /* Stripe logic */ }

// adapter/paypal/payment.go — implementation B (new file, zero changes to existing code)
type PaymentService struct { client *paypal.Client }
func (s *PaymentService) CreateCharge(ctx context.Context, ...) (*domain.Charge, error) { /* PayPal logic */ }

// cmd/api/main.go — switching providers is ONE line:
// payment := stripe.New(cfg)    // before
// payment := paypal.New(cfg)    // after — nothing else changes
```

### Liskov Substitution

Any implementation of an interface must be a drop-in replacement.

```go
// All of these satisfy repository.UserRepository identically:
postgresRepo := postgres.NewUserRepository(db)    // production
memoryRepo   := memory.NewUserRepository()         // integration tests
mockRepo     := &mock.UserRepository{...}          // unit tests

// The app service does not know or care which one it received:
svc := auth.NewService(postgresRepo, hasher, tokens)  // production
svc := auth.NewService(mockRepo, hasher, tokens)       // test — identical API
```

**If a mock needs special setup that the real implementation does not, the interface contract is wrong.**

### Interface Segregation

Small, focused interfaces. No god interfaces.

```go
// GOOD: HasherService has exactly 2 methods
type HasherService interface {
    Hash(password string) (string, error)
    Compare(hash, password string) error
}

// GOOD: TokenService has exactly 3 methods
type TokenService interface {
    Generate(userID string, role string) (accessToken string, refreshToken string, err error)
    Validate(token string) (*Claims, error)
    Revoke(ctx context.Context, token string) error
}

// BAD: a god interface with 20 methods that no single consumer needs entirely
type SecurityService interface {
    Hash(password string) (string, error)
    Compare(hash, password string) error
    GenerateToken(userID string, role string) (string, string, error)
    ValidateToken(token string) (*Claims, error)
    RevokeToken(ctx context.Context, token string) error
    Encrypt(data []byte) ([]byte, error)
    Decrypt(data []byte) ([]byte, error)
    GenerateOTP() string
    VerifyOTP(code string) bool
    // ... 11 more methods nobody uses together
}
```

### Dependency Inversion

The app layer depends on port interfaces, never on adapter implementations.

```go
// CORRECT: app/auth/service.go — depends on interfaces from port/
import (
    "marketplace-backend/internal/port/repository"
    "marketplace-backend/internal/port/service"
)

type Service struct {
    users  repository.UserRepository  // interface
    hasher service.HasherService       // interface
    tokens service.TokenService        // interface
}

// WRONG: app/auth/service.go — imports concrete adapter
import (
    "marketplace-backend/internal/adapter/postgres"  // NEVER DO THIS
    "marketplace-backend/internal/adapter/redis"      // NEVER DO THIS
)
```

All wiring happens in `cmd/api/main.go`. Tests inject mocks through the same constructor.

---

## Layer rules

### domain/ — Pure business entities and rules

- Zero external imports. Only Go standard library.
- Contains: entities (structs + methods), value objects, domain errors
- Entities validate themselves: `user.New(email, name, hash, role)` returns error if invalid
- Business rules live HERE, not in app/ or handler/

```go
// CORRECT: validation in domain
func New(email, name, hash string, role Role) (*User, error) {
    if email == "" {
        return nil, ErrInvalidEmail
    }
    if !role.IsValid() {
        return nil, ErrInvalidRole
    }
    return &User{
        ID:        uuid.New().String(),
        Email:     email,
        Name:      name,
        Hash:      hash,
        Role:      role,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }, nil
}

// WRONG: validation in handler or app
```

**Domain modules for this marketplace:**

| Module | Key entities | Notes |
|--------|-------------|-------|
| `user` | User, Role (Agency/Enterprise/Provider) | Provider has `ReferrerEnabled` bool |
| `profile` | AgencyProfile, EnterpriseProfile, ProviderProfile | Role-specific profile data |
| `mission` | Mission, MissionStatus | Enterprise publishes, Agency/Provider applies |
| `contract` | Contract, ContractStatus | Agreement between parties |
| `message` | Conversation, Message | Real-time messaging between users |
| `review` | Review, Rating | Post-contract reviews |
| `notification` | Notification, NotificationType | In-app notifications |
| `payment` | Payment, Invoice, PaymentStatus | Contract payments and commissions |

**Files per domain module**: `entity.go`, `entity_test.go`, `errors.go`, and optionally value objects

### port/ — Interface contracts

- Defines WHAT the system needs, not HOW
- Two sub-packages:
  - `repository/` — data persistence interfaces
  - `service/` — external service interfaces (email, storage, payment, websocket)
- Interfaces are small and specific. No god interfaces.

```go
// CORRECT: focused interface
type UserRepository interface {
    Create(ctx context.Context, u *user.User) error
    FindByID(ctx context.Context, id string) (*user.User, error)
    FindByEmail(ctx context.Context, email string) (*user.User, error)
    Update(ctx context.Context, u *user.User) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, cursor string, limit int) ([]*user.User, string, error)
}

// CORRECT: focused external service interface
type StorageService interface {
    Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error)
    Delete(ctx context.Context, key string) error
    GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// WRONG: catch-all interface with 30 methods
```

### app/ — Use cases (application services)

- Orchestrates domain entities and port interfaces
- Each sub-package = one functional domain (auth, user, mission, contract, etc.)
- Receives dependencies via constructor injection
- Returns domain types and domain errors — NEVER HTTP concepts

```go
// CORRECT
type Service struct {
    users    repository.UserRepository  // interface, not concrete type
    email    service.EmailService
    tokens   service.TokenService
}

func NewService(
    users repository.UserRepository,
    email service.EmailService,
    tokens service.TokenService,
) *Service {
    return &Service{users: users, email: email, tokens: tokens}
}

// WRONG: importing postgres package directly
```

**Files**: `service.go` + `service_test.go` per module. Tests use mocks.

### adapter/ — Concrete implementations

- Each sub-package implements one or more port interfaces
- Sub-packages: `postgres/`, `redis/`, `s3/`, `resend/`, `ws/`
- Can import external libraries (lib/pq, minio SDK, etc.)
- Each adapter has: `client.go` (setup/config) + implementation files

```go
// postgres/user.go implements repository.UserRepository
type UserRepository struct { db *sql.DB }

// s3/storage.go implements service.StorageService (via MinIO-compatible S3 API)
type StorageService struct { client *minio.Client }

// resend/email.go implements service.EmailService
type EmailService struct { client *resend.Client }

// redis/cache.go implements service.CacheService
type CacheService struct { client *redis.Client }

// ws/hub.go implements service.WebSocketService
type WebSocketHub struct { /* gorilla/websocket hub */ }
```

**To swap a provider**: create new adapter, change ONE line in cmd/api/main.go. Nothing else changes.

### handler/ — HTTP transport

- Converts HTTP requests to app service calls and back
- Uses Chi v5 router
- Contains: route definitions, handlers, middleware, DTOs
- Sub-structure:
  - `router.go` — all route definitions
  - `auth.go`, `user.go`, `mission.go`, `contract.go` — handler groups
  - `middleware/` — auth (JWT), CORS, rate limit, logging, role-based access
  - `dto/request/` — incoming request structs with json tags
  - `dto/response/` — outgoing response structs + helpers

```go
// CORRECT: handler is thin, delegates to app service
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    var req request.RegisterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.BadRequest(w, "invalid request body")
        return
    }
    if err := req.Validate(); err != nil {
        response.BadRequest(w, err.Error())
        return
    }
    u, token, err := h.svc.Register(r.Context(), req.Email, req.Name, req.Password, req.Role)
    if err != nil {
        response.HandleDomainError(w, err)
        return
    }
    response.JSON(w, http.StatusCreated, response.AuthResponse{
        Token: token,
        User:  response.UserFromDomain(u),
    })
}

// WRONG: business logic in handler
```

**Every handler function follows the same pattern:**
1. Decode request body (JSON) or read URL params
2. Validate the request DTO
3. Call the appropriate app service method
4. Encode the response (success or error)

### pkg/ — Public reusable packages

- Can be imported by external projects
- Contains pure utilities: `jwt/`, `hash/`, `validate/`, `pagination/`
- Each package is self-contained with its own tests
- No imports from internal/

### config/ — Configuration

- Single `config.go` with typed Config struct
- All env vars loaded and validated at startup
- Default values for local development
- No config scattered across files

---

## API response envelope

All API responses follow a strict, consistent envelope format.

### Success (single resource)

```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "agency@example.com",
    "name": "Test Agency",
    "role": "agency",
    "created_at": "2026-03-16T10:30:00Z"
  },
  "meta": {
    "request_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
  }
}
```

### Error

```json
{
  "error": {
    "code": "email_already_exists",
    "message": "A user with this email address already exists"
  },
  "meta": {
    "request_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
  }
}
```

### List with cursor-based pagination

```json
{
  "data": [
    { "id": "...", "name": "..." },
    { "id": "...", "name": "..." }
  ],
  "meta": {
    "request_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    "pagination": {
      "next_cursor": "eyJpZCI6IjU1MGU4NDAwIn0=",
      "has_more": true,
      "count": 20
    }
  }
}
```

**Rules:**
- Every response includes `meta.request_id` for debugging and support correlation.
- Error codes are `snake_case` and machine-readable. Messages are human-readable.
- `data` is always the payload — either an object (single) or array (list).
- Empty lists return `"data": []`, not `null`.
- Never add top-level fields outside `data`, `error`, and `meta`.

---

## Cursor-based pagination

**Never use OFFSET pagination.** OFFSET performance degrades linearly with dataset size. A query with `OFFSET 100000` must scan and discard 100,000 rows.

### Standard

| Parameter | Type | Default | Max | Description |
|-----------|------|---------|-----|-------------|
| `cursor` | string (query param) | empty (first page) | - | Opaque cursor from previous response |
| `limit` | int (query param) | 20 | 100 | Items per page |

### Implementation pattern

```go
// port/repository interface
type UserRepository interface {
    List(ctx context.Context, cursor string, limit int) (users []*user.User, nextCursor string, err error)
}

// adapter/postgres implementation
func (r *UserRepository) List(ctx context.Context, cursor string, limit int) ([]*user.User, string, error) {
    if limit <= 0 || limit > 100 {
        limit = 20
    }

    var rows *sql.Rows
    var err error

    if cursor == "" {
        rows, err = r.db.QueryContext(ctx,
            `SELECT id, email, name, role, created_at FROM users
             ORDER BY created_at DESC, id DESC
             LIMIT $1`, limit+1) // fetch one extra to determine has_more
    } else {
        createdAt, id := decodeCursor(cursor)
        rows, err = r.db.QueryContext(ctx,
            `SELECT id, email, name, role, created_at FROM users
             WHERE (created_at, id) < ($1, $2)
             ORDER BY created_at DESC, id DESC
             LIMIT $3`, createdAt, id, limit+1)
    }
    // ... scan rows, build nextCursor from last item if len(results) > limit
}
```

### Cursor encoding

The cursor is a base64-encoded JSON object containing the sort fields of the last item:
```go
// {"created_at":"2026-03-16T10:30:00Z","id":"550e8400-..."}
// encoded to: eyJjcmVhdGVkX2F0IjoiMjAyNi0wMy0xNlQxMDozMDowMFoiLCJpZCI6IjU1MGU4NDAwLSJ9
```

Cursors are opaque to clients. They must not parse, modify, or construct cursors.

---

## Idempotency for critical operations

Any `POST` request that creates a resource or triggers a side effect must support idempotency.

### How it works

1. Client sends `Idempotency-Key: <uuid>` header with the request
2. Server checks Redis for the key:
   - **Key exists**: Return the cached response (same status code, same body). Do NOT re-execute.
   - **Key does not exist**: Execute the operation, cache the response in Redis with 24h TTL.
3. If no `Idempotency-Key` header is provided, the request is executed normally (non-idempotent).

### Implementation

```go
// middleware/idempotency.go
func Idempotency(cache service.CacheService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := r.Header.Get("Idempotency-Key")
            if key == "" || r.Method != http.MethodPost {
                next.ServeHTTP(w, r)
                return
            }

            cached, err := cache.Get(r.Context(), "idempotency:"+key)
            if err == nil && cached != nil {
                // Return cached response
                w.Header().Set("Idempotent-Replayed", "true")
                w.WriteHeader(cached.StatusCode)
                w.Write(cached.Body)
                return
            }

            // Capture response, execute, then cache
            rec := httptest.NewRecorder()
            next.ServeHTTP(rec, r)

            cache.Set(r.Context(), "idempotency:"+key, rec, 24*time.Hour)

            // Copy recorded response to actual writer
            for k, v := range rec.Header() {
                w.Header()[k] = v
            }
            w.WriteHeader(rec.Code)
            w.Write(rec.Body.Bytes())
        })
    }
}
```

### Which endpoints need idempotency

| Endpoint | Reason |
|----------|--------|
| `POST /api/v1/contracts` | Creating duplicate contracts costs money |
| `POST /api/v1/payments` | Duplicate payments are unrecoverable |
| `POST /api/v1/missions` | Duplicate mission listings confuse users |
| `POST /api/v1/auth/register` | Duplicate registrations create data inconsistency |

---

## N+1 query prevention

N+1 queries are the single most common performance killer. They are strictly forbidden.

### The problem

```go
// BAD: N+1 queries — 1 query for missions + N queries for users
missions, _ := missionRepo.List(ctx, cursor, limit)
for _, m := range missions {
    m.Author, _ = userRepo.FindByID(ctx, m.AuthorID)  // N additional queries!
}
```

### The solution

```go
// GOOD: Single query with JOIN
func (r *MissionRepository) ListWithAuthor(ctx context.Context, cursor string, limit int) ([]*MissionWithAuthor, string, error) {
    rows, err := r.db.QueryContext(ctx,
        `SELECT m.id, m.title, m.status, m.created_at,
                u.id, u.name, u.email, u.role
         FROM missions m
         JOIN users u ON u.id = m.author_id
         WHERE ($1 = '' OR (m.created_at, m.id) < (decode_cursor($1)))
         ORDER BY m.created_at DESC, m.id DESC
         LIMIT $2`, cursor, limit+1)
    // ...
}

// GOOD alternative: Batch query when JOIN is not practical
func (r *UserRepository) FindByIDs(ctx context.Context, ids []string) ([]*user.User, error) {
    rows, err := r.db.QueryContext(ctx,
        `SELECT id, email, name, role FROM users WHERE id = ANY($1)`, pq.Array(ids))
    // ...
}
```

### Rule of thumb

If you see `for range` followed by a repository call inside the loop, it is an N+1 query. Refactor immediately.

---

## Context standards

Every operation must have an explicit timeout. Never use `context.Background()` in request handlers.

| Operation | Timeout | Source |
|-----------|---------|--------|
| Database queries | 5 seconds | `context.WithTimeout(ctx, 5*time.Second)` in adapter |
| External HTTP calls | 10 seconds | `context.WithTimeout(ctx, 10*time.Second)` in adapter |
| Handler functions | From request | `r.Context()` — already has request_id from middleware |
| Background jobs | Explicit per job | `context.WithTimeout(context.Background(), 30*time.Second)` |
| Graceful shutdown | 30 seconds | `context.WithTimeout(context.Background(), 30*time.Second)` |

### Implementation pattern

```go
// adapter/postgres/user.go — every query gets a timeout
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    row := r.db.QueryRowContext(ctx,
        `SELECT id, email, name, hash, role, referrer_enabled, created_at, updated_at
         FROM users WHERE email = $1`, email)

    var u user.User
    err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Hash, &u.Role,
        &u.ReferrerEnabled, &u.CreatedAt, &u.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, user.ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("finding user by email: %w", err)
    }
    return &u, nil
}

// middleware/request_id.go — every request gets a request_id in context
func RequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := uuid.New().String()
        ctx := context.WithValue(r.Context(), requestIDKey, id)
        w.Header().Set("X-Request-ID", id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## Error handling chain

Errors flow from domain through app to handler. Each layer adds context but never swallows.

### The three layers

```
Domain:  return user.ErrEmailAlreadyExists           // sentinel error (no wrapping)
App:     return fmt.Errorf("register: %w", err)       // wrap with operation context
Handler: errors.Is(err, user.ErrEmailAlreadyExists)   // unwrap and map to HTTP status
```

### Domain errors — sentinels

```go
// domain/user/errors.go
var (
    ErrNotFound       = errors.New("user not found")
    ErrInvalidEmail   = errors.New("invalid email")
    ErrInvalidRole    = errors.New("invalid role")
    ErrDuplicateEmail = errors.New("email already exists")
)
```

### App layer — wrap with context

```go
// app/auth/service.go
func (s *Service) Register(ctx context.Context, email, name, password, role string) (*user.User, string, error) {
    existing, _ := s.users.FindByEmail(ctx, email)
    if existing != nil {
        return nil, "", user.ErrDuplicateEmail
    }

    hash, err := s.hasher.Hash(password)
    if err != nil {
        return nil, "", fmt.Errorf("register: hashing password: %w", err)
    }

    u, err := user.New(email, name, hash, user.Role(role))
    if err != nil {
        return nil, "", fmt.Errorf("register: creating user: %w", err)
    }

    if err := s.users.Create(ctx, u); err != nil {
        return nil, "", fmt.Errorf("register: persisting user: %w", err)
    }

    accessToken, _, err := s.tokens.Generate(u.ID, string(u.Role))
    if err != nil {
        return nil, "", fmt.Errorf("register: generating token: %w", err)
    }

    return u, accessToken, nil
}
```

### Handler — map to HTTP

```go
// handler/dto/response/error.go
func HandleDomainError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, user.ErrNotFound):
        Error(w, http.StatusNotFound, "user_not_found", err.Error())
    case errors.Is(err, user.ErrDuplicateEmail):
        Error(w, http.StatusConflict, "email_already_exists", err.Error())
    case errors.Is(err, user.ErrInvalidEmail), errors.Is(err, user.ErrInvalidRole):
        Error(w, http.StatusBadRequest, "validation_error", err.Error())
    default:
        slog.Error("unhandled error", "error", err)
        Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
    }
}
```

**Rules:**
- Domain errors are sentinels. Never wrap them with `fmt.Errorf` inside the domain layer.
- App layer always wraps with operation name: `fmt.Errorf("register: %w", err)`.
- Handler uses `errors.Is()` to unwrap and match. Never type-switch on error strings.
- `500 Internal Server Error` responses never expose internal details. Log the real error, return generic message.

---

## File size rules

| File type | Max lines | Action when exceeded |
|-----------|-----------|----------------------|
| Any file | 600 | Split by responsibility |
| Handler file | 300 | Split by sub-resource (e.g., `mission_create.go`, `mission_list.go`) |
| Repository file | 400 | The domain may need splitting, or group queries by operation type |
| Service file | 300 | The service has too many responsibilities. Split the domain. |
| Function | 50 | Extract helper functions or break into pipeline steps |

**Why:** Large files are impossible to review, hard to test, and cause merge conflicts. If a file crosses the threshold, it is a signal that responsibilities need splitting — not that the limit is wrong.

---

## Structured JSON responses

All API responses follow a consistent format:

```json
// Success
{
    "data": { ... },
    "meta": { "request_id": "uuid" }
}

// Error
{
    "error": {
        "code": "VALIDATION_ERROR",
        "message": "email is required"
    },
    "meta": { "request_id": "uuid" }
}
```

---

## Middleware stack (order matters)

```go
r.Use(middleware.RequestID)
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(middleware.CORS(allowedOrigins))
r.Use(middleware.RateLimit)
// Per-route:
r.With(middleware.Auth(tokenService)).Get("/profile", handler.GetProfile)
r.With(middleware.Auth(tokenService), middleware.RequireRole("admin")).Get("/admin/users", handler.ListUsers)
```

---

## How to add a new feature

Example: "Add a reviews feature"

1. **domain/review/entity.go** — Review struct, Rating value object, validation, business methods
2. **domain/review/entity_test.go** — Test validation rules
3. **domain/review/errors.go** — Domain errors (ErrInvalidRating, ErrSelfReview, etc.)
4. **port/repository/review.go** — ReviewRepository interface
5. **app/review/service.go** — Use cases (CreateReview, ListByUser, GetAverage, etc.)
6. **app/review/service_test.go** — Unit tests with mocked repository
7. **adapter/postgres/review.go** — SQL implementation of ReviewRepository
8. **handler/review.go** — HTTP endpoints
9. **handler/dto/request/review.go** — Request DTOs
10. **handler/dto/response/review.go** — Response DTOs
11. **cmd/api/main.go** — Wire: `reviewRepo -> reviewSvc -> router`
12. **migrations/00X_create_reviews.up.sql** — Database table
13. **migrations/00X_create_reviews.down.sql** — Rollback

Always follow this order. Domain first, HTTP last.

---

## How to swap a provider

Example: "Replace MinIO with Cloudflare R2"

1. Create `adapter/r2/client.go` + `storage.go`
2. Implement `service.StorageService` interface
3. In `cmd/api/main.go`, change: `s3.NewStorageService(...)` -> `r2.NewStorageService(...)`
4. Done. Zero changes elsewhere.

---

## Naming conventions

### Go standard
- **Exported types/functions**: PascalCase (`UserRepository`, `NewService`, `HandleLogin`)
- **Unexported types/functions**: camelCase (`validateEmail`, `buildQuery`)
- **Files**: snake_case (`user_repository.go`, `auth_handler.go`, `service_test.go`)
- **Packages**: lowercase, single word preferred (`user`, `auth`, `postgres`)
- **Interfaces**: noun or noun phrase, no "I" prefix (`UserRepository`, not `IUserRepository`)
- **Constructors**: `New` + type name (`NewService`, `NewUserRepository`)
- **Test functions**: `TestServiceName_MethodName_Scenario`

### Directory naming
- snake_case for multi-word directories: `dto/request/`, `dto/response/`
- Domain modules are singular: `domain/user/`, not `domain/users/`
- Adapter packages match the technology: `postgres/`, `redis/`, `s3/`, `resend/`

---

## SQL conventions

- Pure SQL with `database/sql` + `lib/pq`. No ORM.
- Parameterized queries ONLY: `$1, $2, $3` — never string concatenation
- All queries use `context.Context` for timeout/cancellation
- Tables: UUID primary key, `created_at TIMESTAMP NOT NULL DEFAULT NOW()`, `updated_at TIMESTAMP NOT NULL DEFAULT NOW()`
- Use `TEXT` not `VARCHAR`. Index foreign keys.
- No cross-feature foreign keys (only reference `users` table)

```go
// CORRECT
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    row := r.db.QueryRowContext(ctx,
        `SELECT id, email, name, hash, role, referrer_enabled, created_at, updated_at
         FROM users WHERE email = $1`, email)

    var u user.User
    err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Hash, &u.Role,
        &u.ReferrerEnabled, &u.CreatedAt, &u.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, user.ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("finding user by email: %w", err)
    }
    return &u, nil
}

// WRONG: string concatenation
query := "SELECT * FROM users WHERE email = '" + email + "'"
```

---

## Migrations

Powered by `golang-migrate`. Migration files live in `backend/migrations/`.

### File naming

```
migrations/
├── 001_create_users.up.sql
├── 001_create_users.down.sql
├── 002_create_profiles.up.sql
├── 002_create_profiles.down.sql
├── 003_create_missions.up.sql
├── 003_create_missions.down.sql
└── ...
```

- Numbered sequentially: `001`, `002`, ..., `010`, ...
- Each migration has an `.up.sql` (apply) and `.down.sql` (rollback)
- snake_case descriptive name: `create_X`, `add_Y_to_X`, `drop_Z`
- Feature-scoped: each feature's tables in their own migration files

### Rules

- **Migrations are immutable.** Once applied in prod, NEVER edit — create a new migration instead.
- **Always write the down migration.** Every `up` must be reversible.
- **Use `IF NOT EXISTS` / `IF EXISTS`** for idempotent migrations.
- **Test locally before prod.** `make migrate-up` locally -> verify -> push -> apply to prod.
- **No cross-feature foreign keys.** Only `REFERENCES users(id)` is allowed.

### Workflow: local -> prod

```
1. Create migration files   ->  manually or via skill
2. Test locally              ->  make migrate-up (on Docker PostgreSQL, port 5434)
3. Verify schema             ->  psql or any DB viewer
4. Rollback test             ->  make migrate-down (verify down works)
5. Re-apply                  ->  make migrate-up
6. Commit & push             ->  git commit
7. Apply to prod             ->  DATABASE_URL=<prod> make migrate-up
```

### Fixing a broken migration

If a migration fails halfway (dirty state):
```bash
make migrate-status            # shows version + dirty flag
make migrate-force VERSION=N   # force-set to version N (the last clean version)
```
Then fix the SQL and re-run `make migrate-up`.

---

## Testing strategy

### Test tools
- **Assertions**: `github.com/stretchr/testify` — `assert.Equal`, `assert.NoError`, `assert.ErrorIs`
- **Mocks**: Manual mocks in `backend/mock/` — struct with function fields implementing port interfaces
- **Integration**: `testcontainers-go` for real PostgreSQL/Redis in tests
- No external mock generators required

### Unit tests (fast, no dependencies)
- **domain/*_test.go** — Entity validation, business rules
- **app/**/service_test.go** — Use cases with mocked ports
- **pkg/*_test.go** — Utility functions
- Run: `make test-unit`

### Integration tests (need Docker)
- **adapter/postgres/*_test.go** — Against real PostgreSQL via testcontainers
- **test/** — Full request flow tests
- Run: `make test-integration`

### Table-driven tests

Always use table-driven tests for multiple scenarios:

```go
func TestUser_New(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        uname   string
        hash    string
        role    user.Role
        wantErr error
    }{
        {
            name:  "valid agency user",
            email: "agency@example.com",
            uname: "Test Agency",
            hash:  "hashedpassword",
            role:  user.RoleAgency,
        },
        {
            name:    "empty email",
            email:   "",
            uname:   "Test",
            hash:    "hash",
            role:    user.RoleAgency,
            wantErr: user.ErrInvalidEmail,
        },
        {
            name:    "invalid role",
            email:   "test@example.com",
            uname:   "Test",
            hash:    "hash",
            role:    user.Role("invalid"),
            wantErr: user.ErrInvalidRole,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            u, err := user.New(tt.email, tt.uname, tt.hash, tt.role)
            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
                assert.Nil(t, u)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, u)
                assert.Equal(t, tt.email, u.Email)
            }
        })
    }
}
```

### Rules
- Test file lives next to the file it tests: `service.go` -> `service_test.go`
- App layer tests mock ALL ports — test logic, not infrastructure
- Name tests: `TestServiceName_MethodName_Scenario`
- Table-driven tests for multiple scenarios
- **NEVER commit with failing tests**
- **NEVER delete or skip a test to make the suite pass**

### Test -> Fix -> Retest loop

After writing tests, always run them. If they fail:
1. Read the error output
2. Fix the bug (in implementation or test, whichever is actually wrong)
3. Rerun the tests
4. Max 3 fix attempts per failing test
5. If still failing -> document in `BLOCKED-taskX.md` at project root, comment test with `// TODO: fix -- <reason>`

---

## Role-based access control

The marketplace has role-specific endpoints:

```go
// Public routes
r.Post("/api/v1/auth/register", authHandler.Register)
r.Post("/api/v1/auth/login", authHandler.Login)

// Authenticated routes (any role)
r.Group(func(r chi.Router) {
    r.Use(middleware.Auth(tokenService))
    r.Get("/api/v1/profile", profileHandler.GetProfile)
    r.Put("/api/v1/profile", profileHandler.UpdateProfile)
})

// Enterprise-only routes
r.Group(func(r chi.Router) {
    r.Use(middleware.Auth(tokenService))
    r.Use(middleware.RequireRole("enterprise"))
    r.Post("/api/v1/missions", missionHandler.Create)
})

// Agency + Provider routes
r.Group(func(r chi.Router) {
    r.Use(middleware.Auth(tokenService))
    r.Use(middleware.RequireRole("agency", "provider"))
    r.Post("/api/v1/missions/{id}/apply", missionHandler.Apply)
})

// Admin-only routes
r.Group(func(r chi.Router) {
    r.Use(middleware.Auth(tokenService))
    r.Use(middleware.RequireRole("admin"))
    r.Get("/api/v1/admin/users", adminHandler.ListUsers)
})
```

---

## Blocker policy (backend-specific)

If a Go test or implementation is stuck:
- **Test failure**: max 3 fix attempts -> then comment `// TODO: fix -- <reason>` + log in `BLOCKED-taskX.md`
- **Compilation failure**: TOP PRIORITY — fix immediately or revert last changes with `git checkout -- <files>`
- **Dependency issue** (go get fails, API changed): log in `BLOCKED-taskX.md`, move to next task
- **Never leave `go build ./...` broken** — this blocks all other tasks

---

## Commands

```bash
make run              # Start API server (loads .env automatically)
make build            # Build binary to bin/api
make test             # Run all tests
make test-unit        # Run unit tests only (short flag)
make test-integration # Run integration tests
make migrate-up       # Apply all pending migrations
make migrate-down     # Rollback last migration
make migrate-status   # Show current migration version
make seed             # Seed initial data
make mock             # Generate mocks (placeholder)
make lint             # Run go vet
make tidy             # go mod tidy
make clean            # Remove build artifacts
make dev              # Alias for make run
```
