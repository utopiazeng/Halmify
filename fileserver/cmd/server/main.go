package main

import (
	"fileserver/internal/app"
	"fileserver/internal/config"
	"fmt"
	"log"
	"os"
)

func main() {
	command := "server"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if command == "-h" || command == "--help" || command == "help" {
		printUsage()
		return
	}
	if command != "server" && command != "migrate" {
		printUsage()
		log.Fatalf("unknown command: %s", command)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	if command == "migrate" {
		if err := app.RunMigration(cfg); err != nil {
			log.Fatalf("migrate failed: %v", err)
		}
		log.Printf("migration finished")
		return
	}

	if err := app.RunServer(cfg); err != nil {
		log.Fatalf("run failed: %v", err)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  fileserver          Start fileserver")
	fmt.Println("  fileserver server   Start fileserver")
	fmt.Println("  fileserver migrate  Run database migration")
}
