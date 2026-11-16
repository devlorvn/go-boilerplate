package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"go-boilerplate/internal/golibs/configs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
)

// newMigrateCmd applies migrations/<service> to the database configured for
// that service (same config layers as gserver, so env overrides apply too).
func newMigrateCmd(configDir *string) *cobra.Command {
	var (
		migrationsDir string
		steps         int
	)
	cmd := &cobra.Command{
		Use:   "migrate <service> up|down",
		Short: "Apply (up) or roll back (down --steps N) a service's SQL migrations",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, direction := args[0], args[1]

			cfg, err := configs.Load[configs.Base](svc, *configDir)
			if err != nil {
				return err
			}
			src := "file://" + filepath.ToSlash(filepath.Join(migrationsDir, svc))
			m, err := migrate.New(src, cfg.Postgres.MigrationConnectionString())
			if err != nil {
				return fmt.Errorf("migrate %s (db %s): %w", svc, cfg.Postgres.DBName, err)
			}
			defer m.Close()

			switch direction {
			case "up":
				err = m.Up()
			case "down":
				if steps < 1 {
					return errors.New("--steps must be >= 1")
				}
				err = m.Steps(-steps)
			default:
				return fmt.Errorf("unknown direction %q, use up or down", direction)
			}
			if err != nil && !errors.Is(err, migrate.ErrNoChange) {
				return fmt.Errorf("migrate %s %s: %w", svc, direction, err)
			}

			version, dirty, _ := m.Version()
			fmt.Printf("%s: migrated %s (db %s, version %d, dirty %v)\n", svc, direction, cfg.Postgres.DBName, version, dirty)
			return nil
		},
	}
	cmd.Flags().StringVar(&migrationsDir, "migrations-dir", "migrations", "root folder holding one sub-folder per service")
	cmd.Flags().IntVar(&steps, "steps", 1, "number of migrations to roll back with down")
	return cmd
}
