package main

import (
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

func TestCapabilityOperationsDeclareAuthorization(t *testing.T) {
	t.Parallel()
	doc := buildOpenAPI("test").OpenAPI()
	selfService := map[string]bool{
		"get-account":            true,
		"update-account":         true,
		"rotate-account-api-key": true,
		"revoke-account-api-key": true,
	}

	for path, item := range doc.Paths {
		for _, operation := range pathOperations(item) {
			if operation == nil || len(operation.Security) == 0 || selfService[operation.OperationID] {
				continue
			}
			if operation.Extensions["x-authz"] == nil {
				t.Errorf("%s %s (%s) has no x-authz requirement", operation.Method, path, operation.OperationID)
			}
			if operation.Responses["403"] == nil {
				t.Errorf("%s %s (%s) has no 403 response", operation.Method, path, operation.OperationID)
			}
		}
	}
}

func pathOperations(item *huma.PathItem) []*huma.Operation {
	return []*huma.Operation{
		item.Get,
		item.Put,
		item.Post,
		item.Delete,
		item.Options,
		item.Head,
		item.Patch,
		item.Trace,
	}
}
