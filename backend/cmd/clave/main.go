package main

import (
	"context"
	"log"
	"net/http"

	"github.com/cheetahbyte/clave/internal/api"
	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/handlers"
	"github.com/cheetahbyte/clave/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	r := chi.NewRouter()

	pool, err := pgxpool.New(context.Background(), "postgres://clave@localhost:54321/clave?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	defer pool.Close()

	q := db.New(pool)

	svc := services.InitServices(q)

	h := handlers.New(svc)

	api.Register(r, h)

	if err := http.ListenAndServe(":8000", r); err != nil {
		log.Fatal("failed to start server")
	}
}
