# Mobile — Flutter 3.16+ / Dart 3.2+

## Purpose

Standalone Flutter mobile app for the B2B marketplace. Providers, agencies, and enterprises manage missions, messaging, contracts, and profiles on the go. Communicates with the Go backend exclusively via REST API. English-language UI.

## Tech Stack

| Layer | Technology | Version |
|-------|-----------|---------|
| Framework | Flutter | >= 3.16.0 |
| Language | Dart | >= 3.2.0 < 4.0.0 |
| State management | flutter_riverpod | ^2.4.9 |
| Navigation | go_router | ^13.0.0 |
| HTTP client | dio | ^5.4.0 |
| Secure storage | flutter_secure_storage | ^9.0.0 |
| Code generation | freezed + json_serializable | ^2.4.6 / ^6.7.1 |
| Image caching | cached_network_image | ^3.3.1 |
| Connectivity | connectivity_plus | ^6.0.1 |
| Testing | flutter_test + mockito | built-in / ^5.4.4 |

## Architecture

Clean Architecture with feature-based folder structure. Each feature module contains three layers that enforce a strict dependency rule.

```
lib/
├── main.dart                      -> Entry point + ProviderScope
├── core/
│   ├── theme/app_theme.dart       -> Light/dark themes, AppColors extension
│   ├── router/app_router.dart     -> GoRouter with auth guards
│   ├── network/
│   │   ├── api_client.dart        -> Dio + JWT interceptors + token refresh
│   │   └── api_exception.dart     -> Structured API error handling
│   ├── storage/
│   │   └── secure_storage.dart    -> Encrypted token + user cache storage
│   └── utils/
│       └── extensions.dart        -> DateTime, String, Currency helpers
│
├── features/
│   ├── auth/
│   │   ├── data/                  -> Repository implementations, DTOs
│   │   ├── domain/
│   │   │   ├── entities/          -> Freezed data classes
│   │   │   ├── repositories/      -> Abstract repository interfaces
│   │   │   └── usecases/         -> Business logic orchestration
│   │   └── presentation/
│   │       ├── providers/         -> Riverpod state notifiers
│   │       ├── screens/           -> Full-page widgets
│   │       └── widgets/           -> Reusable UI components
│   │
│   ├── mission/                   -> Same data/domain/presentation structure
│   ├── messaging/
│   ├── enterprise/
│   ├── provider_profile/
│   ├── project/
│   ├── invoice/
│   ├── notification/
│   ├── review/
│   └── referral/
│
└── generated/                     -> Auto-generated code (*.g.dart, *.freezed.dart)
```

### Dependency Rule

```
presentation -> domain <- data
```

- **domain/** has zero external imports (pure Dart). Entities, repository interfaces, and use cases only.
- **data/** implements domain interfaces, calls `ApiClient`, maps DTOs to domain entities.
- **presentation/** uses Riverpod providers that depend on domain use cases. Never imports from `data/` directly.

---

## Code Quality Standards

These limits are non-negotiable. They keep the codebase maintainable as it grows.

| Metric | Limit | Rationale |
|--------|-------|-----------|
| File length | 600 lines max | Split into separate files if exceeded |
| Function/method body | 50 lines max | Extract helper methods |
| Constructor parameters | 4 positional max | Use named parameters for additional params |
| `build()` method | 100 lines max | Extract sub-widgets into private methods or separate widgets |
| Widget nesting depth | 3 levels max | Extract at level 4 into a named widget |

### Dart-Specific Rules
- **`const` constructors everywhere possible** — the analyzer will tell you when you can add `const`. Always do it.
- **`final` for all fields that do not change** — mutable fields are the exception, not the rule.
- **Prefer `const` widget instantiation** — `const Text('Hello')` not `Text('Hello')`.
- **No `dynamic` type** — use proper types or generics. `dynamic` hides bugs.
- **No `print()` in production code** — use a logger or `debugPrint()` for development.

---

## SOLID Principles (Flutter Adaptation)

- **S — Single Responsibility**: One widget = one UI concern. One provider = one state concern. A `MissionCard` renders a card; it does not fetch missions or manage navigation.
- **O — Open/Closed**: Widgets are extensible via constructor parameters, not by modifying their internals. Add a new `variant` parameter rather than forking the widget.
- **L — Liskov Substitution**: Repository implementations are interchangeable. `MockAuthRepository` can replace `AuthRepositoryImpl` in tests without changing any consumer code.
- **I — Interface Segregation**: Small repository interfaces per feature. `AuthRepository` has auth methods only; it does not also handle profile updates.
- **D — Dependency Inversion**: Presentation depends on domain interfaces (abstract repositories, use cases). Never import from `data/` in `presentation/`. Wiring happens through Riverpod providers.

---

## STUPID Anti-Patterns (What to Avoid)

- **No `setState` in complex widgets** — use Riverpod. `setState` is only acceptable for trivial local UI state (form field focus, animation controllers, toggle visibility).
- **No business logic in widgets** — extract to providers or use cases. Widgets render; providers compute.
- **No deep widget nesting** — extract sub-widgets at 3 levels. Deeply nested `build()` methods are unreadable and untestable.
- **No hardcoded strings** — use constants or l10n. UI text goes through localization. Route paths, API endpoints, and storage keys are constants.
- **No Singleton services** — use Riverpod providers for dependency injection. Singletons are hard to test and create hidden coupling.
- **No God providers** — a provider that manages auth + navigation + profile + settings is doing too much. One provider per concern.

---

## State Management — Riverpod 2

All app state flows through Riverpod providers. This is the single source of truth for state management.

### Provider Types and When to Use Them

| Provider Type | Use Case | Example |
|---------------|----------|---------|
| `Provider` | Dependency injection, computed values | Repository instances, API client |
| `StateNotifierProvider` | Complex mutable state with methods | Auth state, form state |
| `AsyncNotifierProvider` | Async operations with loading/error/data | Fetching missions, submitting forms |
| `FutureProvider` | Simple one-shot async data | User profile fetch |
| `StreamProvider` | Real-time data | Messaging, notifications |

### Rules
- **`ref.watch()` in `build()`** — for reactive rebuilds. This is the default.
- **`ref.read()` outside `build()`** — for one-off actions (button press callbacks, initialization).
- **Never `ref.read()` in `build()`** — this breaks reactivity and causes stale data bugs.
- Providers are declared at the top of their respective files, not in a global file.
- Use `AsyncValue` pattern matching (`when`, `maybeWhen`) for all async state rendering.

### Auth Provider
The `authProvider` is a `StateNotifierProvider<AuthNotifier, AuthState>` that manages login, registration, logout, and session restoration. A convenience `authStateProvider` wraps it as `AsyncValue` for the router's redirect logic.

---

## Navigation — GoRouter

Defined in `core/router/app_router.dart`.

- **Auth guard**: redirects unauthenticated users to `/login`.
- **Auth route redirect**: redirects authenticated users away from `/login` and `/register`.
- **ShellRoute**: wraps authenticated screens with a persistent bottom navigation bar.
- **`RoutePaths` class**: centralized string constants to avoid magic strings.

Routes: `/login`, `/register`, `/dashboard`, `/messaging`, `/missions`, `/profile`, `/settings`.

---

## API Client — Dio

Defined in `core/network/api_client.dart`.

- Base URL: `http://10.0.2.2:8080` (Android emulator -> host localhost).
- **Request interceptor**: injects `Authorization: Bearer <token>` on every request.
- **Error interceptor**: on 401, attempts token refresh via `/api/v1/auth/refresh`.
  - Refresh succeeds: retries the original request transparently.
  - Refresh fails: clears tokens, triggers redirect to login via auth state change.
- Uses a separate Dio instance for the refresh call to avoid interceptor loops.
- Provides typed methods: `get`, `post`, `put`, `patch`, `delete`, `upload`.

### Backend API format (matches Go backend)
```json
// Success:  { "data": { ... }, "meta": { ... } }
// Error:    { "error": { "code": "...", "message": "..." } }
```

---

## Error Handling

### The AsyncValue Pattern
All async operations must use `AsyncValue` for consistent loading/error/data states.

```dart
// CORRECT — pattern matching on AsyncValue
ref.watch(missionsProvider).when(
  data: (missions) => MissionListView(missions: missions),
  loading: () => const MissionListSkeleton(),
  error: (error, stack) => ErrorDisplay(
    message: _userFriendlyMessage(error),
    onRetry: () => ref.invalidate(missionsProvider),
  ),
);

// WRONG — manual loading/error tracking
if (isLoading) return CircularProgressIndicator();
if (error != null) return Text(error.toString()); // Raw error shown to user
```

### Rules
- **User-friendly error messages** — never show raw exceptions, stack traces, or technical error strings to users.
- **Retry mechanisms for network failures** — every error state must offer a retry action.
- **Offline mode awareness** — use `connectivity_plus` to detect offline state and show appropriate UI. Cache critical data locally for read-only offline access.
- **Loading states are mandatory** — use `shimmer` package for skeleton screens. Skeletons always preferred over spinners.
- **Empty states with clear CTAs** — every list view must handle the empty case with guidance for the user.

---

## Performance Standards

### Targets
| Metric | Target |
|--------|--------|
| Animation/scrolling | 60fps minimum |
| App cold start | < 2 seconds |
| Screen transition | < 300ms |

### Widget Performance
- **`ListView.builder`** for all dynamic lists — never `ListView(children: [...])` for lists that could grow.
- **`const` widgets** for all static content — this skips rebuild entirely.
- **Avoid rebuilding entire widget trees** — scope `ref.watch` to the smallest possible widget. Use `ConsumerWidget` at the leaf, not the root.
- **`RepaintBoundary`** around expensive custom painters and animations.

### Image Performance
- **`CachedNetworkImage`** for all network images — never raw `Image.network()`.
- Always provide `placeholder` and `errorWidget` for cached images.
- Resize images on the server when possible — do not load full-resolution images for thumbnails.

### Lazy Loading
- **Deferred imports** for feature screens that are not on the initial route.
- **Pagination** for all list endpoints — never fetch all items at once.

---

## Secure Storage

Encrypted key-value store for sensitive data:
- Access token, refresh token (JWT pair).
- Cached user JSON (offline fallback).
- Android: EncryptedSharedPreferences.
- iOS: Keychain (first_unlock accessibility).

All token access goes through `SecureStorageService` — never read/write tokens directly.

---

## Theme — Direction Soleil v2

The mobile app ships under the **Soleil v2** visual direction, shared with web and admin (ivoire & corail palette, Fraunces serif for display, Inter Tight sans for UI, Geist Mono for numbers). **Source of truth**: [`/design/INDEX.md`](../design/INDEX.md). Read [`/design/rules.md`](../design/rules.md) before any UI change.

### Tokens (Material 3 + custom `SoleilColors` extension)

Full table in [`/design/DESIGN_SYSTEM.md`](../design/DESIGN_SYSTEM.md). Quick reference:

| Semantic | Soleil hex | Material 3 mapping |
|----------|-----------|---------------------|
| Background (ivoire) | `#fffbf5` | `colorScheme.surface` |
| Card (white) | `#ffffff` | `colorScheme.surfaceContainerLowest` |
| Encre (text) | `#2a1f15` | `colorScheme.onSurface` |
| Tabac (mute text) | `#7a6850` | `colorScheme.onSurfaceVariant` |
| Border | `#f0e6d8` | `colorScheme.outline` |
| Corail (CTA) | `#e85d4a` | `colorScheme.primary` |
| Sapin (success) | `#5a9670` | `SoleilColors.success` |
| Corail foncé | `#c43a26` | `colorScheme.error` |

**Type**:
- `SoleilTextStyles.display` — Fraunces, 38-44px, weight 400-500, italic for accent words
- `SoleilTextStyles.body` — Inter Tight, 14-16px
- `SoleilTextStyles.mono` — Geist Mono, used for amounts, IDs, dates metadata

**Custom theme extension** `SoleilColors` for accentSoft, pinkSoft, greenSoft, sapinSoft (all the "soft" pastels used in pills/badges).

Access: `Theme.of(context).extension<SoleilColors>()!`

### Hard rules

- Colors only via `colorScheme` or `SoleilColors` extension — **no `Color(0xFF...)` hardcoded** in widgets.
- Typography only via `SoleilTextStyles` — **no inline `TextStyle(fontSize: ...)`** with magic numbers.
- Photos = `Portrait(id: n)` widget — never initials, never asset fallbacks.
- French strings via `AppLocalizations.of(context)` (i18n) — never hardcoded in widgets.
- `const` constructors wherever possible (perf budget: 60fps minimum).

---

## Testing Strategy

### Coverage Requirements

| Layer | What to Test | How |
|-------|-------------|-----|
| Domain (use cases) | Business logic, edge cases | Unit tests with mocked repositories |
| Domain (entities) | Freezed equality, serialization | Unit tests |
| Data (repositories) | API calls, DTO mapping, error handling | Unit tests with mocked Dio/ApiClient |
| Presentation (providers) | State transitions, async flows | Unit tests with mocked use cases |
| Presentation (screens) | Rendering, user interactions | Widget tests |
| Critical flows | Login, register, mission apply | Integration tests |
| Key UI components | Visual regression | Golden tests |

### Rules
- **Mock all external dependencies** with Mockito — never hit real APIs in tests.
- **Test the `when` branches** — every `AsyncValue` must be tested in loading, error, and data states.
- **Golden tests** for key screens — capture pixel-perfect snapshots and diff against baseline.
- **Integration tests** for the flows that would cost money if broken: login, registration, mission application.

```bash
flutter test                    # Run all tests
flutter test test/unit/         # Unit tests only
flutter test test/widget/       # Widget tests only
flutter test --update-goldens   # Update golden files
```

---

## Code Generation

Freezed for immutable entities, json_serializable for JSON (de)serialization:

```bash
dart run build_runner build --delete-conflicting-outputs    # One-shot generation
dart run build_runner watch --delete-conflicting-outputs    # Watch mode
```

Generated files: `*.freezed.dart`, `*.g.dart` — excluded from analysis via `analysis_options.yaml`, **committed alongside source files** (convention since P12, 2026-05-02). Run `build_runner` locally **before each commit** that touches a Freezed/json_serializable-annotated file so the generated artefacts stay in sync.

---

## File Naming

Dart convention: **snake_case** for all files and directories.

| Category | Convention | Example |
|----------|-----------|---------|
| Entities | Feature noun | `user.dart`, `mission.dart` |
| Repositories (interface) | `*_repository.dart` | `auth_repository.dart` |
| Repositories (impl) | `*_repository_impl.dart` | `auth_repository_impl.dart` |
| Use cases | Verb phrase | `login_usecase.dart`, `get_missions_usecase.dart` |
| Providers | `*_provider.dart` | `auth_provider.dart`, `mission_provider.dart` |
| Screens | `*_screen.dart` | `login_screen.dart`, `dashboard_screen.dart` |
| Widgets | Descriptive noun | `role_selector.dart`, `mission_card.dart` |
| DTOs | `*_request.dart` / `*_response.dart` | `login_request.dart`, `user_response.dart` |

---

## Key Rules Summary

- **Standalone app** — no shared packages with web/ or admin/.
- **All API communication via `ApiClient`** — never raw Dio or http calls.
- **Tokens stored exclusively in `SecureStorageService`** — no SharedPreferences for auth data.
- **English-language UI strings** — all user-facing text in English.
- **Each feature is self-contained** — never import from one feature into another. Share via `core/` or domain events.
- **Generated files are committed** — run `dart run build_runner build --delete-conflicting-outputs` locally before each commit that changes a Freezed/json_serializable-annotated source so the `*.freezed.dart` and `*.g.dart` artefacts stay in sync (convention since P12, 2026-05-02).

---

## Commands

```bash
flutter run                                                # Run on connected device
flutter run --dart-define=API_URL=http://192.168.1.X:8080  # Custom API URL
dart run build_runner build --delete-conflicting-outputs    # Generate code (freezed, json)
dart run build_runner watch --delete-conflicting-outputs    # Watch mode for code gen
flutter test                                               # Run all tests
flutter analyze                                            # Static analysis
flutter build apk --release                                # Build Android APK
flutter build ios --release                                # Build iOS (requires macOS)
```

---

## Adding a New Feature

1. Create the feature directory: `lib/features/<name>/`.
2. Create the three layers:
   - `data/` — repository implementation, DTOs with `@JsonSerializable`.
   - `domain/entities/` — Freezed data classes.
   - `domain/repositories/` — abstract repository interface.
   - `domain/usecases/` — business logic that orchestrates repository calls.
   - `presentation/providers/` — Riverpod providers wrapping use cases.
   - `presentation/screens/` — full-page widgets.
   - `presentation/widgets/` — reusable UI components for this feature.
3. Register providers (repository, use case, notifier) in the feature's provider file.
4. Add routes in `core/router/app_router.dart`.
5. Run `dart run build_runner build --delete-conflicting-outputs` to generate Freezed/JSON code.
6. Never import from other features — if shared logic is needed, extract to `core/`.
