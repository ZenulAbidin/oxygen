package test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/oxygenpay/oxygen/internal/db/connection/pg"
	"github.com/oxygenpay/oxygen/internal/util"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type Database struct {
	genericConnection *pg.Connection
	context           context.Context
	connection        *pg.Connection
	name              string
}

const (
	dbHost                 = "127.0.0.1"
	dbUser                 = "postgres"
	dbPassword             = ""
	maxConnections         = 64
	migrationsRelativePath = "../../scripts/migrations/"
	testDBDataSourceEnv    = "OXYGEN_TEST_DB_DATA_SOURCE"
	testDBConnectTimeout   = 5 * time.Second
)

func NewDB(ctx context.Context) *Database {
	nopLogger := zerolog.Nop()

	cfg, err := testDatabaseConfig("")
	if err != nil {
		panic(err.Error())
	}

	conn, err := openTestDBConnection(ctx, cfg, &nopLogger, "admin")
	if err != nil {
		panic(err.Error())
	}

	db := &Database{
		context:           ctx,
		genericConnection: conn,
		name:              "oxygen_test_" + util.Strings.Random(8),
	}

	// create tmp database
	if _, errDB := conn.Exec(ctx, "create database "+db.name); errDB != nil {
		panic("unable to create test database: " + errDB.Error())
	}

	// connect to the tmp database
	tmpConnectionCfg, err := testDatabaseConfig(db.name)
	if err != nil {
		panic(err.Error())
	}

	tmpConn, err := openTestDBConnection(ctx, tmpConnectionCfg, &nopLogger, db.name)
	if err != nil {
		panic(err.Error())
	}

	db.connection = tmpConn

	db.applyMigrations()

	return db
}

func (db *Database) Conn() *pg.Connection {
	return db.connection
}

func (db *Database) Context() context.Context {
	return db.context
}

func (db *Database) Name() string {
	return db.name
}

func (db *Database) TearDown() {
	if err := db.connection.Shutdown(); err != nil {
		panic("unable to close test database: " + err.Error())
	}

	if _, err := db.genericConnection.Exec(db.context, "drop database "+db.name); err != nil {
		panic("unable to drop test database: " + err.Error())
	}

	if err := db.genericConnection.Shutdown(); err != nil {
		panic("unable to close test generic database: " + err.Error())
	}
}

//nolint:dogsled
func (db *Database) applyMigrations() {
	// locate migrations directory relative to this very file
	_, filename, _, _ := runtime.Caller(1)
	currentDir := path.Dir(filename)

	migrationsDirectory := path.Join(currentDir, migrationsRelativePath)

	// get all filenames
	migrations, err := os.ReadDir(migrationsDirectory)
	if err != nil {
		db.TearDown()
		panic("unable to open migrations directory: " + err.Error())
	}

	// apply migrations
	for _, file := range migrations {
		db.applySingleMigration(migrationsDirectory + "/" + file.Name())
	}
}

func (db *Database) applySingleMigration(filename string) {
	// open file
	f, err := os.Open(filename)
	if err != nil {
		panic("unable to open " + filename)
	}
	defer f.Close()

	// read all
	bytes, err := io.ReadAll(f)
	if err != nil {
		panic("unable to read " + filename)
	}

	text := string(bytes)

	// cut "down" command
	lastIndex := strings.Index(text, "-- +migrate Down")
	if lastIndex == -1 {
		panic("invalid migration: " + filename)
	}

	migrationSQL := text[:lastIndex]

	// apply migration
	if _, err := db.connection.Exec(db.context, migrationSQL); err != nil {
		panic("unable to apply migration: " + filename)
	}
}

func testDatabaseConfig(dbName string) (*pgxpool.Config, error) {
	dataSource := strings.TrimSpace(os.Getenv(testDBDataSourceEnv))
	if dataSource == "" {
		dataSource = defaultTestDatabaseDataSource()
	}

	cfg, err := pgxpool.ParseConfig(dataSource)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse %s", testDBDataSourceEnv)
	}

	if dbName != "" {
		cfg.ConnConfig.Database = dbName
	}

	return cfg, nil
}

func openTestDBConnection(
	ctx context.Context,
	cfg *pgxpool.Config,
	logger *zerolog.Logger,
	label string,
) (*pg.Connection, error) {
	connectCtx, cancel := context.WithTimeout(ctx, testDBConnectTimeout)
	defer cancel()

	conn, err := pg.OpenConfig(connectCtx, cfg, logger)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"unable to open %s database within %s; ensure postgres is reachable or set %s",
			label,
			testDBConnectTimeout,
			testDBDataSourceEnv,
		)
	}

	return conn, nil
}

func defaultTestDatabaseDataSource() string {
	parts := []string{
		fmt.Sprintf("user=%s", dbUser),
		fmt.Sprintf("host=%s", dbHost),
		"dbname=postgres",
		"sslmode=disable",
		fmt.Sprintf("pool_max_conns=%d", maxConnections),
	}

	if dbPassword != "" {
		parts = append(parts, fmt.Sprintf("password=%s", dbPassword))
	}

	return strings.Join(parts, " ")
}
