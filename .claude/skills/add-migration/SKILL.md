---
name: add-migration
description: Create a numbered up/down SQL migration pair following project conventions. Use when adding or modifying database tables, columns, indexes, or constraints.
user-invocable: true
allowed-tools: Read, Write, Bash, Glob, Grep
---

# Add Migration

Create migration for: **$ARGUMENTS**

You are creating a SQL migration for the marketplace backend. Follow every rule below precisely.

---

## STEP 1 — Determine the next migration number

Check existing migration files:
```bash
ls /home/hassad/Documents/marketplaceServiceGo/backend/migrations/*.up.sql 2>/dev/null | sort
```

If no migrations exist yet, start at `001`. Otherwise, increment the highest number by 1. Pad to 3 digits: `001`, `002`, ..., `010`, ..., `100`.

---

## STEP 2 — Determine the migration name

Parse `$ARGUMENTS` to derive a descriptive snake_case name.

Examples:
- "create missions table" -> `002_create_missions`
- "add bio to profiles" -> `003_add_bio_to_profiles`
- "create contracts table" -> `004_create_contracts`
- "add index on status" -> `005_add_index_on_missions_status`

---

## STEP 3 — Create the UP migration

Create `backend/migrations/{NNN}_{name}.up.sql`

### SQL conventions (mandatory, reference `backend/migrations/001_create_users.up.sql`):

**Primary keys:**
```sql
id UUID PRIMARY KEY DEFAULT gen_random_uuid()
```

**Timestamps (every table must have these):**
```sql
created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
```

**String columns — use TEXT, not VARCHAR:**
```sql
title TEXT NOT NULL,
description TEXT NOT NULL DEFAULT ''
```
Exception: use VARCHAR only when the existing codebase already does (e.g., the users table uses VARCHAR — match existing style within the same table if altering it).

**Enum columns — use CHECK constraints:**
```sql
status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'closed', 'cancelled'))
```

**Foreign keys — ONLY to users table:**
```sql
user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
```

For referencing other feature tables, store the ID without a FK constraint:
```sql
mission_id UUID NOT NULL,  -- no FK: missions is an independent feature
```
Add a comment explaining why there is no FK.

**Boolean columns:**
```sql
is_active BOOLEAN NOT NULL DEFAULT false
```

**JSONB for flexible data:**
```sql
metadata JSONB NOT NULL DEFAULT '{}'::jsonb
```

**Nullable columns — use sparingly:**
```sql
completed_at TIMESTAMPTZ  -- NULL = not yet completed
```

**Indexes — always index foreign keys and frequently queried columns:**
```sql
CREATE INDEX idx_{table}_{column} ON {table}({column});
```

**Partial indexes — use when most queries filter on a condition:**
```sql
CREATE INDEX idx_{table}_{column}_active ON {table}({column}) WHERE status = 'active';
CREATE INDEX idx_{table}_{column}_notnull ON {table}({column}) WHERE {column} IS NOT NULL;
```

**Unique indexes:**
```sql
CREATE UNIQUE INDEX idx_{table}_{col1}_{col2}_unique ON {table}({col1}, {col2});
```

**Updated_at trigger — reuse the existing function from migration 001:**
```sql
CREATE TRIGGER {table}_updated_at
    BEFORE UPDATE ON {table}
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
```
Do NOT redefine `update_updated_at()` — it already exists from `001_create_users.up.sql`.

### Full table creation template:

```sql
CREATE TABLE {feature_table} (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- feature-specific columns
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_{feature_table}_user_id ON {feature_table}(user_id);
CREATE INDEX idx_{feature_table}_status ON {feature_table}(status);

CREATE TRIGGER {feature_table}_updated_at
    BEFORE UPDATE ON {feature_table}
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();
```

### Column modification template:

```sql
ALTER TABLE {table} ADD COLUMN {column} TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_{table}_{column} ON {table}({column});
```

---

## STEP 4 — Create the DOWN migration

Create `backend/migrations/{NNN}_{name}.down.sql`

The down migration must **perfectly reverse** the up migration, in **reverse order**:

- `CREATE TRIGGER` -> `DROP TRIGGER IF EXISTS {trigger} ON {table};`
- `CREATE INDEX` -> `DROP INDEX IF EXISTS {index_name};`
- `CREATE TABLE` -> `DROP TABLE IF EXISTS {table};`
- `ALTER TABLE ADD COLUMN` -> `ALTER TABLE {table} DROP COLUMN IF EXISTS {column};`
- `ALTER TABLE ADD CONSTRAINT` -> `ALTER TABLE {table} DROP CONSTRAINT IF EXISTS {constraint};`

Always use `IF EXISTS` in down migrations to make them idempotent.

**Important:** Do NOT drop `update_updated_at()` function in any migration except the one that created it (001). Other migrations only create triggers that reference it.

### Full reversal template:

```sql
DROP TRIGGER IF EXISTS {feature_table}_updated_at ON {feature_table};
DROP INDEX IF EXISTS idx_{feature_table}_status;
DROP INDEX IF EXISTS idx_{feature_table}_user_id;
DROP TABLE IF EXISTS {feature_table};
```

---

## STEP 5 — Verify the cross-feature FK rule

Read the up migration and check EVERY `REFERENCES` clause:
- `REFERENCES users(id)` -> OK (users is core, always allowed)
- `REFERENCES {any_other_table}` -> FORBIDDEN

If a cross-feature FK is detected, restructure:
1. Remove the `REFERENCES` clause
2. Keep the column as a plain UUID
3. Add a comment: `-- no FK: {other_feature} is an independent feature`
4. Index the column anyway for query performance

---

## STEP 6 — Feature table naming verification

Verify the table name follows conventions:

| Feature | Expected tables |
|---------|----------------|
| user | `users` |
| profile | `agency_profiles`, `enterprise_profiles`, `provider_profiles` |
| mission | `missions`, `mission_applications` |
| contract | `contracts` |
| message | `conversations`, `messages`, `conversation_participants` |
| review | `reviews` |
| notification | `notifications` |
| payment | `payments`, `invoices` |

New features should use a clear, descriptive plural table name that does not conflict with existing features.

---

## STEP 7 — Validate SQL syntax

Read the generated SQL and verify:
- No typos in SQL keywords
- Matching parentheses
- Correct PostgreSQL syntax (TIMESTAMPTZ not DATETIME, TEXT not VARCHAR for new tables, gen_random_uuid() not UUID_GENERATE_V4())
- Proper comma separation between columns (no trailing comma before closing parenthesis)
- CHECK constraint values match the domain value object constants (if they exist)
- Index names are unique and follow `idx_{table}_{column}` pattern

---

## STEP 8 — Test instructions

Print the test commands for the user:
```bash
cd /home/hassad/Documents/marketplaceServiceGo/backend
make migrate-up      # Apply the new migration
make migrate-down    # Rollback to verify down works
make migrate-up      # Re-apply to confirm idempotency
```

---

## Output

Report:
1. **Files created** — `backend/migrations/{NNN}_{name}.up.sql` and `.down.sql`
2. **Tables/columns affected** — what was created or modified
3. **Indexes created** — list of indexes
4. **Triggers created** — list of triggers
5. **FK verification** — pass/fail (list any cross-feature FKs found)
6. **Warnings** — anything the user should be aware of

Example:
```
Created:
  backend/migrations/003_create_missions.up.sql
  backend/migrations/003_create_missions.down.sql

Tables: missions
Columns: id, user_id, title, description, budget_min, budget_max, status, location, remote_ok, created_at, updated_at
Indexes: idx_missions_user_id, idx_missions_status
Triggers: missions_updated_at (reuses update_updated_at function)
FK check: PASS (only references users table)

Test: run `make migrate-up && make migrate-down && make migrate-up`
```
