package main

import (
	"fmt"
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/spf13/cobra"

	"github.com/woodleighschool/woodgate/internal/account"
	"github.com/woodleighschool/woodgate/internal/api"
	"github.com/woodleighschool/woodgate/internal/buildinfo"
	checkinapi "github.com/woodleighschool/woodgate/internal/checkin/httpapi"
	directoryapi "github.com/woodleighschool/woodgate/internal/directory/httpapi"
	authzapi "github.com/woodleighschool/woodgate/internal/rbac/httpapi"
	stationapi "github.com/woodleighschool/woodgate/internal/station/httpapi"
)

func openAPICommand() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use: "openapi", Short: "Print the OpenAPI document for the API", Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			payload, err := buildOpenAPI(buildinfo.Version).OpenAPI().YAML()
			if err != nil {
				return fmt.Errorf("encode openapi: %w", err)
			}
			if len(payload) == 0 || payload[len(payload)-1] != '\n' {
				payload = append(payload, '\n')
			}
			if output == "" || output == "-" {
				_, err = os.Stdout.Write(payload)
				return err
			}
			return os.WriteFile(output, payload, 0o600)
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "write OpenAPI YAML to this path (default stdout)")
	return cmd
}

func buildOpenAPI(version string) huma.API {
	schema, routes := api.NewSchema(version)
	account.RegisterOpenAPI(routes)
	directoryapi.RegisterOpenAPI(routes)
	authzapi.RegisterOpenAPI(routes)
	checkinapi.RegisterOpenAPI(routes)
	stationapi.RegisterOpenAPI(routes)
	return schema
}
