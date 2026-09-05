package api

import "github.com/danielgtaylor/huma/v2"

// OpenAPI resource tags are shared by capability operations and API metadata.
const (
	TagAccount         = "Account"
	TagAuthorization   = "Authorization"
	TagCheckins        = "Check-ins"
	TagDirectoryGroups = "Directory groups"
	TagDirectorySync   = "Directory sync"
	TagDirectoryUsers  = "Directory users"
	TagLocations       = "Locations"
	TagSession         = "Session"
	TagStations        = "Stations"
)

type openAPITagGroup struct {
	Name string   `json:"name" yaml:"name"`
	Tags []string `json:"tags" yaml:"tags"`
}

// configureOpenAPI declares the resource hierarchy used by API documentation.
func configureOpenAPI(doc *huma.OpenAPI) {
	doc.Tags = []*huma.Tag{
		resourceTag(TagAccount, "Account"),
		resourceTag(TagAuthorization, "Roles and permissions"),
		resourceTag(TagCheckins, "Check-ins"),
		resourceTag(TagDirectoryGroups, "Groups"),
		resourceTag(TagDirectorySync, "Sync"),
		resourceTag(TagDirectoryUsers, "Users"),
		resourceTag(TagLocations, "Locations"),
		resourceTag(TagSession, "Session"),
		resourceTag(TagStations, "Stations"),
	}
	doc.Extensions = map[string]any{
		"x-tagGroups": []openAPITagGroup{
			{Name: "Account", Tags: []string{TagAccount}},
			{Name: "Directory", Tags: []string{TagDirectorySync, TagDirectoryGroups, TagDirectoryUsers}},
			{Name: "Check-in", Tags: []string{TagCheckins, TagLocations, TagStations}},
			{Name: "Authorization", Tags: []string{TagAuthorization}},
			{Name: "Session", Tags: []string{TagSession}},
		},
	}
}

func resourceTag(name string, displayName string) *huma.Tag {
	return &huma.Tag{
		Name:       name,
		Extensions: map[string]any{"x-displayName": displayName},
	}
}
