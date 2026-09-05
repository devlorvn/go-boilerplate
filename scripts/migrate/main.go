package main

import (
	"fmt"
	"log"
	"os"

	"go-boilerplate/internal/golibs/configs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/migrate/main.go [up|down]")
		os.Exit(1)
	}

	cmd := os.Args[1]

	cfg, err := configs.LoadConfig(".")
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	dsn := cfg.DB.MigrationConnectionString()

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Fatalf("❌ Failed to connect Migration Engine: %v", err)
	}
	defer m.Close()

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("❌ Migration UP failed: %v", err)
		}
		fmt.Println("✅ PostgreSQL Migrations UP executed successfully!")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("❌ Migration DOWN failed: %v", err)
		}
		fmt.Println("✅ PostgreSQL Migrations DOWN rolled back successfully!")
	default:
		fmt.Printf("Invalid command: %s. Use 'up' or 'down'.\n", cmd)
	}
}
