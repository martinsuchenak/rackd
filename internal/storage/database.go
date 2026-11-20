package storage

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/martinsuchenak/xdal/pkg/mysql"
	"github.com/martinsuchenak/xdal/pkg/postgres"
	"github.com/martinsuchenak/xdal/pkg/sqlite"
	"github.com/martinsuchenak/xdal/pkg/xdal"
)

// Rackd keeps its data in SQLite by default: rackd.db in the data directory.
// DATABASE_URL points it at PostgreSQL or MySQL 8 instead:
//
//	postgres://user:pass@host:5432/rackd?sslmode=require
//	mysql://user:pass@host:3306/rackd?tls=true
//
// A MySQL URL's query parameters are the driver's DSN parameters; rackd always
// reads times back as UTC and sets the session time zone to UTC.

// serverPoolSize is the connection pool size on PostgreSQL and MySQL.
const serverPoolSize = 16

// backendOf names the backend db is connected to, from its driver.
func backendOf(db *sql.DB) string {
	switch db.Driver().(type) {
	case *stdlib.Driver:
		return xdal.DialectPostgres
	case *gomysql.MySQLDriver:
		return xdal.DialectMySQL
	}
	return xdal.DialectSQLite
}

// xdalFor wraps db for xdal in its backend's dialect.
func xdalFor(db *sql.DB) xdal.DBInterface {
	switch backendOf(db) {
	case xdal.DialectPostgres:
		return postgres.NewPostgresDB(db, nil)
	case xdal.DialectMySQL:
		return mysql.NewMySQLDB(db, nil)
	}
	return sqlite.NewSQLiteDB(db, nil)
}

// isSQLite reports whether db is a SQLite database.
func isSQLite(db *sql.DB) bool { return backendOf(db) == xdal.DialectSQLite }

// DatabaseBackend names the backend a DATABASE_URL selects: sqlite when it is
// empty, otherwise postgres or mysql. It reports a URL rackd can't use.
func DatabaseBackend(databaseURL string) (string, error) {
	if databaseURL == "" {
		return xdal.DialectSQLite, nil
	}
	scheme, _, ok := strings.Cut(databaseURL, "://")
	if !ok {
		return "", fmt.Errorf("DATABASE_URL must start with postgres:// or mysql://")
	}
	switch strings.ToLower(scheme) {
	case "postgres", "postgresql":
		return xdal.DialectPostgres, nil
	case "mysql":
		return xdal.DialectMySQL, nil
	}
	return "", fmt.Errorf("DATABASE_URL: unsupported database %q; use postgres:// or mysql://, or leave it empty for SQLite", scheme)
}

// openServerDB opens the PostgreSQL or MySQL database databaseURL names and
// checks that it answers.
func openServerDB(ctx context.Context, databaseURL string) (*sql.DB, error) {
	backend, err := DatabaseBackend(databaseURL)
	if err != nil {
		return nil, err
	}
	var db *sql.DB
	switch backend {
	case xdal.DialectPostgres:
		cfg, err := pgx.ParseConfig(databaseURL)
		if err != nil {
			// pgx leaves the password out of its message.
			return nil, fmt.Errorf("DATABASE_URL: %w", err)
		}
		db = openPostgresPool(cfg)
	case xdal.DialectMySQL:
		cfg, err := mysqlConfig(databaseURL)
		if err != nil {
			return nil, err
		}
		connector, err := gomysql.NewConnector(cfg)
		if err != nil {
			return nil, fmt.Errorf("DATABASE_URL: %w", err)
		}
		db = sql.OpenDB(connector)
	default:
		return nil, fmt.Errorf("openServerDB: %s is not a database server", backend)
	}
	db.SetMaxOpenConns(serverPoolSize)
	db.SetMaxIdleConns(serverPoolSize)
	db.SetConnMaxLifetime(time.Hour)

	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to the %s database: %w", backend, err)
	}
	return db, nil
}

// mysqlConfig turns a mysql:// URL into the driver's configuration.
func mysqlConfig(databaseURL string) (*gomysql.Config, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL: %w", redactURLError(err))
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return nil, fmt.Errorf("DATABASE_URL: name the MySQL database, as in mysql://user:pass@host:3306/rackd")
	}
	// The query string carries the driver's own DSN parameters.
	cfg, err := gomysql.ParseDSN("/" + name + "?" + u.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL: %w", err)
	}
	cfg.User = u.User.Username()
	cfg.Passwd, _ = u.User.Password()
	cfg.Net = "tcp"
	cfg.Addr = u.Host
	if u.Port() == "" {
		cfg.Addr = u.Host + ":3306"
	}
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	// DATETIME defaults (CURRENT_TIMESTAMP) follow the session time zone.
	cfg.Params["time_zone"] = "'+00:00'"
	return cfg, nil
}

// redactURLError drops the URL, which may hold a password, from a url.Parse
// error.
func redactURLError(err error) error {
	if ue, ok := err.(*url.Error); ok {
		return ue.Err
	}
	return err
}

// OpenDatabase opens rackd's database without migrating it, for commands that
// inspect or migrate it: the SQLite file in dataDir when databaseURL is empty,
// which must exist, otherwise the database server databaseURL names.
func OpenDatabase(ctx context.Context, dataDir, databaseURL string) (*sql.DB, error) {
	if databaseURL != "" {
		return openServerDB(ctx, databaseURL)
	}
	dbPath := filepath.Join(dataDir, "rackd.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database not found at %s", dbPath)
	}
	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return db, nil
}

// Backend names the backend storage runs on: sqlite, postgres or mysql.
func (s *SQLiteStorage) Backend() string { return s.q.x.Dialect().Name() }
