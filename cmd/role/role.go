package role

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/paularlott/cli"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/martinsuchenak/rackd/internal/model"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "role",
		Usage: "Manage roles and permissions",
		Commands: []*cli.Command{
			ListRolesCommand(),
			ListPermissionsCommand(),
			CreateRoleCommand(),
			DeleteRoleCommand(),
			AssignRoleCommand(),
			RevokeRoleCommand(),
		},
	}
}

func ListRolesCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List roles",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "name",
				Usage: "Filter by name",
			},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			queryParams := ""
			if name := cmd.GetString("name"); name != "" {
				queryParams = fmt.Sprintf("?name=%s", name)
			}

			resp, err := c.DoRequest("GET", "/api/roles"+queryParams, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var roles []model.RoleResponse
			if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tSYSTEM\tCREATED\tUPDATED")
			for _, role := range roles {
				system := "no"
				if role.IsSystem {
					system = "yes"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					role.ID, role.Name, role.Description, system,
					role.CreatedAt.Format("2006-01-02 15:04"),
					role.UpdatedAt.Format("2006-01-02 15:04"))
			}
			w.Flush()

			return nil
		},
	}
}

func ListPermissionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "permissions",
		Usage: "List permissions",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "resource",
				Usage: "Filter by resource",
			},
			&cli.StringFlag{
				Name:  "action",
				Usage: "Filter by action",
			},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			queryParams := ""
			if resource := cmd.GetString("resource"); resource != "" {
				if queryParams == "" {
					queryParams = "?"
				} else {
					queryParams += "&"
				}
				queryParams += fmt.Sprintf("resource=%s", resource)
			}
			if action := cmd.GetString("action"); action != "" {
				if queryParams == "" {
					queryParams = "?"
				} else {
					queryParams += "&"
				}
				queryParams += fmt.Sprintf("action=%s", action)
			}

			resp, err := c.DoRequest("GET", "/api/permissions"+queryParams, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var permissions []model.Permission
			if err := json.NewDecoder(resp.Body).Decode(&permissions); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tRESOURCE\tACTION\tCREATED")
			for _, perm := range permissions {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					perm.ID, perm.Name, perm.Resource, perm.Action,
					perm.CreatedAt.Format("2006-01-02 15:04"))
			}
			w.Flush()

			return nil
		},
	}
}

func CreateRoleCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new role",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Usage:    "Role name",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "description",
				Usage: "Role description",
			},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			req := model.CreateRoleRequest{
				Name:        cmd.GetString("name"),
				Description: cmd.GetString("description"),
			}

			body, err := json.Marshal(req)
			if err != nil {
				return fmt.Errorf("failed to marshal request: %w", err)
			}

			resp, err := c.DoRequest("POST", "/api/roles", bytes.NewReader(body))
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			var role model.RoleResponse
			if err := json.NewDecoder(resp.Body).Decode(&role); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			fmt.Printf("Role created: %s (%s)\n", role.Name, role.ID)
			return nil
		},
	}
}

func DeleteRoleCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a role",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "id",
				Usage:    "Role ID",
				Required: true,
			},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")
			resp, err := c.DoRequest("DELETE", "/api/roles/"+id, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				return client.HandleError(resp)
			}

			fmt.Println("Role deleted successfully")
			return nil
		},
	}
}

func AssignRoleCommand() *cli.Command {
	return &cli.Command{
		Name:  "assign",
		Usage: "Assign a role to a user",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "user-id",
				Usage:    "User ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "role-id",
				Usage:    "Role ID",
				Required: true,
			},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			req := model.GrantRoleRequest{
				UserID: cmd.GetString("user-id"),
				RoleID: cmd.GetString("role-id"),
			}

			body, err := json.Marshal(req)
			if err != nil {
				return fmt.Errorf("failed to marshal request: %w", err)
			}

			resp, err := c.DoRequest("POST", "/api/users/grant-role", bytes.NewReader(body))
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			fmt.Println("Role assigned successfully")
			return nil
		},
	}
}

func RevokeRoleCommand() *cli.Command {
	return &cli.Command{
		Name:  "revoke",
		Usage: "Revoke a role from a user",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "user-id",
				Usage:    "User ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "role-id",
				Usage:    "Role ID",
				Required: true,
			},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			req := model.RevokeRoleRequest{
				UserID: cmd.GetString("user-id"),
				RoleID: cmd.GetString("role-id"),
			}

			body, err := json.Marshal(req)
			if err != nil {
				return fmt.Errorf("failed to marshal request: %w", err)
			}

			resp, err := c.DoRequest("POST", "/api/users/revoke-role", bytes.NewReader(body))
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				return client.HandleError(resp)
			}

			fmt.Println("Role revoked successfully")
			return nil
		},
	}
}
