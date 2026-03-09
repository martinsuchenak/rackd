package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/paularlott/cli"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "backup",
		Usage: "Backup the rackd database",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "data-dir", Usage: "Data directory (default: from config)", DefaultValue: "./data"},
			&cli.StringFlag{Name: "output", Usage: "Output file path (default: rackd-backup-<timestamp>.db)"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := config.Load()
			dataDir := cmd.GetString("data-dir")
			if dataDir == "" {
				dataDir = cfg.DataDir
			}

			srcPath := filepath.Join(dataDir, "rackd.db")
			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				return fmt.Errorf("database not found at %s", srcPath)
			}

			dstPath := cmd.GetString("output")
			if dstPath == "" {
				dstPath = fmt.Sprintf("rackd-backup-%s.db", time.Now().Format("20060102-150405"))
			}

			src, err := os.Open(srcPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer src.Close()

			dst, err := os.Create(dstPath)
			if err != nil {
				return fmt.Errorf("failed to create backup file: %w", err)
			}
			defer dst.Close()

			n, err := io.Copy(dst, src)
			if err != nil {
				os.Remove(dstPath)
				return fmt.Errorf("backup failed: %w", err)
			}

			// Also copy WAL and SHM files if they exist
			for _, suffix := range []string{"-wal", "-shm"} {
				walSrc := srcPath + suffix
				if _, err := os.Stat(walSrc); err == nil {
					walDst := dstPath + suffix
					if err := copyFile(walSrc, walDst); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to copy %s: %v\n", suffix, err)
					}
				}
			}

			fmt.Printf("Backup created: %s (%.1f MB)\n", dstPath, float64(n)/1024/1024)
			return nil
		},
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
