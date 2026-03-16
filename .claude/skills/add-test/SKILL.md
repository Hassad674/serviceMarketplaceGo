---
name: add-test
description: Scaffold test files alongside features to ensure proper test coverage across all apps (backend Go, web Next.js, mobile Flutter). Accepts a feature name, file path, or "all" to identify untested code.
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# Add Test

Target: **$ARGUMENTS**

You are scaffolding test files for the B2B marketplace. Every feature must have tests before being considered complete. This skill creates properly structured tests following project conventions across all three apps.

---

## STEP 1 — Determine scope

Based on `$ARGUMENTS`, decide what to scaffold:

### If a feature name is given (e.g., "auth", "mission", "contract"):
Scaffold tests for that entire feature across all apps where it exists.

### If a file path is given (e.g., "backend/internal/domain/user/entity.go"):
Create a test file for that specific file only.

### If "all" is given:
Scan all apps for source files missing corresponding test files, then scaffold tests for each.

### Discovery commands

```bash
# Find existing backend source files for a feature
ls /home/hassad/Documents/marketplaceServiceGo/backend/internal/domain/{feature}/
ls /home/hassad/Documents/marketplaceServiceGo/backend/internal/app/{feature}/
ls /home/hassad/Documents/marketplaceServiceGo/backend/internal/handler/{feature}_handler.go

# Find existing web source files for a feature
ls /home/hassad/Documents/marketplaceServiceGo/web/src/features/{feature}/

# Find existing mobile source files for a feature
ls /home/hassad/Documents/marketplaceServiceGo/mobile/lib/features/{feature}/
```

For each source file found, check whether a corresponding test file exists. List what is **present** and what is **missing**.

---

## STEP 2 — Backend Go tests

Read the actual source files before writing tests. Never write tests against imagined APIs. Match the real constructors, method signatures, and error types.

### 2a. Domain tests (unit — entity validation, business rules)

**File**: `backend/internal/domain/{feature}/{feature}_test.go`

**Pattern**: table-driven tests with `testify/assert`. Test the constructor, validation logic, and all business methods on the entity.

```go
package {feature}_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "marketplace-backend/internal/domain/{feature}"
)

func TestNew{Entity}_ValidInput(t *testing.T) {
    tests := []struct {
        name    string
        // input fields matching the real constructor signature
        wantErr error
    }{
        {"valid input", /* valid args */, nil},
        {"missing required field", /* invalid args */, {feature}.ErrInvalid{Field}},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := {feature}.New{Entity}(/* tt fields */)
            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

**What to test**:
- Constructor with valid input returns a valid entity (non-nil, correct fields, generated UUID, timestamps set)
- Constructor rejects each invalid field with the correct sentinel error
- Business methods on the entity (status transitions, computed fields)
- Value object validation (`IsValid()` on enums, format checks)

**Rules**:
- Import only the domain package under test, testify, and stdlib
- Never mock anything in domain tests — domain has zero dependencies
- One test function per constructor or business method
- Use `assert.ErrorIs` for sentinel errors, `assert.NoError` for success paths

### 2b. Service tests (unit — mock repos, test use cases)

**File**: `backend/internal/app/{feature}/{feature}_test.go`

**Pattern**: mock all repository and service dependencies via `testify/mock`. Test each service method in isolation.

First, read the service constructor to identify all dependencies (repository interfaces, service interfaces). Then read the port interfaces to know the exact method signatures for mocking.

```go
package {feature}_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    domain{Feature} "marketplace-backend/internal/domain/{feature}"
    app{Feature} "marketplace-backend/internal/app/{feature}"
)

// Mock definitions — one per dependency interface

type Mock{Feature}Repository struct {
    mock.Mock
}

// Implement every method from the repository interface
func (m *Mock{Feature}Repository) Create(ctx context.Context, entity *domain{Feature}.{Entity}) error {
    args := m.Called(ctx, entity)
    return args.Error(0)
}

func (m *Mock{Feature}Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain{Feature}.{Entity}, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*domain{Feature}.{Entity}), args.Error(1)
}

// ... implement remaining interface methods

// Test functions — one per service method per scenario

func TestService_Create_Success(t *testing.T) {
    mockRepo := new(Mock{Feature}Repository)
    // Set up expectations
    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*{feature}.{Entity}")).Return(nil)

    svc := app{Feature}.NewService(mockRepo)
    result, err := svc.Create(context.Background(), /* valid input */)

    assert.NoError(t, err)
    assert.NotNil(t, result)
    mockRepo.AssertExpectations(t)
}

func TestService_Create_DuplicateError(t *testing.T) {
    mockRepo := new(Mock{Feature}Repository)
    mockRepo.On("Create", mock.Anything, mock.Anything).Return(domain{Feature}.ErrDuplicate{Entity})

    svc := app{Feature}.NewService(mockRepo)
    _, err := svc.Create(context.Background(), /* valid input */)

    assert.ErrorIs(t, err, domain{Feature}.ErrDuplicate{Entity})
    mockRepo.AssertExpectations(t)
}

func TestService_GetByID_NotFound(t *testing.T) {
    mockRepo := new(Mock{Feature}Repository)
    mockRepo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
        Return(nil, domain{Feature}.Err{Feature}NotFound)

    svc := app{Feature}.NewService(mockRepo)
    _, err := svc.GetByID(context.Background(), uuid.New())

    assert.ErrorIs(t, err, domain{Feature}.Err{Feature}NotFound)
    mockRepo.AssertExpectations(t)
}
```

**What to test**:
- Success path for every service method
- Each distinct error path (not found, duplicate, validation failure, repository error)
- That the service calls the correct repository methods with the correct arguments
- That the service returns properly constructed domain objects

**Rules**:
- Mock definitions must implement the FULL interface (every method), even if a test only exercises one
- Use `mock.Anything` for `context.Context` — never match on context values
- Use `mock.AnythingOfType("*{feature}.{Entity}")` when the exact value does not matter
- Call `mockRepo.AssertExpectations(t)` at the end of every test to verify all expected calls were made
- Test naming: `TestService_{MethodName}_{Scenario}`

### 2c. Handler tests (integration — test HTTP layer)

**File**: `backend/internal/handler/{feature}_handler_test.go`

**Pattern**: `httptest.NewRecorder` + Chi router. Test request decoding, response encoding, status codes, and error mapping.

```go
package handler_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/go-chi/chi/v5"
    "github.com/stretchr/testify/assert"
    "marketplace-backend/internal/handler"
)

func TestCreate{Feature}_Success(t *testing.T) {
    // Set up mock service (or use a test service with mock repo)
    // Create handler
    h := handler.New{Feature}Handler(mockSvc)

    body, _ := json.Marshal(map[string]string{
        "title": "Test",
    })
    req := httptest.NewRequest(http.MethodPost, "/{feature}s", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()

    // Use chi router for URL params
    r := chi.NewRouter()
    r.Post("/{feature}s", h.Create)
    r.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusCreated, rec.Code)
}
```

**What to test**:
- Correct status codes (201 Created, 200 OK, 404 Not Found, 400 Bad Request, 403 Forbidden)
- Response body structure matches DTO
- Invalid request body returns 400
- Domain errors are mapped to correct HTTP status codes

---

## STEP 3 — Web Next.js tests

Read the actual component and hook source files before writing tests. Match the real props, exports, and UI text.

### 3a. Component tests (Vitest + Testing Library)

**File**: `web/src/features/{feature}/components/__tests__/{component}.test.tsx`

```typescript
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { {Component} } from "../{component}"

describe("{Component}", () => {
  it("renders required elements", () => {
    render(<{Component} />)
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/mot de passe/i)).toBeInTheDocument()
  })

  it("shows validation error for empty required field", async () => {
    render(<{Component} />)
    fireEvent.click(screen.getByRole("button", { name: /soumettre/i }))
    expect(await screen.findByText(/requis/i)).toBeInTheDocument()
  })

  it("calls onSubmit with valid data", async () => {
    const onSubmit = vi.fn()
    render(<{Component} onSubmit={onSubmit} />)

    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: "test@example.com" },
    })
    fireEvent.click(screen.getByRole("button", { name: /soumettre/i }))

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({ email: "test@example.com" })
      )
    })
  })

  it("disables submit button while loading", () => {
    render(<{Component} isLoading={true} />)
    expect(screen.getByRole("button", { name: /soumettre/i })).toBeDisabled()
  })
})
```

**What to test**:
- Component renders without crashing
- All required UI elements are present (labels, buttons, inputs)
- Validation errors appear for invalid input
- Callbacks fire with correct data on user interaction
- Loading and disabled states render correctly
- Accessibility: elements have proper roles, labels, and ARIA attributes

**Rules**:
- Use `screen.getByRole` and `screen.getByLabelText` over `getByTestId` (accessible queries first)
- Use French labels when the UI is in French
- Mock Server Actions with `vi.fn()` — never call real APIs in component tests
- Create `__tests__/` directory inside the component folder

### 3b. Hook tests (Vitest)

**File**: `web/src/features/{feature}/hooks/__tests__/{hook}.test.ts`

```typescript
import { renderHook, act } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { use{Feature} } from "../use-{feature}"

describe("use{Feature}", () => {
  it("returns initial state", () => {
    const { result } = renderHook(() => use{Feature}())
    expect(result.current.data).toBeNull()
    expect(result.current.isLoading).toBe(false)
  })

  it("fetches data on mount", async () => {
    const { result } = renderHook(() => use{Feature}())
    await act(async () => {
      // trigger fetch
    })
    expect(result.current.data).toBeDefined()
  })
})
```

### 3c. E2E tests (Playwright)

**File**: `web/e2e/{feature}.spec.ts`

```typescript
import { test, expect } from "@playwright/test"

test.describe("{Feature}", () => {
  test("happy path — complete user flow", async ({ page }) => {
    await page.goto("/{feature}")
    // Step through the main user flow
    // Assert key UI states along the way
    await expect(page.getByRole("heading", { name: /{feature}/i })).toBeVisible()
  })

  test("error case — handles invalid input", async ({ page }) => {
    await page.goto("/{feature}")
    await page.getByRole("button", { name: /soumettre/i }).click()
    await expect(page.getByText(/requis/i)).toBeVisible()
  })
})
```

**E2E rules**:
- Minimum 2 tests per feature: one happy path, one error case
- Use accessible locators (`getByRole`, `getByLabel`, `getByText`)
- Test against a running dev server (not mocked)
- Keep E2E tests focused on critical flows — leave edge cases to unit tests

---

## STEP 4 — Flutter tests

Read the actual Dart source files before writing tests. Match real class names, constructors, and method signatures.

### 4a. Domain tests (unit)

**File**: `mobile/test/features/{feature}/domain/{entity}_test.dart`

```dart
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/{feature}/domain/{entity}.dart';

void main() {
  group('{Entity}', () {
    test('creates valid entity from required fields', () {
      final entity = {Entity}(
        id: '123',
        // ... required fields
      );
      expect(entity.id, '123');
    });

    test('fromJson parses correctly', () {
      final json = {
        'id': '123',
        // ... JSON fields
      };
      final entity = {Entity}.fromJson(json);
      expect(entity.id, '123');
    });

    test('toJson serializes correctly', () {
      final entity = {Entity}(id: '123');
      final json = entity.toJson();
      expect(json['id'], '123');
    });
  });
}
```

**What to test**:
- Construction with valid fields
- JSON serialization/deserialization roundtrip
- Validation methods
- Computed properties

### 4b. Repository tests (unit with mockito)

**File**: `mobile/test/features/{feature}/data/{feature}_repository_test.dart`

```dart
import 'package:flutter_test/flutter_test.dart';
import 'package:mockito/annotations.dart';
import 'package:mockito/mockito.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/features/{feature}/data/{feature}_repository.dart';

@GenerateMocks([ApiClient])
import '{feature}_repository_test.mocks.dart';

void main() {
  late Mock{ApiClient} mockApiClient;
  late {Feature}Repository repository;

  setUp(() {
    mockApiClient = MockApiClient();
    repository = {Feature}Repository(apiClient: mockApiClient);
  });

  group('{Feature}Repository', () {
    test('getAll returns list of entities', () async {
      when(mockApiClient.get('/{feature}s')).thenAnswer(
        (_) async => /* mock response */,
      );
      final result = await repository.getAll();
      expect(result, isA<List<{Entity}>>());
      verify(mockApiClient.get('/{feature}s')).called(1);
    });

    test('getById returns single entity', () async {
      when(mockApiClient.get('/{feature}s/123')).thenAnswer(
        (_) async => /* mock response */,
      );
      final result = await repository.getById('123');
      expect(result.id, '123');
    });

    test('getById throws when not found', () async {
      when(mockApiClient.get('/{feature}s/999')).thenThrow(
        NotFoundException(),
      );
      expect(
        () => repository.getById('999'),
        throwsA(isA<NotFoundException>()),
      );
    });
  });
}
```

**Rules**:
- Use `@GenerateMocks` annotation and run `dart run build_runner build` to generate mocks
- Mock the API client, not the HTTP layer
- Test success and error paths

### 4c. Widget tests

**File**: `mobile/test/features/{feature}/presentation/screens/{screen}_test.dart`

```dart
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/{feature}/presentation/screens/{screen}_screen.dart';

void main() {
  group('{Screen}Screen', () {
    testWidgets('renders key elements', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(home: {Screen}Screen()),
      );
      expect(find.text('{Expected Title}'), findsOneWidget);
    });

    testWidgets('shows loading indicator while fetching', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(home: {Screen}Screen()),
      );
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets('tapping button triggers action', (tester) async {
      await tester.pumpWidget(
        const MaterialApp(home: {Screen}Screen()),
      );
      await tester.tap(find.byType(ElevatedButton));
      await tester.pump();
      // Assert state change
    });
  });
}
```

**What to test**:
- Screen renders without crashing
- Key UI elements are present
- Loading and error states display correctly
- User interactions trigger expected behavior

---

## STEP 5 — Run all tests

After scaffolding, run the tests to verify they compile and pass (or fail with expected reasons if the feature is incomplete).

```bash
# Backend
cd /home/hassad/Documents/marketplaceServiceGo/backend && go test ./internal/domain/{feature}/... -v -count=1 -timeout 30s
cd /home/hassad/Documents/marketplaceServiceGo/backend && go test ./internal/app/{feature}/... -v -count=1 -timeout 30s

# Web (if node_modules exist)
cd /home/hassad/Documents/marketplaceServiceGo/web && [ -d node_modules ] && npx vitest run src/features/{feature}/ --reporter=verbose 2>/dev/null

# Mobile (if flutter is available)
cd /home/hassad/Documents/marketplaceServiceGo/mobile && command -v flutter >/dev/null && flutter test test/features/{feature}/ 2>/dev/null
```

If tests fail because the feature implementation is incomplete (missing methods, unfinished code), that is acceptable — note it in the report. If tests fail due to test code errors (wrong imports, typos, mismatched signatures), fix them immediately.

---

## STEP 6 — Fix failures (max 3 attempts per test)

For each failing test:

1. Read the error output carefully
2. Determine if the bug is in the **test** or the **source code**
3. Fix the actual bug (prefer fixing implementation over adjusting assertions, unless the assertion is wrong)
4. Re-run the specific failing test
5. If still failing after 3 attempts: mark with a `// TODO: fix — {reason}` comment and move on

### Do NOT:
- Delete or skip a test to make the suite pass
- Change assertions to match buggy behavior
- Add `t.Skip()` / `.skip()` / `skip:` without a clear documented reason
- Write tests that always pass regardless of implementation correctness

---

## Rules — non-negotiable

1. **Every feature MUST have tests before being considered complete**
2. **Domain/entity tests are NON-NEGOTIABLE** — they protect business rules and have zero dependencies
3. **Service tests mock ALL dependencies** — they are pure unit tests, no database, no network
4. **Handler tests verify HTTP contract** — correct status codes, response shapes, error mapping
5. **E2E tests cover the happy path + at least 1 error case**
6. **Test names describe the scenario**: `TestServiceName_MethodName_Scenario` (Go), `describe/it` blocks (TS/Dart)
7. **No skipped tests without a comment explaining why**
8. **Never delete a test to make the suite pass**
9. **Read source files before writing tests** — never guess at method signatures or constructor parameters
10. **Tests must compile independently** — if a test file is deleted, no other test file breaks

---

## Output

When finished, report:

```
# Test Scaffolding Report — {target}

## Files created

### Backend
  - internal/domain/{feature}/{feature}_test.go — X test functions
  - internal/app/{feature}/{feature}_test.go — Y test functions
  - internal/handler/{feature}_handler_test.go — Z test functions

### Web
  - src/features/{feature}/components/__tests__/{component}.test.tsx — A tests
  - src/features/{feature}/hooks/__tests__/{hook}.test.ts — B tests
  - e2e/{feature}.spec.ts — C tests

### Mobile
  - test/features/{feature}/domain/{entity}_test.dart — D tests
  - test/features/{feature}/data/{feature}_repository_test.dart — E tests
  - test/features/{feature}/presentation/screens/{screen}_test.dart — F tests

## Test results
| App     | Total | Passed | Failed | Skipped |
|---------|-------|--------|--------|---------|
| Backend |   X   |   X    |   0    |    0    |
| Web     |   Y   |   Y    |   0    |    0    |
| Mobile  |   Z   |   Z    |   0    |    0    |

## Notes
- {Any tests marked TODO with reason}
- {Any missing source files that prevented test creation}
- {Recommendations for additional test coverage}
```
