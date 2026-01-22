package datacenter

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func DeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a datacenter",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Datacenter ID", Required: true},
			&cli.BoolFlag{Name: "force", Usage: "Skip confirmation"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			dcID := cmd.GetString("id")

			if !cmd.GetBool("force") {
				fmt.Printf("Are you sure you want to delete datacenter %s? [y/N]: ", dcID)
				reader := bufio.NewReader(os.Stdin)
				confirm, _ := reader.ReadString('\n')
				confirm = strings.TrimSpace(strings.ToLower(confirm))
				if confirm != "y" && confirm != "yes" {
					fmt.Println("Deletion cancelled")
					return nil
				}
			}

			resp, err := c.DoRequest("DELETE", "/api/datacenters/"+dcID, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
				return client.HandleError(resp)
			}

			fmt.Println("Datacenter deleted successfully")
			return nil
		},
	}
}
