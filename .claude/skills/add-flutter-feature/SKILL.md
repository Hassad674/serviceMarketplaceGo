---
name: add-flutter-feature
description: Scaffold a complete new Flutter feature with Clean Architecture layers — Freezed entities, abstract repository, Dio-based implementation, Riverpod providers, screen, and widgets.
user-invocable: true
allowed-tools: Read, Write, Edit, Bash, Grep, Glob, Agent
---

# Add Flutter Feature

Create the Flutter feature: **$ARGUMENTS**

You are scaffolding a new feature for the Flutter mobile app. Follow EVERY step below in order. Domain layer first, presentation last. Read existing code for patterns before writing anything.

---

## STEP 0 — Understand the request

Parse `$ARGUMENTS` to determine:
- **Feature name** (snake_case for files/directories, PascalCase for classes)
- **Core entity** and its fields (types, required vs optional, defaults)
- **Repository operations** needed (CRUD or custom)
- **Screen type** — list, detail, form, or combination

If the request is ambiguous, ask the user to clarify before proceeding.

---

## STEP 1 — Domain: Entity

Create `mobile/lib/features/{feature}/domain/entities/{entity}.dart`:

**Pattern** (reference `mobile/lib/features/auth/domain/entities/user.dart`):
- Freezed data class with `@freezed` annotation
- All fields use named parameters
- Include `part` directives for generated files
- Add `fromJson` factory for JSON deserialization
- Use Dart types: `String`, `int`, `double`, `bool`, `DateTime`
- Enums defined in the same file if feature-specific

```dart
import 'package:freezed_annotation/freezed_annotation.dart';

part '{entity}.freezed.dart';
part '{entity}.g.dart';

@freezed
class {Entity} with _${Entity} {
  const factory {Entity}({
    required String id,
    // ... entity fields
    required DateTime createdAt,
    required DateTime updatedAt,
  }) = _{Entity};

  factory {Entity}.fromJson(Map<String, dynamic> json) => _${Entity}FromJson(json);
}
```

**Naming rules:**
- Dart fields use camelCase (`firstName`), JSON uses snake_case (`first_name`)
- Freezed + json_serializable handles the conversion automatically with `@JsonKey` if needed
- Use `@Default(value)` for fields with defaults

---

## STEP 2 — Domain: Repository interface

Create `mobile/lib/features/{feature}/domain/repositories/{feature}_repository.dart`:

**Pattern** (reference `mobile/lib/features/auth/domain/repositories/auth_repository.dart`):
- Abstract class — NO implementation details
- Methods return domain entities, never DTOs or JSON maps
- All methods return `Future<>` for async operations
- Import ONLY from domain layer (entities)

```dart
import '../entities/{entity}.dart';

abstract class {Feature}Repository {
  Future<List<{Entity}>> getAll();
  Future<{Entity}> getById(String id);
  Future<{Entity}> create({/* named params matching entity fields */});
  Future<{Entity}> update(String id, {/* optional named params */});
  Future<void> delete(String id);
}
```

Only include methods that are actually needed for the feature.

---

## STEP 3 — Domain: Use cases (if needed)

Create use cases in `mobile/lib/features/{feature}/domain/usecases/` only if there is business logic beyond simple CRUD. Skip this step for straightforward data features.

**Pattern** (reference `mobile/lib/features/auth/domain/usecases/login_usecase.dart`):
```dart
import '../repositories/{feature}_repository.dart';
import '../entities/{entity}.dart';

class {Action}{Entity}UseCase {
  final {Feature}Repository _repository;

  {Action}{Entity}UseCase(this._repository);

  Future<{Entity}> call(/* params */) async {
    // Business logic / validation
    return _repository.someMethod(/* params */);
  }
}
```

---

## STEP 4 — Data: Repository implementation

Create `mobile/lib/features/{feature}/data/{feature}_repository_impl.dart`:

**Pattern** (reference `mobile/lib/features/auth/data/auth_repository_impl.dart`):
- Implements the abstract repository from domain layer
- Uses `ApiClient` from core/network for HTTP calls
- Parses JSON responses into domain entities
- Handles API response format: `{ "data": { ... } }` for single, `{ "data": [...], "meta": {...} }` for lists

```dart
import 'package:marketplace_mobile/core/network/api_client.dart';
import '../domain/entities/{entity}.dart';
import '../domain/repositories/{feature}_repository.dart';

class {Feature}RepositoryImpl implements {Feature}Repository {
  final ApiClient _apiClient;

  {Feature}RepositoryImpl({required ApiClient apiClient}) : _apiClient = apiClient;

  @override
  Future<List<{Entity}>> getAll() async {
    final response = await _apiClient.get('/api/v1/{feature}s');
    final items = (response.data['data'] as List)
        .map((json) => {Entity}.fromJson(json as Map<String, dynamic>))
        .toList();
    return items;
  }

  @override
  Future<{Entity}> getById(String id) async {
    final response = await _apiClient.get('/api/v1/{feature}s/$id');
    return {Entity}.fromJson(response.data['data'] as Map<String, dynamic>);
  }

  @override
  Future<{Entity}> create({/* params */}) async {
    final response = await _apiClient.post('/api/v1/{feature}s', data: {
      // Map camelCase params to snake_case API fields
    });
    return {Entity}.fromJson(response.data['data'] as Map<String, dynamic>);
  }

  @override
  Future<void> delete(String id) async {
    await _apiClient.delete('/api/v1/{feature}s/$id');
  }
}
```

---

## STEP 5 — Presentation: Riverpod providers

Create `mobile/lib/features/{feature}/presentation/providers/{feature}_provider.dart`:

**Pattern** (reference `mobile/lib/features/auth/presentation/providers/auth_provider.dart`):
- Repository provider for dependency injection
- State class for the feature's UI state
- StateNotifier or AsyncNotifier for state management
- Use `AsyncValue` for async operations

```dart
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../../core/network/api_client.dart';
import '../../data/{feature}_repository_impl.dart';
import '../../domain/entities/{entity}.dart';
import '../../domain/repositories/{feature}_repository.dart';

// Repository provider (DI)
final {feature}RepositoryProvider = Provider<{Feature}Repository>((ref) {
  final apiClient = ref.watch(apiClientProvider);
  return {Feature}RepositoryImpl(apiClient: apiClient);
});

// List provider
final {feature}sProvider = FutureProvider<List<{Entity}>>((ref) async {
  final repository = ref.watch({feature}RepositoryProvider);
  return repository.getAll();
});

// Single item provider
final {feature}Provider = FutureProvider.family<{Entity}, String>((ref, id) async {
  final repository = ref.watch({feature}RepositoryProvider);
  return repository.getById(id);
});
```

For features with complex state (forms, multi-step flows), use `StateNotifierProvider` instead of `FutureProvider`. Follow the auth provider pattern.

---

## STEP 6 — Presentation: Main screen

Create `mobile/lib/features/{feature}/presentation/screens/{feature}_screen.dart`:

**Pattern:**
- `ConsumerWidget` or `ConsumerStatefulWidget` for Riverpod integration
- Handle `AsyncValue` states: loading, error, data
- Use `ref.watch()` for reactive state, `ref.read()` for one-off actions
- Use `const` constructors where possible

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/{feature}_provider.dart';
import '../widgets/{feature}_card.dart';

class {Feature}Screen extends ConsumerWidget {
  const {Feature}Screen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final {feature}sAsync = ref.watch({feature}sProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('{Feature title}')),
      body: {feature}sAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, stack) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('Erreur: ${error.toString()}'),
              const SizedBox(height: 8),
              ElevatedButton(
                onPressed: () => ref.invalidate({feature}sProvider),
                child: const Text('Reessayer'),
              ),
            ],
          ),
        ),
        data: (items) {
          if (items.isEmpty) {
            return const Center(child: Text('Aucun element'));
          }
          return ListView.builder(
            itemCount: items.length,
            itemBuilder: (context, index) => {Feature}Card(item: items[index]),
          );
        },
      ),
    );
  }
}
```

---

## STEP 7 — Presentation: Widgets

Create reusable widgets in `mobile/lib/features/{feature}/presentation/widgets/`:

Typical set (adjust to the feature):
- `{feature}_card.dart` — Card widget for list items
- `{feature}_form.dart` — Form widget for create/edit (if needed)

**Widget rules:**
- Use `const` constructors
- Named parameters for constructors with more than 2 parameters
- Use `Theme.of(context)` for colors and text styles
- Use `Theme.of(context).extension<AppColors>()!` for custom colors
- Lists must use `ListView.builder` (never `ListView` with `children` list)
- Images must use `CachedNetworkImage` (never raw `Image.network`)
- No `setState` — use Riverpod providers
- No `print()` — use a logger if debugging is needed
- Keep `build()` methods under 100 lines

---

## STEP 8 — Run code generation

After creating Freezed entities, run:

```bash
cd /home/hassad/Documents/marketplaceServiceGo/mobile && dart run build_runner build --delete-conflicting-outputs
```

Verify:
- `{entity}.freezed.dart` was generated
- `{entity}.g.dart` was generated (for JSON serialization)
- No build errors

---

## STEP 9 — Wire into router (if screen is a top-level route)

If this feature has its own screen accessible from navigation, update `mobile/lib/core/router/app_router.dart`:
- Add a route path constant to `RoutePaths`
- Add a `GoRoute` for the feature screen
- Place it in the correct location (inside `ShellRoute` for authenticated screens)

---

## STEP 10 — Verify feature isolation

Run these checks before reporting done:

1. **Clean Architecture layers:**
   - `domain/` imports ONLY other domain files (entities, repositories) and pure Dart — never `package:flutter`, never `data/`, never `presentation/`
   - `data/` imports from `domain/` (implements interfaces) and `core/` (ApiClient) — never from `presentation/`
   - `presentation/` imports from `domain/` (entities) and `core/` — never directly from `data/` except through providers

2. **Feature isolation:**
   - No file in this feature imports from another feature's directory
   - Only `core/` and this feature's own files are imported
   - No shared mutable state with other features

3. **No forbidden patterns:**
   - No `setState` in any widget (use Riverpod)
   - No `print()` statements
   - No raw `Image.network` (use CachedNetworkImage)

If any check fails, fix it before finishing.

---

## Output

When finished, report:
1. Files created (grouped by layer: domain, data, presentation)
2. Generated files (freezed, g.dart)
3. Files modified (router, if applicable)
4. Architecture verification result
5. Any decisions made and why
