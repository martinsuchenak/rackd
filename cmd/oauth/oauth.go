package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "oauth",
		Usage: "Manage OAuth clients",
		Commands: []*cli.Command{
			listCommand(),
			deleteCommand(),
		},
	}
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List registered OAuth clients",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			c := client.NewClient(client.LoadConfig())
			resp, err := c.DoRequest("GET", "/api/oauth/clients", nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var clients []map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&clients)

			if cmd.GetString("output") == "json" {
				client.PrintJSON(clients)
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "CLIENT ID\tNAME\tGRANT TYPES\tCONFIDENTIAL\tCREATED")
			for _, cl := range clients {
				grants := ""
				if g, ok := cl["grant_types"].([]interface{}); ok {
					for i, v := range g {
						if i > 0 {
							grants += ", "
						}
						grants += fmt.Sprint(v)
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n",
					client.GetString(cl, "client_id"),
					client.GetString(cl, "client_name"),
					grants,
					cl["is_confidential"],
					client.GetString(cl, "created_at"))
			}
			w.Flush()
			return nil
		},
	}
}

func deleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete an OAuth client",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "OAuth client ID", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			c := client.NewClient(client.LoadConfig())
			resp, err := c.DoRequest("DELETE", "/api/oauth/clients/"+cmd.GetString("id"), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				return client.HandleError(resp)
			}
			fmt.Printf("OAuth client %s deleted\n", cmd.GetString("id"))
			return nil
		},
	}
}
