// Package testdb points tests at the database backend RACKD_TEST_BACKEND
// names: sqlite (the default), postgres or mysql. The server backends need
// xdaltest's connection variables (XDAL_TEST_POSTGRES_DSN,
// XDAL_TEST_MYSQL_DSN).
package testdb

import (
	"net/url"
	"os"
	"testing"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/martinsuchenak/xdal/pkg/xdal"
	"github.com/martinsuchenak/xdal/pkg/xdaltest"
)

// Env is the variable that selects the backend.
const Env = "RACKD_TEST_BACKEND"

// Backend returns the backend tests run on.
func Backend() string {
	if b := os.Getenv(Env); b != "" {
		return b
	}
	return xdal.DialectSQLite
}

// NewURL creates a fresh, empty database on the server backend b, dropped
// when the test ends, and returns its DATABASE_URL.
func NewURL(tb testing.TB, b string) string {
	tb.Helper()
	return URL(tb, xdaltest.Open(tb, b))
}

// URL returns a DATABASE_URL for the xdaltest database db.
func URL(tb testing.TB, db *xdaltest.DB) string {
	tb.Helper()
	switch db.Name {
	case xdal.DialectPostgres:
		var schema string
		if err := db.SQL.QueryRow("SELECT current_schema()").Scan(&schema); err != nil {
			tb.Fatal(err)
		}
		u, err := url.Parse(os.Getenv(xdaltest.EnvPostgresDSN))
		if err != nil {
			tb.Fatal(err)
		}
		q := u.Query()
		q.Set("search_path", schema)
		u.RawQuery = q.Encode()
		return u.String()
	case xdal.DialectMySQL:
		var name string
		if err := db.SQL.QueryRow("SELECT DATABASE()").Scan(&name); err != nil {
			tb.Fatal(err)
		}
		cfg, err := gomysql.ParseDSN(os.Getenv(xdaltest.EnvMySQLDSN))
		if err != nil {
			tb.Fatal(err)
		}
		u := url.URL{Scheme: "mysql", User: url.UserPassword(cfg.User, cfg.Passwd), Host: cfg.Addr, Path: "/" + name}
		return u.String()
	}
	tb.Fatalf("testdb: no DATABASE_URL for %s", db.Name)
	return ""
}
