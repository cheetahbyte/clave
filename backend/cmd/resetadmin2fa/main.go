package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: resetadmin2fa <email>\n")
		os.Exit(1)
	}

	email := os.Args[1]

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://clave@localhost:54321/clave?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	q := db.New(pool)
	admin, err := q.GetAdminByEmail(context.Background(), email)
	if err != nil {
		fmt.Fprintf(os.Stderr, "admin not found: %s\n", email)
		os.Exit(1)
	}

	if err := q.DisableTOTP(context.Background(), admin.ID); err != nil {
		fmt.Fprintf(os.Stderr, "failed to disable TOTP: %v\n", err)
		os.Exit(1)
	}

	if err := q.InvalidateRecoveryCodes(context.Background(), admin.ID); err != nil {
		fmt.Fprintf(os.Stderr, "failed to invalidate recovery codes: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("2FA reset for %s (id=%s)\n", admin.Email, admin.ID)
}
