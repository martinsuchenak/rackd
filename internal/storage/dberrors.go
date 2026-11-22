package storage

import (
	"errors"
	"regexp"
	"strings"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
)

// Constraint violations come back differently from each backend: SQLite
// names the table and column in its message, PostgreSQL and MySQL return
// driver errors with a code. These helpers read all three.

// SQLite: "UNIQUE constraint failed: users.username", or for a key over
// several columns "... reservations.pool_id, reservations.ip_address"; the
// last column is reported, as for PostgreSQL.
var sqliteConstraint = regexp.MustCompile(`(UNIQUE|NOT NULL) constraint failed: ((?:[A-Za-z_]+\.[A-Za-z_]+)(?:, [A-Za-z_]+\.[A-Za-z_]+)*)`)

// PostgreSQL's detail for a unique violation: "Key (username)=(alice) already exists."
var pgKeyDetail = regexp.MustCompile(`^Key \(([^)]+)\)=`)

// MySQL: "Duplicate entry 'alice' for key 'users.username'" and
// "Column 'name' cannot be null".
var (
	mysqlDuplicateKey = regexp.MustCompile(`for key '(?:([^'.]+)\.)?([^']+)'`)
	mysqlNullColumn   = regexp.MustCompile(`Column '([^']+)' cannot be null`)
	// xdal names table-level unique constraints uq_<table>_<columns>_<hash>.
	xdalUniqueName = regexp.MustCompile(`^uq_(.+)_[0-9a-f]{8}$`)
)

// mysqlExpressionIndexColumn names the column of the baseline's partial
// unique indexes, which MySQL builds as expression indexes and names by
// index: the column PostgreSQL reports for the same violation.
var mysqlExpressionIndexColumn = map[string]string{
	"idx_conflicts_active_duplicate_ip": "ip_address",
	"idx_reservations_pool_ip_unique":   "ip_address",
}

// PostgreSQL SQLSTATE and MySQL error numbers for the violations.
const (
	pgUniqueViolation     = "23505"
	pgNotNullViolation    = "23502"
	pgForeignKeyViolation = "23503"

	mysqlDuplicateEntry   = 1062
	mysqlColumnNull       = 1048
	mysqlNoReferencedRow  = 1452 // insert or update refers to a missing row
	mysqlRowIsReferenced  = 1451 // delete of a row still referred to
	mysqlNoReferencedRow1 = 1216
	mysqlRowIsReferenced1 = 1217
)

// DuplicateField reports whether err is a unique-constraint violation, and the
// column it was on, such as "name". The column is empty when the backend
// doesn't say (MySQL, for a key over several columns).
func DuplicateField(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		if pg.Code != pgUniqueViolation {
			return "", false
		}
		if m := pgKeyDetail.FindStringSubmatch(pg.Detail); m != nil {
			return lastColumn(m[1]), true
		}
		return pg.ColumnName, true
	}
	var my *gomysql.MySQLError
	if errors.As(err, &my) {
		if my.Number != mysqlDuplicateEntry {
			return "", false
		}
		// MySQL names the index, not the column. xdal names a unique
		// constraint uq_<table>_<columns>_<hash>, so the columns follow the
		// table name (joined by "_" for a key over several). A primary key
		// or a named index doesn't say.
		m := mysqlDuplicateKey.FindStringSubmatch(my.Message)
		if m == nil {
			return "", true
		}
		if col, ok := mysqlExpressionIndexColumn[m[2]]; ok {
			return col, true
		}
		if u := xdalUniqueName.FindStringSubmatch(m[2]); u != nil && strings.HasPrefix(u[1], m[1]+"_") {
			return strings.TrimPrefix(u[1], m[1]+"_"), true
		}
		if m[2] == "PRIMARY" || strings.HasPrefix(m[2], "idx_") || strings.HasPrefix(m[2], "uq_") {
			return "", true
		}
		return m[2], true
	}
	return sqliteColumn(err, "UNIQUE")
}

// MissingField reports whether err is a NOT NULL violation, and its column.
func MissingField(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		return pg.ColumnName, pg.Code == pgNotNullViolation
	}
	var my *gomysql.MySQLError
	if errors.As(err, &my) {
		if my.Number != mysqlColumnNull {
			return "", false
		}
		if m := mysqlNullColumn.FindStringSubmatch(my.Message); m != nil {
			return m[1], true
		}
		return "", true
	}
	return sqliteColumn(err, "NOT NULL")
}

// IsForeignKeyFailure reports whether err is a foreign-key violation: a write
// that refers to a missing record, or a delete of one still referred to.
func IsForeignKeyFailure(err error) bool {
	if err == nil {
		return false
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		return pg.Code == pgForeignKeyViolation
	}
	var my *gomysql.MySQLError
	if errors.As(err, &my) {
		switch my.Number {
		case mysqlNoReferencedRow, mysqlRowIsReferenced, mysqlNoReferencedRow1, mysqlRowIsReferenced1:
			return true
		}
		return false
	}
	return strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}

// isUniqueViolation reports whether err is a unique-constraint violation on
// any backend.
func isUniqueViolation(err error) bool {
	_, ok := DuplicateField(err)
	return ok
}

func sqliteColumn(err error, kind string) (string, bool) {
	m := sqliteConstraint.FindStringSubmatch(err.Error())
	if m == nil || m[1] != kind {
		return "", false
	}
	last := lastColumn(m[2])
	return last[strings.LastIndex(last, ".")+1:], true
}

// lastColumn returns the last of a comma-separated column list: for a
// composite key such as (pool_id, ip_address) the distinguishing column.
func lastColumn(cols string) string {
	parts := strings.Split(cols, ",")
	return strings.TrimSpace(parts[len(parts)-1])
}
