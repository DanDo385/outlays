// Command migrate runs goose Up migrations against MIGRATE_DATABASE_URL (owner role).
package main

import (
	"log"
	"os"

	"github.com/djmagro/outlays/core/internal/store"
)

func main() {
	url := os.Getenv("MIGRATE_DATABASE_URL")
	if url == "" {
		log.Fatal("MIGRATE_DATABASE_URL is required")
	}
	if err := store.Migrate(url); err != nil {
		log.Fatal(err)
	}
}
