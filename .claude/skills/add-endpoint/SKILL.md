---
name: add-endpoint
description: Add a new API endpoint (use case) to an existing feature. Lighter than /add-feature — scaffolds handler method, DTO, service method, and optional repository method for a single operation.
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# Add Endpoint

Add endpoint: **$ARGUMENTS**

You are adding a new endpoint to an existing feature in the marketplace backend. This is lighter than `/add-feature` — the domain, ports, and adapter already exist. You are adding a new use case.

---

## STEP 0 — Parse the request

Determine from `$ARGUMENTS`:
- **Feature name** — which existing feature (auth, user, profile, mission, contract, message, review, notification, payment)
- **Operation** — what the endpoint does (apply-to-mission, approve-contract, submit-review, etc.)
- **HTTP method + path** — derive from the operation (POST /api/v1/missions/{id}/apply, PUT /api/v1/contracts/{id}/approve, etc.)
- **Access control** — public, authenticated (any role), or role-restricted (enterprise, agency, provider, admin)

---

## STEP 1 — Read existing feature code

Before writing anything, read the current state of the feature:

1. **Domain entity** — `backend/internal/domain/{feature}/entity.go` — understand the struct and existing methods
2. **Domain errors** — `backend/internal/domain/{feature}/error.go` — check available error types
3. **Repository interface** — `backend/internal/port/repository/{feature}_repository.go` — check if new data methods are needed
4. **App service** — `backend/internal/app/{feature}/service.go` — understand existing methods and dependencies
5. **Handler** — `backend/internal/handler/{feature}_handler.go` — understand existing handler structure
6. **Router** — `backend/internal/handler/router.go` — see where routes are registered
7. **DTOs** — `backend/internal/handler/dto/request/{feature}.go` and `dto/response/{feature}.go`

---

## STEP 2 — Domain changes (if needed)

If the new use case requires new business logic on the entity, add methods to the domain:

```go
// In domain/{feature}/entity.go
func (e *{Entity}) Approve() error {
    if e.Status != StatusPending {
        return ErrInvalidStatusTransition
    }
    e.Status = StatusApproved
    e.UpdatedAt = time.Now()
    return nil
}
```

If the new use case requires a new domain error, add it to `error.go`:
```go
var ErrInvalidStatusTransition = errors.New("invalid status transition")
```

Add tests for any new domain method in `entity_test.go`.

If the new use case requires a new field on the entity, add it and create a migration with `/add-migration`.

**Skip this step if the existing domain is sufficient.**

---

## STEP 3 — Repository changes (if needed)

If the use case needs a new data access method:

### 3a. Add to port interface
```go
// In port/repository/{feature}_repository.go
type {Feature}Repository interface {
    // ... existing methods
    ListByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*{feature}.{Entity}, int, error)
}
```

### 3b. Implement in adapter
```go
// In adapter/postgres/{feature}_repository.go
func (r *{Feature}Repository) ListByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*{feature}.{Entity}, int, error) {
    ctx, cancel := context.WithTimeout(ctx, queryTimeout)
    defer cancel()

    var total int
    err := r.db.QueryRowContext(ctx,
        `SELECT COUNT(*) FROM {table} WHERE user_id = $1`, userID,
    ).Scan(&total)
    if err != nil {
        return nil, 0, fmt.Errorf("failed to count {feature}s: %w", err)
    }

    rows, err := r.db.QueryContext(ctx,
        `SELECT /* columns */ FROM {table} WHERE user_id = $1
         ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
        userID, limit, offset,
    )
    // ... scan rows
    return results, total, nil
}
```

**Rules for adapter SQL:**
- Always use `context.WithTimeout(ctx, queryTimeout)` with `defer cancel()`
- Parameterized queries only: `$1, $2, $3` — never string concatenation
- Map `sql.ErrNoRows` to the appropriate domain error
- Map pq unique violation `"23505"` to a domain duplicate error

**Skip this step if existing repository methods are sufficient.**

---

## STEP 4 — Service method

Add the new use case to `backend/internal/app/{feature}/service.go`:

**Pattern** (reference `backend/internal/app/auth/service.go`):
```go
// Use Input struct for endpoints with multiple parameters
type Apply{Action}Input struct {
    UserID    uuid.UUID
    {Feature}ID uuid.UUID
    // ... other fields
}

func (s *Service) Apply{Action}(ctx context.Context, input Apply{Action}Input) (*domain{Feature}.{Entity}, error) {
    entity, err := s.{feature}s.GetByID(ctx, input.{Feature}ID)
    if err != nil {
        return nil, err
    }

    // Business logic — call domain methods
    if err := entity.Apply(input.UserID); err != nil {
        return nil, err
    }

    if err := s.{feature}s.Update(ctx, entity); err != nil {
        return nil, fmt.Errorf("updating {feature}: %w", err)
    }

    return entity, nil
}
```

**Rules:**
- Accept primitive types, uuid.UUID, or Input structs as parameters
- Return domain types and domain errors
- No HTTP concepts (no status codes, no request/response structs)
- Use the existing injected dependencies (repository + services)
- Wrap infrastructure errors with `fmt.Errorf("context: %w", err)`

---

## STEP 5 — Service test

Add test for the new method in `backend/internal/app/{feature}/service_test.go`:

```go
func TestService_{MethodName}_Success(t *testing.T) {
    // Setup mocks
    // Call service method
    // Assert result
}

func TestService_{MethodName}_{Feature}NotFound(t *testing.T) {
    // Setup mocks to return ErrNotFound
    // Call service method
    // Assert error returned
}
```

Use table-driven tests when there are multiple scenarios. Name: `TestService_MethodName_Scenario`.

---

## STEP 6 — Request DTO

Add the request struct in `backend/internal/handler/dto/request/{feature}.go`:

```go
type Apply{Action}Request struct {
    // Only fields from the HTTP request body — URL params handled separately
    Message string `json:"message"`
}
```

If the file does not exist yet, create it. If it exists, append the new struct.

---

## STEP 7 — Response DTO (if needed)

If the endpoint returns feature-specific data not already covered, add a response struct or helper in `backend/internal/handler/dto/response/{feature}.go`.

For simple success/error responses, use the existing `res.JSON()` and `res.Error()` from `marketplace-backend/pkg/response`.

---

## STEP 8 — Handler method

Add the handler method to `backend/internal/handler/{feature}_handler.go`:

**Pattern** (reference `backend/internal/handler/auth_handler.go`):
```go
func (h *{Feature}Handler) Apply{Action}(w http.ResponseWriter, r *http.Request) {
    // 1. Extract auth context
    userID, ok := middleware.GetUserID(r.Context())
    if !ok {
        res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
        return
    }

    // 2. Extract URL params
    idStr := chi.URLParam(r, "id")
    id, err := uuid.Parse(idStr)
    if err != nil {
        res.Error(w, http.StatusBadRequest, "invalid_id", "invalid {feature} ID")
        return
    }

    // 3. Decode request body (if POST/PUT/PATCH)
    var req request.Apply{Action}Request
    if err := validator.DecodeJSON(r, &req); err != nil {
        res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
        return
    }

    // 4. Call service
    entity, err := h.svc.Apply{Action}(r.Context(), app{Feature}.Apply{Action}Input{
        UserID:      userID,
        {Feature}ID: id,
    })
    if err != nil {
        handle{Feature}Error(w, err)
        return
    }

    // 5. Respond
    res.JSON(w, http.StatusOK, response.New{Feature}Response(entity))
}
```

**THE THIN HANDLER RULE:** The handler NEVER contains business logic. Its only responsibilities are:
1. Decode the HTTP request (body, URL params, query params, auth context)
2. Validate request structure (required fields present, valid UUID format)
3. Call ONE service method
4. Encode the response (success DTO or error mapping)

If you find yourself writing `if/else` chains with business rules in the handler, move that logic to the service or domain layer.

---

## STEP 9 — Update error mapping

If new domain errors were added in STEP 2, update the `handle{Feature}Error` function:

```go
func handle{Feature}Error(w http.ResponseWriter, err error) {
    switch {
    // ... existing cases
    case errors.Is(err, domain{Feature}.ErrInvalidStatusTransition):
        res.Error(w, http.StatusConflict, "invalid_status", err.Error())
    // ...
    }
}
```

---

## STEP 10 — Register route

Add the route in `backend/internal/handler/router.go`:

```go
// In the appropriate group:
r.Post("/missions/{id}/apply", deps.Mission.ApplyAction)
```

- Public endpoints go outside middleware groups
- Authenticated endpoints go inside `r.Use(middleware.Auth(deps.TokenService))`
- Role-restricted endpoints add `r.Use(middleware.RequireRole("enterprise"))` etc.
- Use RESTful conventions: `GET /resources`, `POST /resources`, `GET /resources/{id}`, `POST /resources/{id}/action`

---

## STEP 11 — Verify

```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend && go build ./...
```

### Checklist:
- [ ] Service method has no HTTP concepts
- [ ] Handler is thin — decode, validate, call ONE service method, encode
- [ ] New repository method (if any) uses parameterized SQL with context timeout
- [ ] Test covers success + primary error cases
- [ ] Route is in the correct group (public/authenticated/role-restricted)
- [ ] No cross-feature imports introduced
- [ ] Domain error added to handler error mapping function

---

## Output

Report:
1. **Endpoint** — `METHOD /path` (public/authenticated/role: X)
2. **Files created** — new files
3. **Files modified** — existing files with changes
4. **Test coverage** — tests written
5. **Domain changes** — new entity methods or errors (if any)

Example:
```
Added endpoint: POST /api/v1/missions/{id}/apply (authenticated, role: agency,provider)

Modified:
  internal/domain/mission/entity.go — added Apply() method
  internal/domain/mission/error.go — added ErrAlreadyApplied
  internal/domain/mission/entity_test.go — added 2 tests
  internal/port/repository/mission_repository.go — added HasApplication() method
  internal/adapter/postgres/mission_repository.go — implemented HasApplication()
  internal/app/mission/service.go — added ApplyToMission()
  internal/app/mission/service_test.go — added 3 tests
  internal/handler/mission_handler.go — added Apply handler + error mapping
  internal/handler/dto/request/mission.go — added ApplyToMissionRequest
  internal/handler/router.go — registered POST /api/v1/missions/{id}/apply

Tests: 5 new (domain: 2, service: 3)
```
