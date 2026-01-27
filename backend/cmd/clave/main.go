package main

import (
	"log"
	"net/http"

	"github.com/cheetahbyte/clave/internal/api"
	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()

	api.Register(r)

	if err := http.ListenAndServe(":8000", r); err != nil {
		log.Fatal("failed to start server")
	}
}
