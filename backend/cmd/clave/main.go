package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/cheetahbyte/clave/internal/api"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/handlers"
	"github.com/cheetahbyte/clave/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func truthy(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "true" || v == "1" || v == "yes"
}

func verboseLoggingEnabled() bool {
	level := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	return truthy(os.Getenv("VERBOSE_LOGGING")) || level == "debug" || level == "verbose" || level == "trace"
}

func configureLogging(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))
}

func main() {
	verboseLogging := verboseLoggingEnabled()
	configureLogging(verboseLogging)

	databaseURL := getEnv("DATABASE_URL", "postgres://clave@localhost:54321/clave?sslmode=disable")

	if truthy(os.Getenv("RUN_MIGRATIONS")) {
		log.Println("running database migrations")
		migDb, err := sql.Open("pgx", databaseURL)
		if err != nil {
			log.Fatalf("failed to open migration connection: %v", err)
		}
		if err := goose.Up(migDb, "./migrations"); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
		migDb.Close()
		log.Println("migrations complete")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	q := db.New(pool)
	svc := services.InitServices(q, pool)
	h := handlers.New(svc)

	r := chi.NewRouter()
	api.Register(r, h, verboseLogging)

	port := getEnv("PORT", "8000")
	addr := "0.0.0.0:" + port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal("failed to start server")
	}
}
