.PHONY: run build test test-unit test-integration migrate-up migrate-down migrate-status seed mock lint tidy clean dev

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

