// Seed-token inserts one staff token into auth_tokens and prints the raw token for use in the UI.
// Run after migrations. Requires DATABASE_URL and optionally AUTH_SECRET (default: dev-secret-change-in-production).
//
//   go run ./cmd/seed-token
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourusername/project/internal/auth"
)

const defaultSecret = "dev-secret-change-in-production"
const defaultRawToken = "local-staff-token"

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("POSTGRES_DSN")
	}
	if dsn == "" {
		log.Fatal("DATABASE_URL or POSTGRES_DSN must be set")
	}
	secret := os.Getenv("AUTH_SECRET")
	if secret == "" {
		secret = os.Getenv("HMAC_SECRET")
	}
	if secret == "" {
		secret = defaultSecret
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	hash := auth.HashToken([]byte(secret), defaultRawToken)
	_, err = pool.Exec(ctx,
		`INSERT INTO auth_tokens (token_hash, token_type, is_active) VALUES ($1, 'staff', true)`,
		hash,
	)
	if err != nil {
		log.Fatalf("insert token: %v", err)
	}

	fmt.Println("Staff token created. Use in UI (Settings → Bearer token):")
	fmt.Println(defaultRawToken)
}
