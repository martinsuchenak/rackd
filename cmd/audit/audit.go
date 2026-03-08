package audit

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/paularlott/cli"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "audit",
		Usage: "Audit log management",
		Commands: []*cli.Command{
			listCommand(),
			exportCommand(),
		},
	}
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List audit logs",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "resource", Usage: "Filter by resource type"},
			&cli.StringFlag{Name: "resource-id", Usage: "Filter by resource ID"},
			&cli.StringFlag{Name: "action", Usage: "Filter by action"},
			&cli.IntFlag{Name: "limit", Usage: "Limit number of results", DefaultValue: 50},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			dataDir := cmd.GetString("data-dir")
			if dataDir == "" {
				dataDir = "./data"
			}

			store, err := storage.NewExtendedStorage(dataDir)
			if err != nil {
				return err
			}
			defer store.Close()

			svc := service.NewServices(store, nil, nil)

			filter := &model.AuditFilter{
				Resource:   cmd.GetString("resource"),
				ResourceID: cmd.GetString("resource-id"),
				Action:     cmd.GetString("action"),
				Pagination: model.Pagination{Limit: cmd.GetInt("limit")},
			}

			ctx = service.SystemContext(ctx, "cli")
			logs, err := svc.Audit.List(ctx, filter)
			if err != nil {
				return err
			}

			if len(logs) == 0 {
				fmt.Println("No audit logs found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TIMESTAMP\tACTION\tRESOURCE\tRESOURCE_ID\tUSER\tIP\tSTATUS")

			for _, log := range logs {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					log.Timestamp.Format(time.RFC3339),
					log.Action,
					log.Resource,
					log.ResourceID,
					log.Username,
					log.IPAddress,
					log.Status,
				)
			}

			return w.Flush()
		},
	}
}

func exportCommand() *cli.Command {
	return &cli.Command{
		Name:  "export",
		Usage: "Export audit logs",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "format", Usage: "Export format (json/csv)", DefaultValue: "json"},
			&cli.StringFlag{Name: "output", Usage: "Output file (default: stdout)"},
			&cli.StringFlag{Name: "resource", Usage: "Filter by resource type"},
			&cli.StringFlag{Name: "resource-id", Usage: "Filter by resource ID"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			dataDir := cmd.GetString("data-dir")
			if dataDir == "" {
				dataDir = "./data"
			}

			store, err := storage.NewExtendedStorage(dataDir)
			if err != nil {
				return err
			}
			defer store.Close()

			svc := service.NewServices(store, nil, nil)

			filter := &model.AuditFilter{
				Resource:   cmd.GetString("resource"),
				ResourceID: cmd.GetString("resource-id"),
			}

			ctx = service.SystemContext(ctx, "cli")
			format := cmd.GetString("format")

			data, err := svc.Audit.Export(ctx, filter, format)
			if err != nil {
				return err
			}

			output := cmd.GetString("output")
			if output == "" {
				fmt.Println(string(data))
			} else {
				if err := os.WriteFile(output, data, 0644); err != nil {
					return err
				}
				fmt.Printf("Exported %d audit logs to %s\n", len(data), output)
			}

			return nil
		},
	}
}
