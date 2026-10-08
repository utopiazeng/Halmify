package main

import (
	"fileserver/internal/app"
	"fileserver/internal/config"
	"log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}
	if err := app.RunMigration(cfg); err != nil {
		log.Fatalf("migrate failed: %v", err)
	}
	log.Printf("migration finished")
}
