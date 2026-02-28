package nat

import (
	"context"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func DeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a NAT mapping",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "NAT mapping ID", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")
			resp, err := c.DoRequest("DELETE", "/api/nat/"+id, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				return client.HandleError(resp)
			}

			return nil
		},
	}
}
