package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var databaseURL string
	var migrationsPath string
	var command string

	flag.StringVar(&databaseURL, "database", "", "Database URL (required)")
	flag.StringVar(&migrationsPath, "path", "migrations", "Path to migrations directory")
	flag.StringVar(&command, "command", "up", "Migration command: up, down, version, force")
	flag.Parse()

	// Check for database URL from flag or environment
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}

	if databaseURL == "" {
		log.Fatal("Database URL is required. Use -database flag or DATABASE_URL environment variable")
	}

	// Clean up the URL (remove extra query params that might cause issues)
	// The URL should work as-is, but we'll log it for debugging
	log.Printf("Connecting to database...")
	log.Printf("Migrations path: %s", migrationsPath)

	// Create migration instance
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		databaseURL,
	)
	if err != nil {
		log.Fatalf("Failed to create migration instance: %v", err)
	}
	defer m.Close()

	// Execute command
	switch command {
	case "up":
		log.Println("Running migrations up...")
		err = m.Up()
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("No migrations to run (database is up to date)")
		} else {
			log.Println("Migrations completed successfully!")
		}

	case "down":
		log.Println("Rolling back migrations...")
		err = m.Down()
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		log.Println("Rollback completed successfully!")

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		log.Printf("Current version: %d (dirty: %v)", version, dirty)

	case "force":
		if len(flag.Args()) < 1 {
			log.Fatal("Force command requires a version number: -command force <version>")
		}
		var version int
		_, err := fmt.Sscanf(flag.Arg(0), "%d", &version)
		if err != nil {
			log.Fatalf("Invalid version number: %v", err)
		}
		err = m.Force(version)
		if err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}
		log.Printf("Forced version to: %d", version)

	default:
		log.Fatalf("Unknown command: %s (use: up, down, version, force)", command)
	}
}
