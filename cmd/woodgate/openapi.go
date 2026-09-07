package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/woodleighschool/woodgate/internal/transport/http/httpapi"
)

func exportOpenAPI(args []string) error {
	flags := flag.NewFlagSet("openapi", flag.ContinueOnError)
	output := flags.String("output", "", "write the OpenAPI schema to a file")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("openapi accepts no positional arguments")
	}
	data, err := httpapi.OpenAPI().YAML()
	if err != nil {
		return fmt.Errorf("encode OpenAPI: %w", err)
	}
	if *output == "" {
		_, err = os.Stdout.Write(data)
	} else {
		err = os.WriteFile(*output, data, 0o600)
	}
	if err != nil {
		return fmt.Errorf("write OpenAPI: %w", err)
	}
	return nil
}
