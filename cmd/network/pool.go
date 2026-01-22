package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/paularlott/cli"
)

func PoolCommand() *cli.Command {
	return &cli.Command{
		Name:  "pool",
		Usage: "Network pool management",
		Commands: []*cli.Command{
			poolListCommand(),
			poolAddCommand(),
		},
	}
}

func poolListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List pools for a network",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "network", Usage: "Network ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			networkID := cmd.GetString("network")

			resp, err := c.DoRequest("GET", "/api/networks/"+networkID+"/pools", nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var pools []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&pools); err != nil {
				return err
			}

			if cmd.GetString("output") == "json" {
				client.PrintJSON(pools)
			} else {
				printPoolTable(pools)
			}
			return nil
		},
	}
}

func poolAddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add a pool to a network",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "network", Usage: "Network ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Pool name", Required: true},
			&cli.StringFlag{Name: "start", Usage: "Start IP", Required: true},
			&cli.StringFlag{Name: "end", Usage: "End IP", Required: true},
			&cli.StringFlag{Name: "description", Usage: "Pool description"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			networkID := cmd.GetString("network")

			pool := model.NetworkPool{
				NetworkID:   networkID,
				Name:        cmd.GetString("name"),
				StartIP:     cmd.GetString("start"),
				EndIP:       cmd.GetString("end"),
				Description: cmd.GetString("description"),
			}

			resp, err := c.DoRequest("POST", "/api/networks/"+networkID+"/pools", pool)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			var created map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
				return err
			}

			if cmd.GetString("output") == "json" {
				client.PrintJSON(created)
			} else {
				fmt.Printf("Pool created successfully\n")
				fmt.Printf("ID: %s\n", created["id"])
				fmt.Printf("Name: %s\n", created["name"])
			}
			return nil
		},
	}
}

func printPoolTable(pools []map[string]interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTART IP\tEND IP")
	for _, p := range pools {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			getString(p, "id"),
			getString(p, "name"),
			getString(p, "start_ip"),
			getString(p, "end_ip"))
	}
	w.Flush()
}
