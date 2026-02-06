package user

import (
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
		Name:  "user",
		Usage: "Manage users",
		Commands: []*cli.Command{
			ListCommand(),
			CreateCommand(),
			UpdateCommand(),
			DeleteCommand(),
			ChangePasswordCommand(),
		},
	}
}

func ListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List users",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "username",
				Usage: "Filter by username",
			},
			&cli.StringFlag{
				Name:  "email",
				Usage: "Filter by email",
			},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			resp, err := c.DoRequest("GET", "/api/users", nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var users []model.UserResponse
			if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tUSERNAME\tEMAIL\tNAME\tADMIN\tACTIVE\tCREATED")
			for _, user := range users {
				admin := "no"
				if user.IsAdmin {
					admin = "yes"
				}
				active := "no"
				if user.IsActive {
					active = "yes"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					user.ID, user.Username, user.Email,
					user.FullName, admin, active,
					user.CreatedAt.Format("2006-01-02 15:04"))
			}
			w.Flush()

			return nil
		},
	}
}

func CreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new user",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "username",
				Usage:    "Username",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "email",
				Usage:    "Email address",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "full-name",
				Usage: "Full name",
			},
			&cli.BoolFlag{
				Name:  "admin",
				Usage: "Make user an admin",
			},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			username := cmd.GetString("username")
			email := cmd.GetString("email")

			fmt.Printf("Enter password for %s: ", username)
			var password1, password2 string
			fmt.Scanln(&password1)
			fmt.Printf("Confirm password: ")
			fmt.Scanln(&password2)

			if password1 != password2 {
				return fmt.Errorf("passwords do not match")
			}

			if len(password1) < 8 {
				return fmt.Errorf("password must be at least 8 characters")
			}

			req := map[string]interface{}{
				"username":  username,
				"email":     email,
				"password":  password1,
				"full_name": cmd.GetString("full-name"),
				"is_admin":  cmd.GetBool("admin"),
			}

			resp, err := c.DoRequest("POST", "/api/users", req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			var user model.UserResponse
			if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			fmt.Printf("User created successfully!\n\n")
			fmt.Printf("ID:       %s\n", user.ID)
			fmt.Printf("Username: %s\n", user.Username)
			fmt.Printf("Email:    %s\n", user.Email)
			fmt.Printf("Admin:    %t\n", user.IsAdmin)

			return nil
		},
	}
}

func UpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a user",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "id",
				Usage:    "User ID",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "email",
				Usage: "Email address",
			},
			&cli.StringFlag{
				Name:  "full-name",
				Usage: "Full name",
			},
			&cli.BoolFlag{
				Name:  "active",
				Usage: "Set user active status",
			},
			&cli.BoolFlag{
				Name:  "inactive",
				Usage: "Set user inactive",
			},
			&cli.BoolFlag{
				Name:  "admin",
				Usage: "Make user an admin",
			},
			&cli.BoolFlag{
				Name:  "not-admin",
				Usage: "Remove admin status",
			},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			req := make(map[string]interface{})
			id := cmd.GetString("id")

			if email := cmd.GetString("email"); email != "" {
				req["email"] = email
			}
			if fullName := cmd.GetString("full-name"); fullName != "" {
				req["full_name"] = fullName
			}
			if cmd.GetBool("active") {
				active := true
				req["is_active"] = active
			}
			if cmd.GetBool("inactive") {
				inactive := false
				req["is_active"] = inactive
			}
			if cmd.GetBool("admin") {
				admin := true
				req["is_admin"] = admin
			}
			if cmd.GetBool("not-admin") {
				notAdmin := false
				req["is_admin"] = notAdmin
			}

			resp, err := c.DoRequest("PUT", fmt.Sprintf("/api/users/%s", id), req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var user model.UserResponse
			if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			fmt.Printf("User updated successfully!\n")
			fmt.Printf("ID:       %s\n", user.ID)
			fmt.Printf("Username: %s\n", user.Username)
			fmt.Printf("Email:    %s\n", user.Email)

			return nil
		},
	}
}

func DeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a user",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "User ID", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")
			resp, err := c.DoRequest("DELETE", fmt.Sprintf("/api/users/%s", id), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				return client.HandleError(resp)
			}

			fmt.Printf("User %s deleted successfully\n", id)
			return nil
		},
	}
}

func ChangePasswordCommand() *cli.Command {
	return &cli.Command{
		Name:  "password",
		Usage: "Change a user's password",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "id",
				Usage:    "User ID",
				Required: true,
			},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")

			fmt.Printf("Enter old password: ")
			var oldPassword string
			fmt.Scanln(&oldPassword)

			fmt.Printf("Enter new password: ")
			var newPassword1 string
			fmt.Scanln(&newPassword1)

			fmt.Printf("Confirm new password: ")
			var newPassword2 string
			fmt.Scanln(&newPassword2)

			if newPassword1 != newPassword2 {
				return fmt.Errorf("new passwords do not match")
			}

			if len(newPassword1) < 8 {
				return fmt.Errorf("password must be at least 8 characters")
			}

			if oldPassword == newPassword1 {
				return fmt.Errorf("new password must be different from old password")
			}

			req := map[string]interface{}{
				"old_password": oldPassword,
				"new_password": newPassword1,
			}

			resp, err := c.DoRequest("POST", fmt.Sprintf("/api/users/%s/password", id), req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNoContent {
				return client.HandleError(resp)
			}

			fmt.Printf("Password changed successfully!\n")

			return nil
		},
	}
}
