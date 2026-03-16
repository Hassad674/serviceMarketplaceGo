package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/pkg/crypto"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5434/marketplace?sslmode=disable"
	}

	db, err := postgres.NewConnection(databaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	hasher := crypto.NewBcryptHasher()
	userRepo := postgres.NewUserRepository(db)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if admin already exists
	exists, err := userRepo.ExistsByEmail(ctx, "admin@marketplace.local")
	if err != nil {
		log.Fatalf("failed to check admin existence: %v", err)
	}
	if exists {
		fmt.Println("admin user already exists, skipping seed")
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
