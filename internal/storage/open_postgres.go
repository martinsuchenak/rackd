package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/stdlib"
)

// openPostgresPool opens a PostgreSQL pool configured the way storage
// expects. Timestamps are read back in UTC, as they are written and as the
// other backends return them; pgx would otherwise convert them to the
// server's local time zone.
func openPostgresPool(cfg *pgx.ConnConfig) *sql.DB {
	return stdlib.OpenDB(*cfg, stdlib.OptionAfterConnect(func(_ context.Context, c *pgx.Conn) error {
		for _, t := range []struct {
			name string
			oid  uint32
		}{{"timestamptz", pgtype.TimestamptzOID}, {"timestamp", pgtype.TimestampOID}} {
			var codec pgtype.Codec = &pgtype.TimestamptzCodec{ScanLocation: time.UTC}
			if t.oid == pgtype.TimestampOID {
				codec = &pgtype.TimestampCodec{ScanLocation: time.UTC}
			}
			c.TypeMap().RegisterType(&pgtype.Type{Name: t.name, OID: t.oid, Codec: codec})
		}
		return nil
	}))
}
