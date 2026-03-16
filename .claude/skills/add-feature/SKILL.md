---
name: add-feature
description: Scaffold a complete new backend feature across all hexagonal layers (domain, port, app, adapter, handler, migration). Use when adding new functionality like profiles, missions, contracts, reviews, etc.
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Grep, Glob, Agent
---

# Add Feature

Create the feature: **$ARGUMENTS**

You are scaffolding a new feature for the B2B marketplace backend. Follow EVERY step below in order. Domain first, HTTP last. Read existing code for patterns before writing anything.

---

## STEP 0 — Understand the request

Parse `$ARGUMENTS` to determine:
- **Feature name** (lowercase for packages, PascalCase for types, snake_case for files/tables)
- **Core entity** and its fields
- **CRUD operations** or custom use cases needed
- **Relationships** — does it reference users? Any role-specific behavior?
- **Whether it needs new database tables**

If the request is ambiguous, ask the user to clarify before proceeding.

**Known marketplace features** (for reference, avoid name collisions):
| Module | Entities | Notes |
|--------|----------|-------|
| `user` | User, Role, Email, Password | Core — other features may FK here |
| `profile` | AgencyProfile, EnterpriseProfile, ProviderProfile | Role-specific |
| `mission` | Mission, MissionStatus | Enterprise publishes, Agency/Provider applies |
| `contract` | Contract, ContractStatus | Agreement between parties |
| `message` | Conversation, Message | Real-time messaging |
| `review` | Review, Rating | Post-contract reviews |
| `notification` | Notification, NotificationType | In-app notifications |
| `payment` | Payment, Invoice, PaymentStatus | Contract payments |

---

## STEP 1 — Domain entity

Create `backend/internal/domain/{feature}/entity.go`:

**Pattern** (reference `backend/internal/domain/user/entity.go`):
- Struct with business fields + `CreatedAt`, `UpdatedAt time.Time`
- ID field is `uuid.UUID`
- Constructor `New{Entity}(...)` that validates required fields, returns `(*Entity, error)`
- Business methods on the struct
- Import ONLY Go stdlib + `github.com/google/uuid`. ZERO other imports.

```go
package {feature}

import (
    "time"

    "github.com/google/uuid"
)

type {Entity} struct {
    ID        uuid.UUID
    // ... business fields
    UserID    uuid.UUID  // FK to users, if applicable
    CreatedAt time.Time
    UpdatedAt time.Time
}

func New{Entity}(/* required params */) (*{Entity}, error) {
    // validate required fields
    if someField == "" {
        return nil, ErrInvalidField
    }
    now := time.Now()
    return &{Entity}{
        ID:        uuid.New(),
        // ... set fields
        CreatedAt: now,
        UpdatedAt: now,
    }, nil
}
```

Also create `entity_test.go` with table-driven tests for validation and business methods.

---

## STEP 2 — Domain errors

Create `backend/internal/domain/{feature}/error.go`:

**Pattern** (reference `backend/internal/domain/user/error.go`):
- Sentinel errors using `errors.New()`
- Feature-scoped: `Err{Feature}NotFound`, `ErrInvalid{Field}`, etc.

```go
package {feature}

import "errors"

var (
    Err{Feature}NotFound   = errors.New("{feature} not found")
    ErrInvalid{Field}      = errors.New("invalid {field}")
    // ... feature-specific errors
)
```

---

## STEP 3 — Value objects (if needed)

Create `backend/internal/domain/{feature}/valueobject.go` only if the feature has fields that need validation logic beyond simple checks (e.g., Status enums, Rating ranges, URL formats).

**Pattern** (reference `backend/internal/domain/user/valueobject.go`):
```go
type Status string

const (
    StatusDraft     Status = "draft"
    StatusPublished Status = "published"
    StatusClosed    Status = "closed"
)

func (s Status) IsValid() bool {
    switch s {
    case StatusDraft, StatusPublished, StatusClosed:
        return true
    }
    return false
}
```

**Skip this step if the entity has no complex value objects.**

---

## STEP 4 — Port repository interface

Create `backend/internal/port/repository/{feature}_repository.go`:

**Pattern** (reference `backend/internal/port/repository/user_repository.go`):
- Small, focused interface
- All methods take `context.Context` as first param
- Use `uuid.UUID` for IDs
- Return domain types, never SQL types
- Include only the methods actually needed for planned use cases

```go
package repository

import (
    "context"

    "github.com/google/uuid"
    "marketplace-backend/internal/domain/{feature}"
)

type {Feature}Repository interface {
    Create(ctx context.Context, entity *{feature}.{Entity}) error
    GetByID(ctx context.Context, id uuid.UUID) (*{feature}.{Entity}, error)
    Update(ctx context.Context, entity *{feature}.{Entity}) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, offset, limit int) ([]*{feature}.{Entity}, int, error)
    // Add methods like ListByUserID if the entity belongs to a user
}
```

---

## STEP 5 — Port service interfaces (if needed)

If the feature needs an external service (email, storage, real-time), add an interface in `backend/internal/port/service/{service_name}.go`.

**Only create if no existing port interface covers the need.** Check existing files:
- `port/service/hasher_service.go` — password hashing
- `port/service/token_service.go` — JWT tokens

**Skip this step if existing port interfaces are sufficient.**

---

## STEP 6 — Application service

Create `backend/internal/app/{feature}/service.go`:

**Pattern** (reference `backend/internal/app/auth/service.go`):
- Struct holds port interfaces (repository + services), injected via constructor
- Methods are use cases, not CRUD wrappers
- Use Input/Output structs for complex operations
- Returns domain types and domain errors — NEVER HTTP concepts
- Use `context.Context` on all methods

```go
package {feature}

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    domain{Feature} "marketplace-backend/internal/domain/{feature}"
    "marketplace-backend/internal/port/repository"
)

type Service struct {
    {feature}s repository.{Feature}Repository
}

func NewService({feature}s repository.{Feature}Repository) *Service {
    return &Service{{feature}s: {feature}s}
}

func (s *Service) Create(ctx context.Context, /* params */) (*domain{Feature}.{Entity}, error) {
    entity, err := domain{Feature}.New{Entity}(/* params */)
    if err != nil {
        return nil, err
    }
    if err := s.{feature}s.Create(ctx, entity); err != nil {
        return nil, fmt.Errorf("creating {feature}: %w", err)
    }
    return entity, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*domain{Feature}.{Entity}, error) {
    return s.{feature}s.GetByID(ctx, id)
}
```

Also create `service_test.go` — unit tests with mocked repository. Use table-driven tests. Name: `TestService_MethodName_Scenario`.

---

## STEP 7 — PostgreSQL adapter

Create `backend/internal/adapter/postgres/{feature}_repository.go`:

**Pattern** (reference `backend/internal/adapter/postgres/user_repository.go`):
- Struct holds `*sql.DB`
- Implements the repository interface
- Pure SQL with `$1, $2, $3` parameters — NEVER string concatenation
- Context timeout on every query: `context.WithTimeout(ctx, queryTimeout)`
- Map `sql.ErrNoRows` to domain error (e.g., `{feature}.Err{Feature}NotFound`)
- Map pq unique violation `"23505"` to domain duplicate error
- Use `QueryRowContext` / `ExecContext` / `QueryContext` — always with context

```go
package postgres

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/lib/pq"

    "marketplace-backend/internal/domain/{feature}"
)

const queryTimeout = 5 * time.Second

type {Feature}Repository struct {
    db *sql.DB
}

func New{Feature}Repository(db *sql.DB) *{Feature}Repository {
    return &{Feature}Repository{db: db}
}

func (r *{Feature}Repository) Create(ctx context.Context, e *{feature}.{Entity}) error {
    ctx, cancel := context.WithTimeout(ctx, queryTimeout)
    defer cancel()

    _, err := r.db.ExecContext(ctx,
        `INSERT INTO {table} (id, /* columns */) VALUES ($1, $2, ...)`,
        e.ID, /* fields */,
    )
    if err != nil {
        var pqErr *pq.Error
        if errors.As(err, &pqErr) && pqErr.Code == "23505" {
            return {feature}.ErrDuplicate{Entity}
        }
        return fmt.Errorf("failed to create {feature}: %w", err)
    }
    return nil
}

func (r *{Feature}Repository) GetByID(ctx context.Context, id uuid.UUID) (*{feature}.{Entity}, error) {
    ctx, cancel := context.WithTimeout(ctx, queryTimeout)
    defer cancel()

    e := &{feature}.{Entity}{}
    err := r.db.QueryRowContext(ctx,
        `SELECT /* columns */ FROM {table} WHERE id = $1`, id,
    ).Scan(/* fields */)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, {feature}.Err{Feature}NotFound
        }
        return nil, fmt.Errorf("failed to get {feature} by id: %w", err)
    }
    return e, nil
}
```

---

## STEP 8 — SQL migration

Create migration files using the next available number (check `backend/migrations/` for the latest):
- `backend/migrations/{NNN}_create_{feature}.up.sql`
- `backend/migrations/{NNN}_create_{feature}.down.sql`

**SQL conventions** (reference `backend/migrations/001_create_users.up.sql`):
- UUID primary key: `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- Timestamps: `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- CHECK constraints for enum columns: `CHECK (status IN ('draft', 'published', 'closed'))`
- Foreign key to users ONLY: `user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE`
- **NO cross-feature foreign keys** — store other feature IDs as plain UUID columns without FK constraint
- Index all foreign keys and frequently queried columns
- Partial indexes where appropriate: `WHERE ... IS NOT NULL`
- Reuse the existing `update_updated_at()` trigger function (defined in migration 001)

```sql
-- up
CREATE TABLE {feature_table} (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- ... feature columns (TEXT, not VARCHAR)
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_{feature_table}_user_id ON {feature_table}(user_id);

CREATE TRIGGER {feature_table}_updated_at
    BEFORE UPDATE ON {feature_table}
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
```

```sql
-- down (reverse order, use IF EXISTS)
DROP TRIGGER IF EXISTS {feature_table}_updated_at ON {feature_table};
DROP TABLE IF EXISTS {feature_table};
```

---

## STEP 9 — Request/Response DTOs

Create `backend/internal/handler/dto/request/{feature}.go`:
```go
package request

type Create{Feature}Request struct {
    // ... fields from HTTP request body with json tags
    Title string `json:"title"`
}
```

Create or update `backend/internal/handler/dto/response/{feature}.go`:
```go
package response

import "marketplace-backend/internal/domain/{feature}"

type {Feature}Response struct {
    ID        string `json:"id"`
    // ... public fields with json tags
    CreatedAt string `json:"created_at"`
}

func New{Feature}Response(e *{feature}.{Entity}) {Feature}Response {
    return {Feature}Response{
        ID:        e.ID.String(),
        // ... map fields
        CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z"),
    }
}
```

---

## STEP 10 — HTTP handler

Create `backend/internal/handler/{feature}_handler.go`:

**Pattern** (reference `backend/internal/handler/auth_handler.go`):
- Handler struct holds `*app.Service`
- Each method: decode request -> validate -> call service -> encode response
- Thin layer — NO business logic
- Use `validator.DecodeJSON(r, &req)` for decoding
- Use `validator.ValidateRequired(...)` for field presence checks
- Use `res.JSON(w, statusCode, data)` for success responses
- Use `res.Error(w, statusCode, code, message)` for error responses
- Create a `handle{Feature}Error(w, err)` function mapping domain errors to HTTP

```go
package handler

import (
    "errors"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"

    app{Feature} "marketplace-backend/internal/app/{feature}"
    domain{Feature} "marketplace-backend/internal/domain/{feature}"
    "marketplace-backend/internal/handler/dto/response"
    "marketplace-backend/internal/handler/middleware"
    "marketplace-backend/pkg/validator"
    res "marketplace-backend/pkg/response"
)

type {Feature}Handler struct {
    svc *app{Feature}.Service
}

func New{Feature}Handler(svc *app{Feature}.Service) *{Feature}Handler {
    return &{Feature}Handler{svc: svc}
}

// ... handler methods

func handle{Feature}Error(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, domain{Feature}.Err{Feature}NotFound):
        res.Error(w, http.StatusNotFound, "{feature}_not_found", err.Error())
    // ... map other domain errors
    default:
        res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
    }
}
```

---

## STEP 11 — Register routes

Update `backend/internal/handler/router.go`:

### 11a. Add the service to RouterDeps
```go
type RouterDeps struct {
    // ... existing deps
    {Feature} *{Feature}Handler
}
```

### 11b. Register routes in the appropriate group
```go
// Inside r.Route("/api/v1", ...) — in a protected group:
r.Route("/{feature}s", func(r chi.Router) {
    r.Use(middleware.Auth(deps.TokenService))
    r.Post("/", deps.{Feature}.Create)
    r.Get("/", deps.{Feature}.List)
    r.Get("/{id}", deps.{Feature}.GetByID)
    r.Put("/{id}", deps.{Feature}.Update)
    r.Delete("/{id}", deps.{Feature}.Delete)
})
```

Use role-restricted groups if the feature is role-specific:
```go
r.Group(func(r chi.Router) {
    r.Use(middleware.Auth(deps.TokenService))
    r.Use(middleware.RequireRole("enterprise"))
    r.Post("/missions", deps.Mission.Create)
})
```

---

## STEP 12 — Wire dependencies in main.go

Update `backend/cmd/api/main.go`:

```go
// Add imports
import (
    app{Feature} "marketplace-backend/internal/app/{feature}"
)

// In main(), after existing wiring:
{feature}Repo := postgres.New{Feature}Repository(db)
{feature}Svc := app{Feature}.NewService({feature}Repo)
{feature}Handler := handler.New{Feature}Handler({feature}Svc)

// Add to RouterDeps:
r := handler.NewRouter(handler.RouterDeps{
    // ... existing
    {Feature}: {feature}Handler,
})
```

---

## STEP 13 — Verify

```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend && go build ./...
```

### Independence checklist:
- [ ] Domain imports ONLY Go stdlib + `github.com/google/uuid`
- [ ] Port imports only domain/ and stdlib
- [ ] App service depends on port interfaces, not concrete adapters
- [ ] No cross-feature imports (domain/{x} never imports domain/{y})
- [ ] Database migration only FKs to `users` table
- [ ] Handler is thin — decode, validate, call service, encode
- [ ] All SQL uses `$1, $2` parameters, never string concatenation
- [ ] All DB queries use `context.WithTimeout`
- [ ] No file exceeds 600 lines
- [ ] No function exceeds 50 lines
- [ ] Removing this feature's files + main.go lines = everything still compiles

If any check fails, fix it before finishing.

---

## Output

When finished, report:
1. **Files created** (grouped by layer)
2. **Files modified** (router.go, main.go)
3. **Migration file names**
4. **Decisions made** and why
5. **Independence verification** result

Example:
```
Created feature: review

Files created:
  internal/domain/review/entity.go
  internal/domain/review/entity_test.go
  internal/domain/review/error.go
  internal/domain/review/valueobject.go
  internal/port/repository/review_repository.go
  internal/app/review/service.go
  internal/app/review/service_test.go
  internal/adapter/postgres/review_repository.go
  internal/handler/review_handler.go
  internal/handler/dto/request/review.go
  internal/handler/dto/response/review.go
  migrations/004_create_reviews.up.sql
  migrations/004_create_reviews.down.sql

Files modified:
  internal/handler/router.go — added ReviewHandler to RouterDeps, registered routes
  cmd/api/main.go — wired reviewRepo -> reviewSvc -> reviewHandler

Independence: PASS — no cross-feature imports, FK only to users
```
