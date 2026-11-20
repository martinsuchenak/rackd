package storage

import (
	"strings"

	"github.com/martinsuchenak/xdal/pkg/xdal"
)

// textMatch returns the join, condition and arguments that match query
// against the text columns of table (aliased as alias) for a search.
//
// SQLite has FTS5 tables (<table>_fts, kept current by triggers) and matches
// the query as a phrase whose last word may be a prefix. PostgreSQL and MySQL
// match it as a case-insensitive substring of any of the columns: their
// full-text search differs from FTS5 and from each other (MySQL drops short
// words and stopwords), while a substring finds what a user typing part of a
// name expects, and inventories are small enough to scan.
func (s *SQLiteStorage) textMatch(table, alias, query string, columns ...string) (join, where string, args []any) {
	if s.q.x.Dialect().Name() == xdal.DialectSQLite {
		fts := table + "_fts"
		return "INNER JOIN " + fts + " fts ON " + alias + ".id = fts.id", fts + " MATCH ?", []any{escapeFTSQuery(query)}
	}
	// PostgreSQL refuses a NUL in text.
	query = strings.ReplaceAll(query, "\x00", "")
	pattern := "%" + escapeLike(strings.ToLower(query)) + "%"
	conds := make([]string, len(columns))
	args = make([]any, len(columns))
	for i, c := range columns {
		conds[i] = "LOWER(" + alias + "." + c + ") LIKE ?"
		args[i] = pattern
	}
	return "", "(" + strings.Join(conds, " OR ") + ")", args
}

// escapeLike makes s match literally in a LIKE pattern on PostgreSQL and
// MySQL, whose LIKE escape character is a backslash by default.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
