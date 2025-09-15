package cmd

import (
	"context"
	"log"
	"os"

	"github.com/frahmantamala/expense-management/internal"

	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
)

var (
	migrateCmd = &cobra.Command{
		RunE:  runMigration,
		Use:   "migrate",
		Short: "to run db migration files under db/migrations directory",
	}
	migrateRollback bool
	migrateDir      string
)

func init() {
	migrateCmd.Flags().BoolVarP(&migrateRollback, "rollback", "r", false, "to rollback the latest version of sql migration")
	migrateCmd.PersistentFlags().StringVarP(&migrateDir, "dir", "d", "db/migrations", "sql migrations directory")
}

func runMigration(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	var cfg *internal.Config
	var err error

	if dbSource := os.Getenv("DB_SOURCE"); dbSource != "" {
		cfg = internal.LoadConfigFromEnv()
	} else {
		cfg, err = loadConfig(".")
		if err != nil {
			log.Fatal(err)
		}
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}

	db, err := goose.OpenDBWithDriver("pgx", cfg.Database.Source)
	if err != nil {
		log.Fatalf("goose: failed to open DB: %v\n", err)
	}
	goose.SetTableName("schema_migrations")

	if err := goose.RunContext(ctx, "up", db, migrateDir); err != nil {
		log.Fatalf("goose up: %v", err)
	}

	return nil
}
