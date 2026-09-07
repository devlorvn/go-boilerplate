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

type serviceConfig struct {
	name     string
	envFile  string
	folder   string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/migrate/main.go [auth|file|all] [up|down]")
		os.Exit(1)
	}

	target := "all"
	cmd := os.Args[1]
	if len(os.Args) >= 3 {
		target = os.Args[1]
		cmd = os.Args[2]
	}

	var services []serviceConfig

	switch target {
	case "auth":
		services = append(services, serviceConfig{"auth", ".env.auth", "migrations/auth"})
	case "file":
		services = append(services, serviceConfig{"file", ".env.file", "migrations/file"})
	case "all", "up", "down":
		if target == "up" || target == "down" {
			cmd = target
		}
		services = append(services,
			serviceConfig{"auth", ".env.auth", "migrations/auth"},
			serviceConfig{"file", ".env.file", "migrations/file"},
		)
	default:
		log.Fatalf("Invalid target: %s. Use 'auth', 'file', or 'all'.", target)
	}

	for _, svc := range services {
		cfg, err := configs.LoadConfig(".", svc.envFile)
		if err != nil {
			log.Fatalf("Failed to load config %s for service %s: %v", svc.envFile, svc.name, err)
		}

		dsn := cfg.DB.MigrationConnectionString()
		m, err := migrate.New(fmt.Sprintf("file://%s", svc.folder), dsn)
		if err != nil {
			log.Fatalf("Failed to connect Migration Engine for service %s (DB: %s): %v", svc.name, cfg.DB.DBName, err)
		}

		switch cmd {
		case "up":
			if err := m.Up(); err != nil && err != migrate.ErrNoChange {
				m.Close()
				log.Fatalf("Migration UP failed for service %s: %v", svc.name, err)
			}
			fmt.Printf("PostgreSQL Migrations UP executed successfully for service '%s' (DB: %s)!\n", svc.name, cfg.DB.DBName)
		case "down":
			if err := m.Down(); err != nil && err != migrate.ErrNoChange {
				m.Close()
				log.Fatalf("Migration DOWN failed for service %s: %v", svc.name, err)
			}
			fmt.Printf("PostgreSQL Migrations DOWN rolled back successfully for service '%s' (DB: %s)!\n", svc.name, cfg.DB.DBName)
		default:
			m.Close()
			log.Fatalf("Invalid command: %s. Use 'up' or 'down'.\n", cmd)
		}
		m.Close()
	}
}
