package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/alexedwards/argon2id"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: createadmin <email> <password> [role]\n")
		os.Exit(1)
	}

	email := os.Args[1]
	password := os.Args[2]
	role := "admin"
	if len(os.Args) > 3 {
		role = os.Args[3]
	}

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

	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to hash password: %v\n", err)
		os.Exit(1)
	}

	q := db.New(pool)

	admin, err := q.CreateAdmin(context.Background(), db.CreateAdminParams{
		Email:        email,
		PasswordHash: hash,
		Role:         role,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create admin: %v\n", err)
		os.Exit(1)
	}

	org, err := q.GetOrganizationBySlug(context.Background(), "default")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			org, err = q.CreateOrganization(context.Background(), db.CreateOrganizationParams{
				Name: "Default Organization",
				Slug: "default",
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create default organization: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "failed to find default organization: %v\n", err)
			os.Exit(1)
		}
	}

	if err := q.CreateOrganizationMember(context.Background(), db.CreateOrganizationMemberParams{
		OrganizationID: org.ID,
		AdminUserID:    admin.ID,
		Role:           role,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to add admin to organization: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Admin created: id=%s email=%s role=%s org=%s\n", admin.ID, admin.Email, admin.Role, org.Name)
}
