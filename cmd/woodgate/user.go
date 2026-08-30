package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/woodleighschool/woodgate/internal/authorization"
	"github.com/woodleighschool/woodgate/internal/directory"
	"github.com/woodleighschool/woodgate/internal/postgres"
)

func userCommand() *cobra.Command {
	var databaseURL string
	cmd := &cobra.Command{Use: "user", Short: "Manage users", Args: cobra.NoArgs}
	cmd.PersistentFlags().StringVar(&databaseURL, "database-url", "", "Postgres URL (defaults to WOODGATE_DATABASE_URL)")
	cmd.AddCommand(createUserCommand(&databaseURL), setUserPasswordCommand(&databaseURL), setUserAccessCommand(&databaseURL))
	return cmd
}

func createUserCommand(databaseURL *string) *cobra.Command {
	var email, name, password string
	cmd := &cobra.Command{Use: "create", Short: "Create an enabled owner", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		resolved, err := commandPassword(cmd, password)
		if err != nil {
			return err
		}
		return withUserServices(cmd.Context(), *databaseURL, func(users *directory.UserService, authz *authorization.Service) error {
			user, err := users.Create(cmd.Context(), directory.UserCreate{Email: email, Name: name, Password: resolved, AccessEnabled: true})
			if err != nil {
				return fmt.Errorf("create user: %w", err)
			}
			if err := authz.AssignOwner(cmd.Context(), user.ID); err != nil {
				return fmt.Errorf("assign owner: %w", err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "created owner %s (id %d)\n", user.Email, user.ID)
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
		return withUserServices(cmd.Context(), *databaseURL, func(users *directory.UserService, _ *authorization.Service) error {
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

func setUserAccessCommand(databaseURL *string) *cobra.Command {
	var email string
	var enabled bool
	cmd := &cobra.Command{Use: "set-access", Short: "Enable or disable sign-in", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return withUserServices(cmd.Context(), *databaseURL, func(users *directory.UserService, _ *authorization.Service) error {
			user, err := users.SetAccessEnabledByEmail(cmd.Context(), email, enabled)
			if err != nil {
				return fmt.Errorf("set access: %w", err)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "set access for %s to %s\n", user.Email, strconv.FormatBool(enabled))
			return err
		})
	}}
	cmd.Flags().StringVar(&email, "email", "", "User email")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Allow sign-in")
	_ = cmd.MarkFlagRequired("email")
	return cmd
}

func withUserServices(ctx context.Context, databaseURL string, action func(*directory.UserService, *authorization.Service) error) error {
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
	return action(directory.NewUserService(directory.NewStore(pool)), authorization.NewService(authorization.NewStore(pool)))
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
