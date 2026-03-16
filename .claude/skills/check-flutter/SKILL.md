---
name: check-flutter
description: Verify Flutter app architecture compliance — Clean Architecture layers, feature isolation, code quality, performance, and state management patterns. Run to detect violations before they accumulate.
user-invocable: true
allowed-tools: Read, Bash, Grep, Glob, Agent
---

# Check Flutter — Architecture & Quality Verification

Target: **$ARGUMENTS**

If `$ARGUMENTS` is empty, check the ENTIRE Flutter app. Otherwise, check only the specified feature(s).

You are the architecture guardian for the Flutter mobile app. Run every check below and produce a clear pass/fail report.

---

## CHECK 1 — Clean Architecture layers

### 1a. Domain purity

Verify that files in `mobile/lib/features/*/domain/` import ONLY:
- Other domain files within the same feature (`../entities/`, `../repositories/`, `../usecases/`)
- Pure Dart packages (`dart:core`, `dart:async`, `dart:collection`, etc.)
- `package:freezed_annotation` and `package:json_annotation` (for code generation)

**How to check:**
Use Grep to search for import statements in all `domain/` directories. Verify none reference:
- `package:flutter` (domain must be framework-independent)
- `data/` directories
- `presentation/` directories
- `core/network/` or `core/storage/`
- `package:dio`
- `package:flutter_riverpod`

**FAIL if:** Any domain file imports from data/, presentation/, core/network/, or Flutter packages.

### 1b. Data layer dependencies

Verify that files in `mobile/lib/features/*/data/` import ONLY:
- Domain files from the same feature (`../domain/entities/`, `../domain/repositories/`)
- Core utilities (`core/network/`, `core/storage/`)
- External packages used for data operations (`package:dio`)

**FAIL if:** Any data file imports from `presentation/` or from another feature's directories.

### 1c. Presentation layer dependencies

Verify that files in `mobile/lib/features/*/presentation/` import from:
- Domain entities and repositories from the same feature
- Core utilities (`core/theme/`, `core/router/`, `core/utils/`)
- Data layer ONLY through the repository provider (the provider file wires data -> domain)
- Flutter and Riverpod packages

**FAIL if:** Presentation files import directly from `data/` (except in the provider file that wires DI). Screens and widgets must never import `*_repository_impl.dart` directly.

---

## CHECK 2 — Feature isolation

### 2a. No cross-feature imports

Verify that no file in `mobile/lib/features/{X}/` imports from `mobile/lib/features/{Y}/`.

**How to check:**
Use Grep to search all files in `mobile/lib/features/` for import statements containing `features/`. For each match, extract the feature name from the import path and compare it to the feature name of the importing file.

**FAIL if:** `features/mission/presentation/screens/mission_screen.dart` imports from `features/auth/domain/entities/user.dart`

### 2b. Features only import from allowed sources

Each feature file can ONLY import from:
- Its own feature directory
- `core/` packages (theme, network, storage, router, utils)
- Flutter SDK and external pub packages
- `dart:*` standard libraries

**FAIL if:** A feature imports from another feature, from `generated/`, or from `main.dart`.

### 2c. No shared state between features

Each feature must have its own providers. Verify that no provider in one feature's `presentation/providers/` references a provider from another feature.

**FAIL if:** `mission_provider.dart` does `ref.watch(authProvider)` where `authProvider` is from `features/auth/`. The exception is auth state, which may be accessed via a core-level provider if one exists.

---

## CHECK 3 — Code quality

### 3a. File length

No file should exceed 600 lines.

**How to check:**
```bash
find /home/hassad/Documents/marketplaceServiceGo/mobile/lib -name "*.dart" ! -name "*.freezed.dart" ! -name "*.g.dart" | xargs wc -l | sort -rn | head -20
```

Exclude generated files (`*.freezed.dart`, `*.g.dart`) from this check.

**FAIL if:** Any non-generated file exceeds 600 lines. **WARN if:** Over 400 lines.

### 3b. Build method length

For each widget class, check that the `build()` method does not exceed 100 lines.

**How to check:**
Search for `Widget build(` in all `.dart` files (excluding generated). For each match, count the lines until the matching closing brace.

**WARN if:** Any `build()` method exceeds 100 lines. Consider extracting sub-widgets.

### 3c. Const constructors

Verify that widget classes use `const` constructors where possible. A widget can have a `const` constructor if all its fields are final and their types support const.

**How to check:**
Search for `class * extends StatelessWidget` and `class * extends ConsumerWidget`. Check if the constructor is `const`.

**WARN if:** A stateless/consumer widget lacks a `const` constructor.

### 3d. All entities use Freezed

Verify that entity classes in `domain/entities/` use the `@freezed` annotation. No manual `==`, `hashCode`, `toString`, or `copyWith` implementations.

**FAIL if:** An entity class in `domain/entities/` does not use `@freezed`.

### 3e. No setState

Search all `.dart` files (excluding generated) for `setState(`. Riverpod should handle all state management.

**FAIL if:** `setState` is used anywhere outside of rare animation controllers or third-party widget wrappers.

### 3f. No print statements

Search all `.dart` files for `print(` calls. Use a logger instead.

**FAIL if:** Any `print()` call found in non-test, non-generated code.

---

## CHECK 4 — Performance

### 4a. ListView.builder for lists

Search for `ListView(` with a `children:` parameter (not `ListView.builder`). Large lists with `children:` create all items eagerly.

**FAIL if:** A `ListView` with `children:` is used for a list that could have more than ~10 items. `ListView.builder` should be used instead.

### 4b. CachedNetworkImage for remote images

Search for `Image.network(` in all `.dart` files. All network images should use `CachedNetworkImage` from the `cached_network_image` package.

**FAIL if:** `Image.network` is used anywhere. Use `CachedNetworkImage` instead.

### 4c. No unnecessary rebuilds

Check that providers are scoped correctly:
- `ref.watch()` should be used in `build()` methods for reactive rebuilds
- `ref.read()` should be used in callbacks and event handlers
- No `ref.watch()` calls inside callbacks (e.g., `onPressed: () { ref.watch(...) }`)

**How to check:**
Search for `ref.watch(` inside function bodies that are callbacks (inside `onPressed:`, `onTap:`, etc.).

**FAIL if:** `ref.watch()` is called inside a callback.

---

## CHECK 5 — State management

### 5a. Providers in correct location

All Riverpod providers must be declared in `presentation/providers/` directories. No providers declared in screens, widgets, or domain/data layers.

**How to check:**
Search for `Provider`, `StateNotifierProvider`, `FutureProvider`, `AsyncNotifierProvider`, `NotifierProvider` declarations outside of `presentation/providers/` directories (and outside `core/` for infrastructure providers like `apiClientProvider`).

**FAIL if:** A provider is declared in a screen, widget, or domain file.

### 5b. No raw StateProvider for complex state

`StateProvider` should only be used for simple primitive values (bool toggles, int counters). Complex state should use `StateNotifierProvider`, `NotifierProvider`, or `AsyncNotifierProvider`.

**How to check:**
Search for `StateProvider` declarations. Check if the type parameter is a complex type (class, List, Map with many fields).

**WARN if:** `StateProvider` holds a complex type. Use `NotifierProvider` instead.

### 5c. AsyncValue for async operations

All providers that fetch data should return `AsyncValue` (via `FutureProvider` or `AsyncNotifierProvider`). Screens and widgets must handle all three states: loading, error, data.

**How to check:**
Search for `FutureProvider` and `AsyncNotifierProvider` usage. For each, find where it is consumed (`ref.watch()`) and verify the result is handled with `.when()`, `.whenData()`, or explicit `AsyncValue` pattern matching.

**FAIL if:** An `AsyncValue` is accessed via `.value!` without handling loading/error states.

---

## Report format

Output a structured report:

```
# Flutter Architecture Check Report

## Summary
- Total checks: X
- Passed: X
- Failed: X
- Warnings: X

## Results

### CHECK 1 — Clean Architecture layers
- [PASS] 1a. Domain purity — all domain files import only dart:* and freezed_annotation
- [FAIL] 1b. Data layer — features/mission/data/mission_repository_impl.dart imports from presentation/providers/auth_provider.dart
  -> Fix: inject auth token via constructor parameter, not by importing the provider

### CHECK 2 — Feature isolation
- [PASS] 2a. No cross-feature imports
- [PASS] 2b. Allowed sources only
- [WARN] 2c. Shared state — mission_provider.dart watches authProvider from auth feature
  -> Fix: extract a core-level currentUserProvider or pass user ID as parameter

### CHECK 3 — Code quality
- [PASS] 3a. File length — longest file: auth_provider.dart at 247 lines
- [WARN] 3b. Build method — mission_screen.dart build() is 112 lines
  -> Fix: extract _buildMissionList() and _buildEmptyState() helper widgets
- [FAIL] 3e. setState — found in features/messaging/presentation/screens/chat_screen.dart:45
  -> Fix: replace with a Riverpod provider for message input state
...
```

For each failure, provide:
1. Exact file path and line number (if applicable)
2. What rule it violates
3. How to fix it
