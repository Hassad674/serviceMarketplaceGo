---
name: generate
description: Run Flutter code generation (build_runner) to regenerate Freezed entities and json_serializable DTOs. Use after any change to @freezed classes or @JsonSerializable DTOs.
user-invocable: true
allowed-tools: Bash, Glob
---

# Generate — Flutter Code Generation

Target: **$ARGUMENTS**

Run `build_runner` to regenerate all `*.freezed.dart` and `*.g.dart` files in the Flutter mobile app.

---

## STEP 1 — Run build_runner

```bash
cd /home/hassad/Documents/marketplaceServiceGo/mobile && dart run build_runner build --delete-conflicting-outputs
```

If the command fails:
- Read the error output carefully
- Common issues: missing `part` directives, syntax errors in Freezed classes, version conflicts
- Fix the issue and re-run

---

## STEP 2 — Verify generated files

List all generated files to confirm they were created:

```bash
find /home/hassad/Documents/marketplaceServiceGo/mobile/lib -name "*.freezed.dart" -o -name "*.g.dart" | sort
```

For each `@freezed` class in the codebase, verify:
- A matching `.freezed.dart` file exists
- A matching `.g.dart` file exists (if the class has `fromJson`)

---

## STEP 3 — Check for errors

Run static analysis to ensure generated code does not introduce issues:

```bash
cd /home/hassad/Documents/marketplaceServiceGo/mobile && flutter analyze
```

If analysis reports errors in generated files, they are usually caused by the source Freezed class. Fix the source and re-run generation.

---

## Output

Report:
1. Build result (success or failure with details)
2. List of generated files
3. Analysis result (clean or warnings/errors)
