package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/woodleighschool/woodgate/internal/directory"
	"github.com/woodleighschool/woodgate/internal/postgres"
	"github.com/woodleighschool/woodgate/internal/rbac"
)

func userCommand() *cobra.Command {
	var databaseURL string
	cmd := &cobra.Command{Use: "user", Short: "Manage users", Args: cobra.NoArgs}
	cmd.PersistentFlags().StringVar(&databaseURL, "database-url", "", "Postgres URL (defaults to WOODGATE_DATABASE_URL)")
	cmd.AddCommand(createUserCommand(&databaseURL), setUserPasswordCommand(&databaseURL), setUserRolesCommand(&databaseURL))
	return cmd
}

func createUserCommand(databaseURL *string) *cobra.Command {
	var email, name, password string
	cmd := &cobra.Command{Use: "create", Short: "Create an admin", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		resolved, err := commandPassword(cmd, password)
		if err != nil {
			return err
		}
		return withUserServices(cmd.Context(), *databaseURL, func(users *directory.UserService, roles *rbac.Store) error {
			roleIDs, err := roleIDsByKey(cmd.Context(), roles, []string{"admin"})
			if err != nil {
				return err
			}
			user, err := users.Create(cmd.Context(), directory.UserCreate{Email: email, Name: name, Password: resolved, RoleIDs: roleIDs})
			if err != nil {
				return fmt.Errorf("create user: %w", err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "created admin %s (id %d)\n", user.Email, user.ID)
			return err
		})
	}}
	cmd.Flags().StringVar(&email, "email", "", "User email")
	cmd.Flags().StringVar(&name, "name", "", "Display name")
	cmd.Flags().StringVar(&password, "password", "", "User password (prompts when omitted)")
	_ = cmd.MarkFlagRequired("email")
	return cmd
}

func setUserPasswordCommand(databaseURL *string) *cobra.Command {
	var email, password string
	cmd := &cobra.Command{Use: "set-password", Short: "Replace a local user's password", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		resolved, err := commandPassword(cmd, password)
		if err != nil {
			return err
		}
		return withUserServices(cmd.Context(), *databaseURL, func(users *directory.UserService, _ *rbac.Store) error {
			user, err := users.SetPasswordByEmail(cmd.Context(), email, resolved)
			if err != nil {
				return fmt.Errorf("set password: %w", err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "updated password for %s\n", user.Email)
			return err
		})
	}}
	cmd.Flags().StringVar(&email, "email", "", "User email")
	cmd.Flags().StringVar(&password, "password", "", "User password (prompts when omitted)")
	_ = cmd.MarkFlagRequired("email")
	return cmd
}

func setUserRolesCommand(databaseURL *string) *cobra.Command {
	var email string
	var keys []string
	cmd := &cobra.Command{Use: "set-roles", Short: "Replace a user's direct roles", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return withUserServices(cmd.Context(), *databaseURL, func(users *directory.UserService, roles *rbac.Store) error {
			roleIDs, err := roleIDsByKey(cmd.Context(), roles, keys)
			if err != nil {
				return err
			}
			user, err := users.SetRolesByEmail(cmd.Context(), email, roleIDs)
			if err != nil {
				return fmt.Errorf("set roles: %w", err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "set %d direct role(s) for %s\n", len(user.Roles), user.Email)
			return err
		})
	}}
	cmd.Flags().StringVar(&email, "email", "", "User email")
	cmd.Flags().StringSliceVar(&keys, "role", nil, "Role key (repeat for multiple; omit for no access)")
	_ = cmd.MarkFlagRequired("email")
	return cmd
}

func roleIDsByKey(ctx context.Context, roles *rbac.Store, keys []string) ([]int64, error) {
	assigned, err := roles.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	byKey := make(map[string]int64, len(assigned))
	for _, role := range assigned {
		byKey[role.Key] = role.ID
	}
	ids := make([]int64, 0, len(keys))
	for _, key := range keys {
		key = strings.ToLower(strings.TrimSpace(key))
		id, ok := byKey[key]
		if !ok {
			return nil, fmt.Errorf("role %q not found", key)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func withUserServices(ctx context.Context, databaseURL string, action func(*directory.UserService, *rbac.Store) error) error {
	databaseURL = strings.TrimSpace(databaseURL)
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("WOODGATE_DATABASE_URL"))
	}
	if databaseURL == "" {
		return errors.New("database URL is required: set WOODGATE_DATABASE_URL or --database-url")
	}
	pool, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()
	return action(directory.NewUserService(directory.NewStore(pool)), rbac.NewStore(pool))
}

func commandPassword(cmd *cobra.Command, value string) (string, error) {
	if cmd.Flags().Changed("password") {
		return value, nil
	}
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("password is required: pass --password or use an interactive terminal")
	}
	if _, err := fmt.Fprint(cmd.ErrOrStderr(), "Password: "); err != nil {
		return "", err
	}
	password, err := term.ReadPassword(fd)
	_, newlineErr := fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	if newlineErr != nil {
		return "", newlineErr
	}
	return string(password), nil
}
