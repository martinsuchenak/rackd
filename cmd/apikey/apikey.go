package apikey

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/paularlott/cli"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/model"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "apikey",
		Usage: "Manage API keys",
		Commands: []*cli.Command{
			ListCommand(),
			CreateCommand(),
			DeleteCommand(),
			GenerateCommand(),
		},
	}
}

func ListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List API keys",
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			resp, err := c.DoRequest("GET", "/api/keys", nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var keys []model.APIKeyResponse
			if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tCREATED\tLAST USED\tEXPIRES")
			for _, key := range keys {
				lastUsed := "never"
				if key.LastUsedAt != nil {
					lastUsed = key.LastUsedAt.Format("2006-01-02 15:04")
				}
				expires := "never"
				if key.ExpiresAt != nil {
					expires = key.ExpiresAt.Format("2006-01-02")
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					key.ID, key.Name, key.Description,
					key.CreatedAt.Format("2006-01-02 15:04"),
					lastUsed, expires)
			}
			w.Flush()

			return nil
		},
	}
}

func CreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new API key",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Usage:    "API key name",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "description",
				Usage: "API key description",
			},
			&cli.StringFlag{
				Name:  "expires",
				Usage: "Expiration date (YYYY-MM-DD)",
			},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			req := map[string]interface{}{
				"name":        cmd.GetString("name"),
				"description": cmd.GetString("description"),
			}

			if expires := cmd.GetString("expires"); expires != "" {
				t, err := time.Parse("2006-01-02", expires)
				if err != nil {
					return fmt.Errorf("invalid expiration date format (use YYYY-MM-DD): %w", err)
				}
				req["expires_at"] = t
			}

			resp, err := c.DoRequest("POST", "/api/keys", req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			var key model.APIKey
			if err := json.NewDecoder(resp.Body).Decode(&key); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			fmt.Printf("API Key created successfully!\n\n")
			fmt.Printf("ID:   %s\n", key.ID)
			fmt.Printf("Name: %s\n", key.Name)
			fmt.Printf("Key:  %s\n\n", key.Key)
			fmt.Printf("⚠️  Save this key securely - it will not be shown again!\n")

			return nil
		},
	}
}

func DeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete an API key",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "API key ID", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")
			resp, err := c.DoRequest("DELETE", fmt.Sprintf("/api/keys/%s", id), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				return client.HandleError(resp)
			}

			fmt.Printf("API key %s deleted successfully\n", id)
			return nil
		},
	}
}

func GenerateCommand() *cli.Command {
	return &cli.Command{
		Name:  "generate",
		Usage: "Generate a random API key (offline)",
		Run: func(ctx context.Context, cmd *cli.Command) error {
			key, err := auth.GenerateKey()
			if err != nil {
				return fmt.Errorf("failed to generate key: %w", err)
			}

			fmt.Printf("Generated API key: %s\n", key)
			fmt.Printf("\nThis is a random key that can be used with the 'create' command.\n")
			return nil
		},
	}
}
