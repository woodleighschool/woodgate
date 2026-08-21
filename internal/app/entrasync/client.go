package entrasync

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/google/uuid"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	graphcore "github.com/microsoftgraph/msgraph-sdk-go-core"
	graphgroups "github.com/microsoftgraph/msgraph-sdk-go/groups"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	graphusers "github.com/microsoftgraph/msgraph-sdk-go/users"
)

const graphPageSize int32 = 999

var graphScopes = []string{"https://graph.microsoft.com/.default"}

// Config holds the application credentials used for Entra synchronization.
type Config struct {
	TenantID     string
	ClientID     string
	ClientSecret string
}

// User is an enabled Entra member user.
type User struct {
	ID          uuid.UUID
	UPN         string
	DisplayName string
	Department  string
}

// Group is a security-enabled Entra group.
type Group struct {
	ID          uuid.UUID
	DisplayName string
	Description string
}

// Snapshot is the complete Entra-owned directory state for one sync pass.
type Snapshot struct {
	Users   []User
	Groups  []Group
	Members map[uuid.UUID][]uuid.UUID
}

// Client fetches Entra users, groups, and transitive memberships from Graph.
type Client struct {
	graph *msgraphsdk.GraphServiceClient
}

// NewClient returns a Graph SDK client authenticated with an Entra application.
func NewClient(cfg Config) (*Client, error) {
	credential, err := azidentity.NewClientSecretCredential(
		cfg.TenantID,
		cfg.ClientID,
		cfg.ClientSecret,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create Entra credential: %w", err)
	}

	graph, err := msgraphsdk.NewGraphServiceClientWithCredentials(credential, graphScopes)
	if err != nil {
		return nil, fmt.Errorf("create Microsoft Graph client: %w", err)
	}
	return newClient(graph), nil
}

func newClient(graph *msgraphsdk.GraphServiceClient) *Client {
	return &Client{graph: graph}
}

// Fetch returns a complete snapshot of enabled member users, security-enabled
// groups, and each user's transitive membership in those groups.
func (client *Client) Fetch(ctx context.Context) (*Snapshot, error) {
	users, err := client.fetchUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch users: %w", err)
	}

	groups, err := client.fetchGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch groups: %w", err)
	}

	members, err := client.fetchMemberships(ctx, users, groups)
	if err != nil {
		return nil, fmt.Errorf("fetch memberships: %w", err)
	}

	return &Snapshot{Users: users, Groups: groups, Members: members}, nil
}

func (client *Client) fetchUsers(ctx context.Context) ([]User, error) {
	filter := "accountEnabled eq true and userType eq 'Member'"
	count := true
	headers := advancedQueryHeaders()
	page, err := client.graph.Users().Get(ctx, &graphusers.UsersRequestBuilderGetRequestConfiguration{
		Headers: headers,
		QueryParameters: &graphusers.UsersRequestBuilderGetQueryParameters{
			Count:  &count,
			Filter: &filter,
			Select: []string{"id", "userPrincipalName", "displayName", "department"},
			Top:    new(graphPageSize),
		},
	})
	if err != nil {
		return nil, err
	}

	iterator, err := graphcore.NewPageIterator[graphmodels.Userable](
		page,
		client.graph.GetAdapter(),
		graphmodels.CreateUserCollectionResponseFromDiscriminatorValue,
	)
	if err != nil {
		return nil, err
	}
	iterator.SetHeaders(headers)

	var users []User
	var parseErr error
	err = iterator.Iterate(ctx, func(graphUser graphmodels.Userable) bool {
		userID, err := parseGraphID(graphUser.GetId())
		if err != nil {
			parseErr = fmt.Errorf("user ID: %w", err)
			return false
		}
		users = append(users, User{
			ID:          userID,
			UPN:         stringValue(graphUser.GetUserPrincipalName()),
			DisplayName: stringValue(graphUser.GetDisplayName()),
			Department:  stringValue(graphUser.GetDepartment()),
		})
		return true
	})
	if err != nil {
		return nil, err
	}
	if parseErr != nil {
		return nil, parseErr
	}
	return users, nil
}

func (client *Client) fetchGroups(ctx context.Context) ([]Group, error) {
	filter := "securityEnabled eq true"
	count := true
	headers := advancedQueryHeaders()
	page, err := client.graph.Groups().Get(ctx, &graphgroups.GroupsRequestBuilderGetRequestConfiguration{
		Headers: headers,
		QueryParameters: &graphgroups.GroupsRequestBuilderGetQueryParameters{
			Count:  &count,
			Filter: &filter,
			Select: []string{"id", "displayName", "description"},
			Top:    new(graphPageSize),
		},
	})
	if err != nil {
		return nil, err
	}

	iterator, err := graphcore.NewPageIterator[graphmodels.Groupable](
		page,
		client.graph.GetAdapter(),
		graphmodels.CreateGroupCollectionResponseFromDiscriminatorValue,
	)
	if err != nil {
		return nil, err
	}
	iterator.SetHeaders(headers)

	var groups []Group
	var parseErr error
	err = iterator.Iterate(ctx, func(graphGroup graphmodels.Groupable) bool {
		groupID, err := parseGraphID(graphGroup.GetId())
		if err != nil {
			parseErr = fmt.Errorf("group ID: %w", err)
			return false
		}
		groups = append(groups, Group{
			ID:          groupID,
			DisplayName: stringValue(graphGroup.GetDisplayName()),
			Description: stringValue(graphGroup.GetDescription()),
		})
		return true
	})
	if err != nil {
		return nil, err
	}
	if parseErr != nil {
		return nil, parseErr
	}
	return groups, nil
}

func (client *Client) fetchMemberships(
	ctx context.Context,
	users []User,
	groups []Group,
) (map[uuid.UUID][]uuid.UUID, error) {
	members := make(map[uuid.UUID][]uuid.UUID, len(groups))
	knownGroups := make(map[uuid.UUID]struct{}, len(groups))
	for _, group := range groups {
		members[group.ID] = nil
		knownGroups[group.ID] = struct{}{}
	}

	pending := make([]membershipRequest, 0, len(users))
	for _, user := range users {
		pending = append(pending, membershipRequest{UserID: user.ID})
	}
	for len(pending) > 0 {
		followups, err := client.applyMembershipBatch(ctx, pending, knownGroups, members)
		if err != nil {
			return nil, err
		}
		pending = followups
	}
	return members, nil
}

func (client *Client) applyMembershipBatch(
	ctx context.Context,
	requests []membershipRequest,
	knownGroups map[uuid.UUID]struct{},
	members map[uuid.UUID][]uuid.UUID,
) ([]membershipRequest, error) {
	adapter := client.graph.GetAdapter()
	batch := graphcore.NewBatchRequestCollectionWithLimit(adapter, len(requests))
	requestsByID := make(map[string]membershipRequest, len(requests))

	for _, request := range requests {
		requestInfo, err := client.membershipRequestInfo(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("build groups request for user %s: %w", request.UserID, err)
		}
		item, err := batch.AddBatchRequestStep(*requestInfo)
		if err != nil {
			return nil, fmt.Errorf("batch groups request for user %s: %w", request.UserID, err)
		}
		if item.GetId() == nil {
			return nil, fmt.Errorf("batch groups request for user %s: missing request ID", request.UserID)
		}
		requestsByID[*item.GetId()] = request
	}

	response, err := batch.Send(ctx, adapter)
	if err != nil {
		return nil, fmt.Errorf("fetch user groups: %w", err)
	}

	var followups []membershipRequest
	for requestID, request := range requestsByID {
		if response.GetResponseById(requestID) == nil {
			return nil, fmt.Errorf("graph batch missing response for user %s", request.UserID)
		}
		page, err := graphcore.GetBatchResponseById[graphmodels.GroupCollectionResponseable](
			response,
			requestID,
			graphmodels.CreateGroupCollectionResponseFromDiscriminatorValue,
		)
		if err != nil {
			return nil, fmt.Errorf("fetch groups for user %s: %w", request.UserID, err)
		}
		for _, graphGroup := range page.GetValue() {
			groupID, err := parseGraphID(graphGroup.GetId())
			if err != nil {
				return nil, fmt.Errorf("group membership ID for user %s: %w", request.UserID, err)
			}
			if _, ok := knownGroups[groupID]; ok {
				members[groupID] = append(members[groupID], request.UserID)
			}
		}
		if nextLink := stringValue(page.GetOdataNextLink()); nextLink != "" {
			followups = append(followups, membershipRequest{
				UserID:   request.UserID,
				NextLink: nextLink,
			})
		}
	}
	return followups, nil
}

func (client *Client) membershipRequestInfo(
	ctx context.Context,
	request membershipRequest,
) (*abstractions.RequestInformation, error) {
	builder := client.graph.Users().ByUserId(request.UserID.String()).TransitiveMemberOf().GraphGroup()
	if request.NextLink != "" {
		return builder.WithUrl(request.NextLink).ToGetRequestInformation(ctx, nil)
	}
	return builder.ToGetRequestInformation(
		ctx,
		&graphusers.ItemTransitiveMemberOfGraphGroupRequestBuilderGetRequestConfiguration{
			QueryParameters: &graphusers.ItemTransitiveMemberOfGraphGroupRequestBuilderGetQueryParameters{
				Select: []string{"id"},
				Top:    new(graphPageSize),
			},
		},
	)
}

type membershipRequest struct {
	UserID   uuid.UUID
	NextLink string
}

func advancedQueryHeaders() *abstractions.RequestHeaders {
	headers := abstractions.NewRequestHeaders()
	headers.Add("ConsistencyLevel", "eventual")
	return headers
}

func parseGraphID(value *string) (uuid.UUID, error) {
	if value == nil {
		return uuid.Nil, fmt.Errorf("missing")
	}
	id, err := uuid.Parse(*value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse %q: %w", *value, err)
	}
	return id, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
