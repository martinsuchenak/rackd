package storage

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/martinsuchenak/xdal/pkg/migrate"
	"github.com/martinsuchenak/xdal/pkg/xdal"
)

// CopyDatabase copies every row of rackd's SQLite database src into dst, a
// PostgreSQL or MySQL database whose schema is at the same version, replacing
// the rows dst has (the roles, permissions and scan profiles its migrations
// seed).
//
// It reads all of src first and returns a *CopyCheckError, writing nothing,
// when some of it can't be stored in dst: text longer than its column, a NUL
// character on PostgreSQL, a time it can't read, or a reference to a row that
// no longer exists. The rows are then written in one transaction, and the row
// counts compared before it commits. progress, if set, is called as each
// table is copied.
func CopyDatabase(ctx context.Context, src, dst *sql.DB, progress func(table string, rows int)) ([]CopiedTable, error) {
	if !isSQLite(src) {
		return nil, fmt.Errorf("the database to copy from must be SQLite")
	}
	if isSQLite(dst) {
		return nil, fmt.Errorf("the database to copy to must be PostgreSQL or MySQL")
	}
	if err := CheckCopyTarget(ctx, dst); err != nil {
		return nil, err
	}
	// Both passes read one snapshot. SQLite transactions here start with
	// BEGIN IMMEDIATE, which also holds off a server left running.
	stx, err := src.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("reading the SQLite database: %w", err)
	}
	defer func() { _ = stx.Rollback() }()
	plan, err := planCopy(ctx, src, stx, dst)
	if err != nil {
		return nil, err
	}
	if err := plan.check(ctx, stx); err != nil {
		return nil, err
	}
	return plan.write(ctx, stx, dst, progress)
}

// seededTables are filled by the migrations themselves, so a new database
// has rows in them; a copy replaces those rows.
var seededTables = map[string]bool{"roles": true, "permissions": true, "role_permissions": true, "scan_profiles": true}

// CheckCopyTarget refuses a target database for CopyDatabase that holds data:
// one with tables that aren't rackd's, or with rackd's and rows beyond those
// its migrations seed. An empty database, or one a failed copy left with
// rackd's schema and no data, is accepted.
func CheckCopyTarget(ctx context.Context, dst *sql.DB) error {
	if isSQLite(dst) {
		return fmt.Errorf("the database to copy to must be PostgreSQL or MySQL")
	}
	if !tableExists(ctx, dst, migrate.DefaultTable) {
		n, err := countTables(ctx, dst)
		if err != nil {
			return err
		}
		if n > 0 {
			return fmt.Errorf("the target database has %d tables that aren't rackd's; copy into an empty database", n)
		}
		return nil
	}
	cols, err := targetColumns(ctx, dst)
	if err != nil {
		return err
	}
	d := xdalFor(dst).Dialect()
	var withData []string
	for table := range cols {
		if seededTables[table] {
			continue
		}
		var n int
		if err := dst.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+d.QuoteIdentifier(table)).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			withData = append(withData, fmt.Sprintf("%s (%d rows)", table, n))
		}
	}
	if len(withData) > 0 {
		slices.Sort(withData)
		return fmt.Errorf("the target database already holds rackd data, which a copy would replace: %s. Copy into an empty database", strings.Join(withData, ", "))
	}
	return nil
}

// querier reads the source database: the snapshot transaction.
type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// CopiedTable is a table CopyDatabase copied and its row count.
type CopiedTable struct {
	Name string
	Rows int
}

// CopyProblem is a value CopyDatabase can't store in the target database.
type CopyProblem struct {
	Table, Column, Row, Problem string
}

func (p CopyProblem) String() string {
	if p.Column == "" {
		return fmt.Sprintf("%s %s: %s", p.Table, p.Row, p.Problem)
	}
	return fmt.Sprintf("%s.%s of %s: %s", p.Table, p.Column, p.Row, p.Problem)
}

// CopyCheckError lists the values that stop a copy. Problems holds the first
// maxCopyProblems; Total counts them all.
type CopyCheckError struct {
	Problems []CopyProblem
	Total    int
}

const maxCopyProblems = 100

func (e *CopyCheckError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d value(s) can't be copied; fix them in the SQLite database and run the copy again. Nothing was written:", e.Total)
	for _, p := range e.Problems {
		b.WriteString("\n  ")
		b.WriteString(p.String())
	}
	if e.Total > len(e.Problems) {
		fmt.Fprintf(&b, "\n  ... and %d more", e.Total-len(e.Problems))
	}
	return b.String()
}

func (e *CopyCheckError) add(p CopyProblem) {
	if len(e.Problems) < maxCopyProblems {
		e.Problems = append(e.Problems, p)
	}
	e.Total++
}

// Kinds of target column, which decide how a SQLite value is converted.
const (
	kindText = iota
	kindBool
	kindTime
	kindInt
	kindFloat
	kindBytes
)

type copyColumn struct {
	name     string
	kind     int
	maxChars int64 // 0: unbounded
}

type copyTable struct {
	name    string
	columns []copyColumn
	hasID   bool
}

type foreignKey struct {
	table, column, refTable, refColumn string
}

type copyPlan struct {
	tables   []copyTable // parents before the tables that refer to them
	fks      []foreignKey
	postgres bool
	dst      xdal.Dialect
}

// planCopy reads the target's tables, columns and foreign keys, and checks
// that the source has the same tables and columns.
func planCopy(ctx context.Context, srcDB *sql.DB, src querier, dst *sql.DB) (*copyPlan, error) {
	if err := sameSchemaVersion(ctx, srcDB, dst); err != nil {
		return nil, err
	}
	plan := &copyPlan{postgres: backendOf(dst) == xdal.DialectPostgres, dst: xdalFor(dst).Dialect()}

	cols, err := targetColumns(ctx, dst)
	if err != nil {
		return nil, fmt.Errorf("failed to read the target's columns: %w", err)
	}
	if plan.fks, err = targetForeignKeys(ctx, dst); err != nil {
		return nil, fmt.Errorf("failed to read the target's foreign keys: %w", err)
	}
	srcTables, err := sourceTables(ctx, src)
	if err != nil {
		return nil, err
	}

	var mismatch []string
	for _, name := range srcTables {
		if _, ok := cols[name]; !ok {
			mismatch = append(mismatch, fmt.Sprintf("table %s is not in the target", name))
		}
	}
	names := make([]string, 0, len(cols))
	for name := range cols {
		names = append(names, name)
	}
	for _, name := range orderByForeignKeys(names, plan.fks) {
		srcCols, err := sourceColumns(ctx, src, name)
		if err != nil {
			return nil, err
		}
		if len(srcCols) == 0 {
			mismatch = append(mismatch, fmt.Sprintf("table %s is not in the SQLite database", name))
			continue
		}
		t := copyTable{name: name, columns: cols[name]}
		dstCols := map[string]bool{}
		for _, c := range t.columns {
			dstCols[c.name] = true
			t.hasID = t.hasID || c.name == "id"
			if !srcCols[c.name] {
				mismatch = append(mismatch, fmt.Sprintf("column %s.%s is not in the SQLite database", name, c.name))
			}
		}
		for c := range srcCols {
			if !dstCols[c] {
				mismatch = append(mismatch, fmt.Sprintf("column %s.%s is not in the target", name, c))
			}
		}
		plan.tables = append(plan.tables, t)
	}
	if len(mismatch) > 0 {
		slices.Sort(mismatch)
		return nil, fmt.Errorf("the two databases' schemas differ:\n  %s", strings.Join(mismatch, "\n  "))
	}
	return plan, nil
}

// sameSchemaVersion checks that both databases have the same portable and
// extension migrations applied and nothing pending.
func sameSchemaVersion(ctx context.Context, src, dst *sql.DB) error {
	versions := func(db *sql.DB) (map[string]bool, error) {
		out := map[string]bool{}
		statuses, err := GetSchemaMigrationStatus(ctx, db)
		if err != nil {
			return nil, err
		}
		for _, s := range statuses {
			if !s.Applied || s.Dirty {
				return nil, fmt.Errorf("schema migration %s %s is not applied", s.Version, s.Name)
			}
			out[SchemaMigrationNamespace+" "+s.Version] = true
		}
		namespaces, err := RecordedExtensionNamespaces(ctx, db)
		if err != nil {
			return nil, err
		}
		for _, ns := range namespaces {
			statuses, err := ExtensionMigrationStatus(ctx, db, ns, nil)
			if err != nil {
				return nil, err
			}
			for _, s := range statuses {
				if s.Dirty {
					return nil, fmt.Errorf("%s migration %s is dirty", ns, s.Version)
				}
				out[ns+" "+s.Version] = true
			}
		}
		return out, nil
	}
	have, err := versions(src)
	if err != nil {
		return fmt.Errorf("SQLite database: %w", err)
	}
	want, err := versions(dst)
	if err != nil {
		return fmt.Errorf("target database: %w", err)
	}
	var diff []string
	for v := range have {
		if !want[v] {
			diff = append(diff, v+" is applied to the SQLite database only")
		}
	}
	for v := range want {
		if !have[v] {
			diff = append(diff, v+" is applied to the target only")
		}
	}
	if len(diff) > 0 {
		slices.Sort(diff)
		return fmt.Errorf("the two databases' migrations differ:\n  %s", strings.Join(diff, "\n  "))
	}
	return nil
}

func targetColumns(ctx context.Context, dst *sql.DB) (map[string][]copyColumn, error) {
	schema := "current_schema()"
	if backendOf(dst) == xdal.DialectMySQL {
		schema = "DATABASE()"
	}
	rows, err := dst.QueryContext(ctx, `SELECT table_name, column_name, data_type, character_maximum_length
		FROM information_schema.columns WHERE table_schema = `+schema+` ORDER BY table_name, ordinal_position`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string][]copyColumn{}
	for rows.Next() {
		var table, column, dataType string
		var maxLen sql.NullInt64
		if err := rows.Scan(&table, &column, &dataType, &maxLen); err != nil {
			return nil, err
		}
		if table == migrate.DefaultTable {
			continue // each database keeps its own history
		}
		c := copyColumn{name: column, kind: columnKind(dataType)}
		if maxLen.Valid && strings.Contains(strings.ToLower(dataType), "char") {
			c.maxChars = maxLen.Int64
		}
		out[table] = append(out[table], c)
	}
	return out, rows.Err()
}

// columnKind classifies a PostgreSQL or MySQL column type. MySQL's BOOLEAN
// is TINYINT, the only TINYINT in the schema.
func columnKind(dataType string) int {
	t := strings.ToLower(dataType)
	switch {
	case t == "boolean" || t == "tinyint":
		return kindBool
	case strings.HasPrefix(t, "timestamp") || t == "datetime" || t == "date":
		return kindTime
	case strings.Contains(t, "int"):
		return kindInt
	case t == "double precision" || t == "double" || t == "real" || t == "float" || t == "numeric" || t == "decimal":
		return kindFloat
	case t == "bytea" || strings.Contains(t, "blob") || strings.Contains(t, "binary"):
		return kindBytes
	}
	return kindText
}

func targetForeignKeys(ctx context.Context, dst *sql.DB) ([]foreignKey, error) {
	q := `SELECT cl.relname, att.attname, rcl.relname, ratt.attname
		FROM pg_constraint c
		JOIN pg_class cl ON cl.oid = c.conrelid
		JOIN pg_class rcl ON rcl.oid = c.confrelid
		JOIN pg_attribute att ON att.attrelid = c.conrelid AND att.attnum = c.conkey[1]
		JOIN pg_attribute ratt ON ratt.attrelid = c.confrelid AND ratt.attnum = c.confkey[1]
		WHERE c.contype = 'f' AND c.connamespace = current_schema()::regnamespace`
	if backendOf(dst) == xdal.DialectMySQL {
		q = `SELECT table_name, column_name, referenced_table_name, referenced_column_name
			FROM information_schema.key_column_usage
			WHERE table_schema = DATABASE() AND referenced_table_name IS NOT NULL`
	}
	rows, err := dst.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []foreignKey
	for rows.Next() {
		var fk foreignKey
		if err := rows.Scan(&fk.table, &fk.column, &fk.refTable, &fk.refColumn); err != nil {
			return nil, err
		}
		out = append(out, fk)
	}
	return out, rows.Err()
}

// sourceTables lists the SQLite tables to copy: all but SQLite's own, the
// full-text search tables (the other backends search without them) and the
// migration history.
func sourceTables(ctx context.Context, src querier) ([]string, error) {
	rows, err := src.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite\_%' ESCAPE '\' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if name == migrate.DefaultTable || name == legacyMigrationTable || strings.Contains(name, "_fts") {
			continue
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func sourceColumns(ctx context.Context, src querier, table string) (map[string]bool, error) {
	rows, err := src.QueryContext(ctx, `SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out[name] = true
	}
	return out, rows.Err()
}

// orderByForeignKeys sorts tables so each comes after the tables it refers
// to, alphabetically where the keys leave a choice.
func orderByForeignKeys(tables []string, fks []foreignKey) []string {
	parents := map[string]map[string]bool{}
	for _, fk := range fks {
		if fk.table == fk.refTable {
			continue
		}
		if parents[fk.table] == nil {
			parents[fk.table] = map[string]bool{}
		}
		parents[fk.table][fk.refTable] = true
	}
	remaining := slices.Sorted(slices.Values(tables))
	placed := map[string]bool{}
	var out []string
	for len(remaining) > 0 {
		progress := false
		for i := 0; i < len(remaining); i++ {
			ready := true
			for p := range parents[remaining[i]] {
				if !placed[p] && slices.Contains(remaining, p) {
					ready = false
					break
				}
			}
			if ready {
				placed[remaining[i]] = true
				out = append(out, remaining[i])
				remaining = slices.Delete(remaining, i, i+1)
				progress = true
				break
			}
		}
		if !progress { // a cycle: keep the rest in name order
			return append(out, remaining...)
		}
	}
	return out
}

func sqliteIdent(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }

// selectAll reads every row of t from the SQLite database.
func (t copyTable) selectAll(ctx context.Context, src querier) (*sql.Rows, error) {
	names := make([]string, len(t.columns))
	for i, c := range t.columns {
		names[i] = sqliteIdent(c.name)
	}
	q := "SELECT " + strings.Join(names, ", ") + " FROM " + sqliteIdent(t.name)
	if t.hasID {
		q += ` ORDER BY "id"`
	}
	return src.QueryContext(ctx, q)
}

// check reads every row of the source and reports what can't be copied.
func (p *copyPlan) check(ctx context.Context, src querier) error {
	problems := &CopyCheckError{}
	for _, t := range p.tables {
		if err := p.checkTable(ctx, src, t, problems); err != nil {
			return err
		}
	}
	for _, fk := range p.fks {
		if err := checkReferences(ctx, src, fk, problems); err != nil {
			return err
		}
	}
	if problems.Total > 0 {
		return problems
	}
	return nil
}

func (p *copyPlan) checkTable(ctx context.Context, src querier, t copyTable, problems *CopyCheckError) error {
	rows, err := t.selectAll(ctx, src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", t.name, err)
	}
	defer func() { _ = rows.Close() }()
	values := make([]any, len(t.columns))
	ptrs := make([]any, len(values))
	for i := range values {
		ptrs[i] = &values[i]
	}
	n := 0
	for rows.Next() {
		n++
		if err := rows.Scan(ptrs...); err != nil {
			return fmt.Errorf("reading %s: %w", t.name, err)
		}
		for i, c := range t.columns {
			v, err := convertValue(values[i], c.kind)
			if err != nil {
				problems.add(CopyProblem{t.name, c.name, t.rowLabel(values, n), err.Error()})
				continue
			}
			s, ok := v.(string)
			if !ok {
				continue
			}
			if c.maxChars > 0 {
				if l := utf8.RuneCountInString(s); int64(l) > c.maxChars {
					problems.add(CopyProblem{t.name, c.name, t.rowLabel(values, n), fmt.Sprintf("%d characters, the column holds %d", l, c.maxChars)})
				}
			}
			if p.postgres && strings.ContainsRune(s, 0) {
				problems.add(CopyProblem{t.name, c.name, t.rowLabel(values, n), "contains a NUL character, which PostgreSQL can't store"})
			}
			if !utf8.ValidString(s) {
				problems.add(CopyProblem{t.name, c.name, t.rowLabel(values, n), "is not valid UTF-8"})
			}
		}
	}
	return rows.Err()
}

// rowLabel names a row in a problem: by its id, or its position.
func (t copyTable) rowLabel(values []any, n int) string {
	if t.hasID {
		for i, c := range t.columns {
			if c.name == "id" {
				return fmt.Sprintf("%v", values[i])
			}
		}
	}
	return fmt.Sprintf("row %d", n)
}

// checkReferences reports rows of fk.table whose reference points at a row
// that doesn't exist, which the target's foreign key would refuse.
func checkReferences(ctx context.Context, src querier, fk foreignKey, problems *CopyCheckError) error {
	rows, err := src.QueryContext(ctx, fmt.Sprintf(`SELECT c.%[2]s FROM %[1]s c
		WHERE c.%[2]s IS NOT NULL AND NOT EXISTS (SELECT 1 FROM %[3]s p WHERE p.%[4]s = c.%[2]s)`,
		sqliteIdent(fk.table), sqliteIdent(fk.column), sqliteIdent(fk.refTable), sqliteIdent(fk.refColumn)))
	if err != nil {
		return fmt.Errorf("checking %s.%s: %w", fk.table, fk.column, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var ref any
		if err := rows.Scan(&ref); err != nil {
			return err
		}
		problems.add(CopyProblem{Table: fk.table, Column: fk.column, Row: fmt.Sprintf("a row referring to %v", ref),
			Problem: fmt.Sprintf("refers to %s %v, which doesn't exist", fk.refTable, ref)})
	}
	return rows.Err()
}

// sqliteTimeLayouts are the forms times take in rackd's SQLite databases:
// written by the Go driver, by CURRENT_TIMESTAMP, or as RFC 3339 text.
var sqliteTimeLayouts = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02",
}

// convertValue turns a value read from SQLite into one for a target column
// of the given kind.
func convertValue(v any, kind int) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch kind {
	case kindBool:
		switch x := v.(type) {
		case bool:
			return x, nil
		case int64:
			return x != 0, nil
		case string:
			if b, err := strconv.ParseBool(x); err == nil {
				return b, nil
			}
		}
	case kindTime:
		switch x := v.(type) {
		case time.Time:
			return x.UTC().Truncate(time.Microsecond), nil
		case string:
			s, _, _ := strings.Cut(x, " m=") // time.Time.String's monotonic reading
			for _, layout := range sqliteTimeLayouts {
				if t, err := time.Parse(layout, s); err == nil {
					return t.UTC().Truncate(time.Microsecond), nil
				}
			}
			return nil, fmt.Errorf("can't read %q as a time", x)
		case int64:
			return time.Unix(x, 0).UTC(), nil
		}
	case kindInt:
		switch x := v.(type) {
		case int64:
			return x, nil
		case float64:
			if x == float64(int64(x)) {
				return int64(x), nil
			}
		case string:
			if n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64); err == nil {
				return n, nil
			}
		case bool:
			if x {
				return int64(1), nil
			}
			return int64(0), nil
		}
	case kindFloat:
		switch x := v.(type) {
		case float64:
			return x, nil
		case int64:
			return float64(x), nil
		case string:
			if f, err := strconv.ParseFloat(strings.TrimSpace(x), 64); err == nil {
				return f, nil
			}
		}
	case kindBytes:
		switch x := v.(type) {
		case []byte:
			return x, nil
		case string:
			return []byte(x), nil
		}
	default:
		switch x := v.(type) {
		case string:
			return x, nil
		case []byte:
			return string(x), nil
		case int64:
			return strconv.FormatInt(x, 10), nil
		case float64:
			return strconv.FormatFloat(x, 'g', -1, 64), nil
		case bool:
			return strconv.FormatBool(x), nil
		case time.Time:
			return x.UTC().Format(time.RFC3339Nano), nil
		}
	}
	return nil, fmt.Errorf("can't store %T %v here", v, v)
}

// Inserts are sent in batches of up to copyBatchRows rows, copyBatchParams
// parameters (PostgreSQL allows 65535) or about copyBatchBytes of data
// (MySQL's max_allowed_packet defaults to 64 MB).
const (
	copyBatchRows   = 500
	copyBatchParams = 30000
	copyBatchBytes  = 4 << 20
)

// write replaces the target's rows with the source's in one transaction.
func (p *copyPlan) write(ctx context.Context, src querier, dst *sql.DB, progress func(string, int)) ([]CopiedTable, error) {
	tx, err := dst.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	// The target's own rows (seeded by its migrations) go first, children
	// before their parents.
	for i := len(p.tables) - 1; i >= 0; i-- {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+p.dst.QuoteIdentifier(p.tables[i].name)); err != nil {
			return nil, fmt.Errorf("clearing %s in the target: %w", p.tables[i].name, err)
		}
	}

	var copied []CopiedTable
	for _, t := range p.tables {
		n, err := p.copyTable(ctx, src, tx, t)
		if err != nil {
			return nil, err
		}
		var got int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+p.dst.QuoteIdentifier(t.name)).Scan(&got); err != nil {
			return nil, err
		}
		if got != n {
			return nil, fmt.Errorf("%s: copied %d rows but the target has %d; nothing was written", t.name, n, got)
		}
		copied = append(copied, CopiedTable{Name: t.name, Rows: n})
		if progress != nil {
			progress(t.name, n)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing the copy: %w", err)
	}
	return copied, nil
}

func (p *copyPlan) copyTable(ctx context.Context, src querier, tx *sql.Tx, t copyTable) (int, error) {
	rows, err := t.selectAll(ctx, src)
	if err != nil {
		return 0, fmt.Errorf("reading %s: %w", t.name, err)
	}
	defer func() { _ = rows.Close() }()

	quoted := make([]string, len(t.columns))
	for i, c := range t.columns {
		quoted[i] = p.dst.QuoteIdentifier(c.name)
	}
	prefix := "INSERT INTO " + p.dst.QuoteIdentifier(t.name) + " (" + strings.Join(quoted, ", ") + ") VALUES "
	tuple := "(" + strings.TrimSuffix(strings.Repeat("?, ", len(t.columns)), ", ") + ")"
	maxRows := min(copyBatchRows, copyBatchParams/len(t.columns))

	var args []any
	batchRows, batchBytes, total := 0, 0, 0
	flush := func() error {
		if batchRows == 0 {
			return nil
		}
		q := prefix + strings.TrimSuffix(strings.Repeat(tuple+", ", batchRows), ", ")
		if _, err := tx.ExecContext(ctx, p.dst.TranslateSQL(q), args...); err != nil {
			return fmt.Errorf("writing %s: %w", t.name, err)
		}
		args, batchRows, batchBytes = args[:0], 0, 0
		return nil
	}

	values := make([]any, len(t.columns))
	ptrs := make([]any, len(values))
	for i := range values {
		ptrs[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return 0, fmt.Errorf("reading %s: %w", t.name, err)
		}
		for i, c := range t.columns {
			v, err := convertValue(values[i], c.kind)
			if err != nil { // check found these; the data changed since
				return 0, fmt.Errorf("%s.%s: %w", t.name, c.name, err)
			}
			switch x := v.(type) {
			case string:
				batchBytes += len(x)
			case []byte:
				batchBytes += len(x)
			}
			args = append(args, v)
		}
		batchRows++
		total++
		if batchRows >= maxRows || batchBytes >= copyBatchBytes {
			if err := flush(); err != nil {
				return 0, err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("reading %s: %w", t.name, err)
	}
	return total, flush()
}
