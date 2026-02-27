package customfield

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func DeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a custom field definition",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Custom field ID", Required: true},
			&cli.BoolFlag{Name: "force", Usage: "Skip confirmation"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")

			// Get the field first to show what will be deleted
			resp, err := c.DoRequest("GET", "/api/custom-fields/"+id, nil)
			if err != nil {
				return err
			}

			if resp.StatusCode == http.StatusNotFound {
				resp.Body.Close()
				return fmt.Errorf("custom field not found: %s", id)
			}

			var field map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&field); err != nil {
				resp.Body.Close()
				return err
			}
			resp.Body.Close()

			// Confirm deletion unless --force is set
			if !cmd.GetBool("force") {
				fmt.Printf("Are you sure you want to delete custom field '%s' (key: %s)? [y/N]: ",
					field["name"], field["key"])
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled")
					return nil
				}
			}

			resp, err = c.DoRequest("DELETE", "/api/custom-fields/"+id, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			fmt.Printf("Custom field '%s' deleted successfully\n", field["name"])
			return nil
		},
	}
}
