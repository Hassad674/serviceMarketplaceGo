---
name: add-adapter
description: Add or swap an adapter (external service implementation) following hexagonal architecture. Use when integrating a new provider (storage, email, video, cache, payment) or replacing an existing one.
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# Add Adapter

Create or swap adapter: **$ARGUMENTS**

You are adding a new adapter to the marketplace backend following hexagonal architecture. An adapter implements a port interface — the rest of the codebase is completely unaware of which concrete provider is used.

---

## STEP 0 — Parse the request

Determine from `$ARGUMENTS`:
- **Adapter category**: storage, email, video, cache, payment, websocket, or new category
- **Provider name**: s3, minio, resend, sendgrid, livekit, redis, stripe, etc.
- **Action**: new adapter, or replacing an existing one

---

## STEP 1 — Identify the port interface

Find the interface this adapter must implement.

### Check existing port interfaces
Read files in `backend/internal/port/service/`:

```bash
ls /home/hassad/Documents/marketplaceServiceGo/backend/internal/port/service/
```

**Known port interfaces:**
| Category | File | Interface |
|----------|------|-----------|
| Hashing | `port/service/hasher_service.go` | `HasherService` |
| Token/JWT | `port/service/token_service.go` | `TokenService` |

Read the interface file to get the exact method signatures.

### If no interface exists for this adapter category

Create one in `backend/internal/port/service/{category}.go`:

```go
package service

import (
    "context"
    "io"
    "time"
)

// {Category}Service defines the contract for {category} operations.
// Implementations: {provider1}, {provider2}
type {Category}Service interface {
    // Define methods based on what the app layer actually needs
    // Keep the interface small and focused
}
```

**Port interface rules:**
- Import ONLY from `domain/` packages and Go stdlib
- Use domain types in method signatures, not provider types
- Keep it small — define only what the app layer actually calls
- All methods take `context.Context` as first parameter
- Use `io.Reader` for file uploads, not provider-specific types

**Common marketplace port interfaces to create:**

**StorageService** (for S3/MinIO/R2):
```go
type StorageService interface {
    Upload(ctx context.Context, key string, data io.Reader, contentType string, size int64) (string, error)
    Delete(ctx context.Context, key string) error
    GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}
```

**EmailService** (for Resend/SendGrid):
```go
type EmailService interface {
    Send(ctx context.Context, to, subject, htmlBody string) error
    SendTemplate(ctx context.Context, to, templateID string, data map[string]any) error
}
```

**VideoService** (for LiveKit):
```go
type VideoService interface {
    CreateRoom(ctx context.Context, roomName string) error
    GenerateToken(ctx context.Context, roomName, participantID, participantName string) (string, error)
    DeleteRoom(ctx context.Context, roomName string) error
}
```

**CacheService** (for Redis):
```go
type CacheService interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key, value string, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
}
```

---

## STEP 2 — Create the adapter package

Create `backend/internal/adapter/{provider}/` with the following files:

### 2a. Client setup — `client.go`

```go
package {provider}

import (
    // provider SDK imports
)

// Client holds the {provider} SDK client and configuration.
type Client struct {
    // SDK client, API key, base URL, etc.
}

// NewClient creates a configured {provider} client.
// All configuration is passed in — no env var reading here.
func NewClient(apiKey string /* other config */) (*Client, error) {
    // Initialize SDK or HTTP client
    // Return error if configuration is invalid
    return &Client{
        // ...
    }, nil
}
```

### 2b. Interface implementation — `{category}.go`

**Pattern** (reference `backend/internal/adapter/postgres/user_repository.go` for style):

```go
package {provider}

import (
    "context"
    "fmt"
    // provider SDK
)

// {Category}Service implements service.{Category}Service using {Provider}.
type {Category}Service struct {
    client *Client
}

// New{Category}Service creates a new {provider}-backed {category} service.
func New{Category}Service(client *Client) *{Category}Service {
    return &{Category}Service{client: client}
}

// Implement every method from the port interface.
func (s *{Category}Service) Upload(ctx context.Context, key string, data io.Reader, contentType string, size int64) (string, error) {
    // 1. Map domain types -> provider SDK types
    // 2. Call provider API with context for cancellation
    // 3. Map provider response -> domain types
    // 4. Return domain types and domain errors (never provider-specific errors)
    return url, nil
}
```

### Adapter implementation rules:

- The adapter imports domain types and the provider's SDK — nothing else from `internal/`
- **Never import another adapter** (no `adapter/postgres` from `adapter/s3`)
- **Never import `app/` or `handler/`**
- Return domain errors, not provider-specific errors. Wrap with context:
  ```go
  return fmt.Errorf("uploading to {provider}: %w", err)
  ```
- Use `context.Context` on all methods for timeout/cancellation
- Handle provider errors gracefully — map to domain errors when possible
- Log provider-specific details at debug level, return generic errors to caller

---

## STEP 3 — Add configuration

Update `backend/internal/config/config.go`:

```go
// In Config struct — add new fields:
{Provider}Key      string
{Provider}Secret   string
{Provider}Endpoint string  // if applicable (e.g., MinIO endpoint)
{Provider}Bucket   string  // if applicable

// In Load() — read from environment:
{Provider}Key:      getEnv("{PROVIDER}_API_KEY", ""),
{Provider}Secret:   getEnv("{PROVIDER}_SECRET", ""),
{Provider}Endpoint: getEnv("{PROVIDER}_ENDPOINT", ""),
{Provider}Bucket:   getEnv("{PROVIDER}_BUCKET", ""),
```

Read the existing `config.go` first to match the exact pattern for `getEnv` or `os.Getenv` usage.

Only add config fields that don't already exist.

---

## STEP 4 — Update .env.example

Add the new environment variables to `backend/.env.example` (create if it does not exist):

```bash
# {Provider} ({Category})
{PROVIDER}_API_KEY=
{PROVIDER}_SECRET=
{PROVIDER}_ENDPOINT=
{PROVIDER}_BUCKET=
```

---

## STEP 5 — Install SDK dependency

If the provider has a Go SDK:
```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend
go get {sdk-package}
go mod tidy
```

**Common SDKs for marketplace adapters:**
| Provider | Package |
|----------|---------|
| MinIO/S3 | `github.com/minio/minio-go/v7` |
| AWS S3 | `github.com/aws/aws-sdk-go-v2/service/s3` |
| Resend | `github.com/resend/resend-go/v2` |
| Redis | `github.com/redis/go-redis/v9` |
| LiveKit | `github.com/livekit/server-sdk-go/v2` |
| Stripe | `github.com/stripe/stripe-go/v82` |

If no official SDK exists, use `net/http` directly. Keep it simple.

---

## STEP 6 — Wire in main.go

Update `backend/cmd/api/main.go`:

### For a NEW adapter (no existing provider):
```go
// Add import
import (
    {provider}Adapter "marketplace-backend/internal/adapter/{provider}"
)

// In main(), after DB connection:
{provider}Client, err := {provider}Adapter.NewClient(cfg.{Provider}Key)
if err != nil {
    slog.Error("failed to create {provider} client", "error", err)
    os.Exit(1)
}
{category}Svc := {provider}Adapter.New{Category}Service({provider}Client)

// Pass to the app service that needs it:
featureSvc := appFeature.NewService(featureRepo, {category}Svc)
```

### For SWAPPING an existing adapter:
Change only the instantiation lines in main.go:

```go
// BEFORE:
minioClient, err := minio.NewClient(cfg.MinIOEndpoint, cfg.MinIOKey, cfg.MinIOSecret)
storageSvc := minio.NewStorageService(minioClient)

// AFTER:
s3Client, err := s3.NewClient(cfg.S3Key, cfg.S3Secret, cfg.S3Region)
storageSvc := s3.NewStorageService(s3Client)
```

**Nothing else changes.** The app service receives the same interface. This is the power of hexagonal architecture.

---

## STEP 7 — Compile-time interface check

In the adapter's implementation file, add a compile-time assertion:

```go
import "marketplace-backend/internal/port/service"

// Compile-time check: {Category}Service implements service.{Category}Service
var _ service.{Category}Service = (*{Category}Service)(nil)
```

This ensures the adapter won't compile if it's missing a method from the interface.

---

## STEP 8 — Verify

```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend && go build ./...
```

### Verify the swap promise:
- [ ] Only `cmd/api/main.go` changed for wiring (plus config.go for new env vars)
- [ ] No `app/` or `handler/` files were modified
- [ ] No `domain/` files were modified
- [ ] The adapter implements the full port interface (compile-time check passes)
- [ ] The adapter does NOT import other adapters
- [ ] The adapter does NOT import `app/` or `handler/`
- [ ] Config is centralized in `config.go`
- [ ] Environment variables documented in `.env.example`

---

## Output

Report:
1. **Files created** — adapter package files
2. **Files modified** — config.go, main.go, .env.example
3. **Interface implemented** — which port interface + all methods
4. **Dependencies added** — new Go modules (if any)
5. **Swap instructions** — if replacing, what to change back

Example:
```
Created adapter: minio (storage)

Files created:
  internal/adapter/minio/client.go
  internal/adapter/minio/storage.go

Files modified:
  internal/config/config.go — added MinIOEndpoint, MinIOKey, MinIOSecret, MinIOBucket
  cmd/api/main.go — wired minioClient -> storageSvc
  .env.example — added MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, MINIO_BUCKET

Port interface: service.StorageService
  - Upload(ctx, key, data, contentType, size) (string, error)
  - Delete(ctx, key) error
  - GetPresignedURL(ctx, key, expiry) (string, error)

Dependencies: github.com/minio/minio-go/v7 v7.0.80

To swap to AWS S3:
  1. Create internal/adapter/s3/ implementing service.StorageService
  2. In main.go: change minio.NewStorageService -> s3.NewStorageService
  3. Update config.go with S3-specific env vars
```
