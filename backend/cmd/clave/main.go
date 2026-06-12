package main

import (
	"log"
	"net/http"

	"github.com/cheetahbyte/clave/internal/app"
	"github.com/cheetahbyte/clave/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	cfg.ConfigureLogging()

	if cfg.RunMigrations {
		app.RunMigrations(cfg.DatabaseURL)
	}

	handler, err := app.NewRouter(cfg)
	if err != nil {
		log.Fatal(err)
	}

	addr := "0.0.0.0:" + cfg.Port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal("failed to start server")
	}
}
