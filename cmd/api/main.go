package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"example.com/notes-api-pz14/internal/config"
	"example.com/notes-api-pz14/internal/db"
	"example.com/notes-api-pz14/internal/notes"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()

	dbConn, err := db.Open(ctx, cfg.DatabaseURL, cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetime, cfg.ConnMaxIdleTime)
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.SQL.Close()

	repo, err := notes.NewRepository(ctx, dbConn.SQL)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           notes.NewHandlers(repo).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Notes API listening on %s", cfg.HTTPAddr)
	log.Fatal(srv.ListenAndServe())
}
