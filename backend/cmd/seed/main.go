package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/adapter/postgres"
	domainskill "marketplace-backend/internal/domain/skill"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/seed"
	"marketplace-backend/pkg/crypto"
)

// devFallbackDatabaseURL mirrors cmd/migrate/main.go — public dev
// fallback used by docker-compose, fail-fast guarded for prod.
//
// #nosec G101 -- public dev fallback, fail-fast guarded for prod
const devFallbackDatabaseURL = "postgres://postgres:postgres@localhost:5434/marketplace?sslmode=disable"

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		// SEC: never seed the public dev defaults into a prod DB.
		if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
			log.Fatal("seed: DATABASE_URL is required in production — refusing to boot with the public dev fallback")
		}
		databaseURL = devFallbackDatabaseURL
	}

	db, err := postgres.NewConnection(databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	hasher := crypto.NewBcryptHasher()
	userRepo := postgres.NewUserRepository(db)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	seedAdmin(ctx, userRepo, hasher)
	seedCuratedSkills(ctx, db)
}

func seedAdmin(ctx context.Context, userRepo *postgres.UserRepository, hasher *crypto.BcryptHasher) {
	exists, err := userRepo.ExistsByEmail(ctx, "admin@marketplace.local")
	if err != nil {
		log.Fatalf("failed to check admin existence: %v", err)
	}
	if exists {
		fmt.Println("admin user already exists, skipping admin seed")
		return
	}

	hashedPassword, err := hasher.Hash("Admin123!")
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	admin := &user.User{
		ID:             uuid.New(),
		Email:          "admin@marketplace.local",
		HashedPassword: hashedPassword,
		FirstName:      "Admin",
		LastName:       "User",
		DisplayName:    "Admin",
		Role:           user.RoleProvider,
		IsAdmin:        true,
		EmailVerified:  true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := userRepo.Create(ctx, admin); err != nil {
		log.Fatalf("failed to create admin user: %v", err)
	}

	fmt.Println("admin user created successfully")
	fmt.Printf("  email: %s\n", admin.Email)
	fmt.Printf("  password: Admin123!\n")
}

// seedCuratedSkills upserts every entry from internal/seed.CuratedSkills
// into skills_catalog with is_curated = true. Idempotent: re-running the
// seed refreshes display_text and expertise_keys without losing usage_count
// (the postgres adapter's Upsert preserves it by design).
func seedCuratedSkills(ctx context.Context, db *sql.DB) {
	catalogRepo := postgres.NewSkillCatalogRepository(db)

	inserted := 0
	for _, s := range seed.CuratedSkills {
		entry, err := domainskill.NewCatalogEntry(s.DisplayText, s.DisplayText, s.ExpertiseKeys, true)
		if err != nil {
			log.Printf("skipping invalid seed skill %q: %v", s.DisplayText, err)
			continue
		}
		if err := catalogRepo.Upsert(ctx, entry); err != nil {
			log.Fatalf("failed to upsert seed skill %q: %v", s.DisplayText, err)
		}
		inserted++
	}
	fmt.Printf("seeded %d curated skills into skills_catalog\n", inserted)
}
