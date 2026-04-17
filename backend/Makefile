.PHONY: run build test test-unit test-integration migrate-up migrate-down migrate-status seed mock lint tidy clean dev \
	reindex-bulk snapshot-typesense drift-check seed-search cover-backend fuzz-search perf-search

# Run API server with .env loaded
run:
	set -a && [ -f .env ] && . ./.env; set +a; go run cmd/api/main.go

# Build binary
build:
	go build -o bin/api cmd/api/main.go

# Run all tests
test:
	go test ./... -v -count=1

# Run unit tests only (skip integration)
test-unit:
	go test -short ./... -v -count=1

# Run integration tests only
test-integration:
	go test -run Integration ./... -v -count=1

# Migrations
migrate-up:
	set -a && [ -f .env ] && . ./.env; set +a; go run cmd/migrate/main.go up

migrate-down:
	set -a && [ -f .env ] && . ./.env; set +a; go run cmd/migrate/main.go down

migrate-status:
	set -a && [ -f .env ] && . ./.env; set +a; go run cmd/migrate/main.go status

# Seed initial data
seed:
	set -a && [ -f .env ] && . ./.env; set +a; go run cmd/seed/main.go

# Generate mocks from port interfaces
mock:
	@echo "Generate mocks from port interfaces"

# Lint
lint:
	go vet ./...

# Tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -rf bin/

# Dev alias
dev: run

# Search engine — bulk reindex every organization into Typesense.
# Idempotent: safe to re-run. Accepts the same --persona and --batch
# flags as the underlying CLI. Example:
#   make reindex-bulk
#   make reindex-bulk ARGS="--persona=freelance --batch=200"
reindex-bulk:
	set -a && [ -f .env ] && . ./.env; set +a; go run cmd/reindex/main.go $(ARGS)

# Search engine — trigger a Typesense snapshot and upload it to MinIO.
# Intended to run from a systemd timer / k8s CronJob. Idempotent: re-
# running on the same day overwrites the archive in place.
#   make snapshot-typesense
snapshot-typesense:
	set -a && [ -f .env ] && . ./.env; set +a; go run cmd/typesense-snapshot/main.go $(ARGS)

# Search engine — compare Postgres vs Typesense per-persona counts
# and log a WARN when drift exceeds 0.5%. Returns exit code 2 on
# alert so it can be piped into monitoring.
#   make drift-check
drift-check:
	set -a && [ -f .env ] && . ./.env; set +a; go run cmd/drift-check/main.go $(ARGS)

# Search engine — seed a large, realistic synthetic dataset (~500
# profiles by default) into the worktree database, then trigger a
# Typesense reindex so the engine reflects the new rows.
# Default seed=42 makes the output deterministic.
#   make seed-search
#   make seed-search ARGS="--count=200 --seed=7 --no-reindex"
seed-search:
	set -a && [ -f .env ] && . ./.env; set +a; go run ./cmd/seed-search $(ARGS)

# Coverage gate — runs the full test suite with coverage and asserts
# that each new search package holds above its threshold. Thresholds:
#   - internal/app/search              >= 85% (pure Go logic)
#   - internal/app/searchanalytics     >= 85% (pure Go logic)
#   - internal/app/searchindex         >= 80% (some ports hard to mock)
#   - internal/search                  >= 80% (HTTP paths are integration-tested)
# Exits non-zero when below threshold so CI blocks the merge.
#   make cover-backend
cover-backend:
	@set -e; \
	echo "running backend coverage"; \
	go test -cover ./internal/search/... ./internal/app/search/... \
		./internal/app/searchanalytics/... ./internal/app/searchindex/... \
		-count=1 2>&1 | tee /tmp/cover-backend.out; \
	fail=0; \
	while IFS= read -r line; do \
		pct=$$(echo "$$line" | grep -oE 'coverage: [0-9.]+%' | head -1 | awk '{print $$2}' | tr -d '%'); \
		pkg=$$(echo "$$line" | awk '{print $$2}'); \
		if [ -z "$$pct" ]; then continue; fi; \
		case "$$pkg" in \
			*internal/app/search|*internal/app/searchanalytics) threshold=85 ;; \
			*) threshold=80 ;; \
		esac; \
		awk -v p="$$pct" -v t="$$threshold" 'BEGIN{exit !(p+0 < t+0)}' && \
			{ echo "FAIL coverage $$pct% < $$threshold% on $$pkg"; fail=1; } || true; \
	done < /tmp/cover-backend.out; \
	[ "$$fail" = "0" ] || { echo "backend coverage gate failed" >&2; exit 1; }

# Run the filter-builder + cursor fuzz suites for 30 seconds each.
# Any panic / failure surfaces a minimised input in testdata/fuzz/.
#   make fuzz-search
fuzz-search:
	go test ./internal/app/search/ -run=NONE -fuzz=FuzzBuildFilterBy$$ -fuzztime=30s
	go test ./internal/app/search/ -run=NONE -fuzz=FuzzBuildFilterBy_Dedupe$$ -fuzztime=30s
	go test ./internal/app/search/ -run=NONE -fuzz=FuzzCursorRoundtrip$$ -fuzztime=30s
	go test ./internal/app/search/ -run=NONE -fuzz=FuzzCursorDecodeNoPanic$$ -fuzztime=30s
	go test ./internal/app/search/ -run=NONE -fuzz=FuzzCursorMonotonic$$ -fuzztime=15s
	go test ./internal/handler/ -run=NONE -fuzz=FuzzParseFilterInput$$ -fuzztime=30s

# Baseline load test against the local backend. Requires k6 installed
# (brew install k6 or https://grafana.com/docs/k6). Uses 50 VUs for
# 30s; p95 must stay under 200ms. DO NOT RUN AGAINST PROD.
#   make perf-search
perf-search:
	@command -v k6 >/dev/null 2>&1 || { echo "k6 not installed — see scripts/perf/k6-search.js header" >&2; exit 2; }
	k6 run ../scripts/perf/k6-search.js

