package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// devFallbackDatabaseURL is the public localhost:5434 docker-compose
// connection string used by the local dev workflow. It is committed
// to the open-source repo on purpose — every contributor needs the
// same default to bootstrap a fresh checkout. The fail-fast guard in
// main() guarantees the fallback never silently survives into a prod
// deployment, mirroring the SEC-04 JWT_SECRET pattern from
// internal/config/config.go.
//
// #nosec G101 -- public dev fallback, fail-fast guarded for prod
const devFallbackDatabaseURL = "postgres://postgres:postgres@localhost:5434/marketplace?sslmode=disable"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: migrate <up|down|status>")
		os.Exit(1)
	}

	flag.Parse()
	command := os.Args[1]

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		// SEC: refuse to silently boot with the public dev fallback in
		// production. APP_ENV=production is the canonical signal and
		// matches what cmd/api/main.go uses via cfg.IsProduction().
		if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
			log.Fatal("migrate: DATABASE_URL is required in production — refusing to boot with the public dev fallback")
		}
		databaseURL = devFallbackDatabaseURL
	}

	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}
	defer m.Close()

	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("migration up failed: %v", err)
		}
		fmt.Println("migrations applied successfully")

	case "down":
		if err := m.Steps(-1); err != nil {
			log.Fatalf("migration down failed: %v", err)
		}
		fmt.Println("last migration reverted successfully")

	case "status":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("failed to get migration version: %v", err)
		}
		fmt.Printf("version: %d, dirty: %t\n", version, dirty)

	default:
		fmt.Printf("unknown command: %s\n", command)
		fmt.Println("usage: migrate <up|down|status>")
		os.Exit(1)
	}
}
