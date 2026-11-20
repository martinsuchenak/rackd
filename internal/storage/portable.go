package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/martinsuchenak/xdal/pkg/xdal"
)

// dbConn is what storage queries run on: the database, or a transaction.
// *sql.DB, *sql.Tx, portableDB and portableTx all satisfy it.
type dbConn interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// portableDB runs hand-written SQL on any backend. Queries are written with
// "?" placeholders and in SQL every backend understands; portableDB rewrites
// the placeholders for the backend ($1, $2... on PostgreSQL). SQL that
// differs between backends (upserts, case-insensitive matching, booleans,
// full-text search) goes through xdal's query builder or a dialect helper
// instead. Builder output passes through unchanged.
type portableDB struct {
	x xdal.DBInterface
}

func (p portableDB) translate(q string) string { return p.x.Dialect().TranslateSQL(q) }

func (p portableDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return p.x.Exec(ctx, p.translate(query), args...)
}

func (p portableDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return p.x.Query(ctx, p.translate(query), args...)
}

func (p portableDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return p.x.QueryRow(ctx, p.translate(query), args...)
}

// BeginTx starts a transaction whose queries are translated the same way.
func (p portableDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*portableTx, error) {
	tx, err := p.x.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &portableTx{tx: tx, d: p.x.Dialect()}, nil
}

// builder returns a query builder for the backend.
func (p portableDB) builder() *xdal.QueryBuilder { return p.x.NewQueryBuilder() }

// ident quotes a column or table name for the backend. Only names that are
// reserved somewhere need it: "key" is reserved in MySQL.
func (p portableDB) ident(name string) string { return p.x.Dialect().QuoteIdentifier(name) }

// portableTx is a transaction on a portableDB.
type portableTx struct {
	tx *xdal.Tx
	d  xdal.Dialect
}

func (t *portableTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.Exec(ctx, t.d.TranslateSQL(query), args...)
}

func (t *portableTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.tx.Query(ctx, t.d.TranslateSQL(query), args...)
}

func (t *portableTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return t.tx.QueryRow(ctx, t.d.TranslateSQL(query), args...)
}

// attempt runs fn inside a savepoint. If fn fails, its changes are rolled
// back and the transaction carries on: PostgreSQL would otherwise refuse
// every later statement in it, and on any backend a half-applied step would
// be committed with the rest.
func (t *portableTx) attempt(ctx context.Context, fn func() error) error {
	if _, err := t.ExecContext(ctx, "SAVEPOINT rackd_step"); err != nil {
		return err
	}
	if err := fn(); err != nil {
		if _, rbErr := t.ExecContext(ctx, "ROLLBACK TO SAVEPOINT rackd_step"); rbErr != nil {
			return fmt.Errorf("%w (and rolling it back failed: %v)", err, rbErr)
		}
		return err
	}
	_, err := t.ExecContext(ctx, "RELEASE SAVEPOINT rackd_step")
	return err
}

func (t *portableTx) Commit() error   { return t.tx.Commit() }
func (t *portableTx) Rollback() error { return t.tx.Rollback() }

// execBuilt builds a builder query and runs it on c.
func execBuilt(ctx context.Context, c dbConn, b interface {
	Build() (string, []interface{}, error)
}) (sql.Result, error) {
	q, args, err := b.Build()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}
	return c.ExecContext(ctx, q, args...)
}

// likeLower returns a condition matching column against a LIKE pattern
// without regard to case. SQLite's and MySQL's LIKE already ignore case;
// PostgreSQL's doesn't, so both sides are lower-cased everywhere.
func likeLower(column string) string {
	return "LOWER(" + column + ") LIKE LOWER(?)"
}
