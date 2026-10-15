package storage

import (
	"context"
	"testing"
)

// Every pooled connection runs with WAL and synchronous=NORMAL; FULL made
// each commit wait for the disk, which was slow on network storage.
func TestConnectionsUseWALAndNormalSync(t *testing.T) {
	s := newTestStorage(t)
	defer func() { _ = s.Close() }()
	ctx := context.Background()

	// Hold several connections at once so the check covers more than one.
	conns := make([]interface{ Close() error }, 0, 3)
	for i := 0; i < 3; i++ {
		c, err := s.db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		conns = append(conns, c)
		var mode string
		var sync int
		if err := c.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
			t.Fatal(err)
		}
		if err := c.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&sync); err != nil {
			t.Fatal(err)
		}
		if mode != "wal" || sync != 1 {
			t.Errorf("connection %d: journal_mode=%s synchronous=%d, want wal and 1 (NORMAL)", i, mode, sync)
		}
	}
	for _, c := range conns {
		_ = c.Close()
	}
}
