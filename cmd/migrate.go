package cmd

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/olekukonko/tablewriter"
	"github.com/oxygenpay/oxygen/internal/config"
	"github.com/oxygenpay/oxygen/scripts"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"
)

const (
	dbDialect       = "postgres"
	dbSchema        = "public"
	migrationsTable = "migrations"
)

var migrateCommand = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate DB",
	Long:  "Allows to use sql-migration commands: status, up, down",
	Run:   migration,
}

var migrateSelectedCommand string

func migration(_ *cobra.Command, _ []string) {
	performMigration(context.Background(), resolveConfig(), migrateSelectedCommand, false)
}

func performMigration(ctx context.Context, cfg *config.Config, command string, silent bool) {
	db := migrationConnection(ctx, cfg)
	source := scripts.MigrationFilesSource()
	migrationSet := &migrate.MigrationSet{
		SchemaName: dbSchema,
		TableName:  migrationsTable,
	}

	log.Printf("Using table %q.%q\n", dbSchema, migrationsTable)

	switch command {
	case "up":
		log.Println("Running migrations")
		_, err := migrationSet.Exec(db, dbDialect, source, migrate.Up)
		if err != nil {
			log.Fatalf("Error while running migrations: %s\n", err.Error())
		}

		log.Println("Applied migrations ✔")

		if !silent {
			migrationStatus(db, migrationSet)
		}
	case "down":
		log.Println("Rolling back migrations")
		_, err := migrationSet.Exec(db, dbDialect, source, migrate.Down)
		if err != nil {
			log.Fatalf("Error while running migrations: %s\n", err.Error())
		}

		log.Println("Rolled back migrations ✔")
		if !silent {
			migrationStatus(db, migrationSet)
		}

	case "status":
		migrationStatus(db, migrationSet)

	default:
		log.Fatalf("Unknown --command %q\n", migrateSelectedCommand)
	}

	if err := db.Close(); err != nil {
		log.Fatalf("Unable to close migration db connection: %s", err.Error())
	}
}

func migrationConnection(ctx context.Context, cfg *config.Config) *sql.DB {
	connCfg, err := parseConn(cfg.Oxygen.Postgres.DataSource)
	if err != nil {
		log.Fatalf("unable to parse DB connection: %s\n", err.Error())
	}

	db := sql.OpenDB(stdlib.GetConnector(*connCfg))

	if _, err = db.Conn(ctx); err != nil {
		log.Fatalf("unable to connect to DB: %s\n", err.Error())
	}

	return db
}

func migrationStatus(db *sql.DB, set *migrate.MigrationSet) {
	items, err := set.GetMigrationRecords(db, dbDialect)
	if err != nil {
		log.Fatalf("Status error: %s", err.Error())
	}

	t := tablewriter.NewWriter(os.Stdout)
	defer t.Render()

	t.SetHeader([]string{"Migration", "Timestamp"})
	for _, item := range items {
		t.Append([]string{item.Id, item.AppliedAt.Format(time.RFC3339)})
	}
}

// parseConn strips irrelevant pgx pool configuration params while preserving all
// connection-level settings, including TLS verification options such as
// sslmode=verify-full, sslrootcert, sslcert, and sslkey.
func parseConn(raw string) (*pgx.ConnConfig, error) {
	poolCfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		return nil, err
	}

	return poolCfg.ConnConfig, nil
}
