package entrasync

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/microsoft/kiota-abstractions-go/authentication"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
)

const (
	testUserOne    = "10000000-0000-0000-0000-000000000001"
	testUserTwo    = "10000000-0000-0000-0000-000000000002"
	testGroupOne   = "20000000-0000-0000-0000-000000000001"
	testGroupTwo   = "20000000-0000-0000-0000-000000000002"
	testGroupThree = "20000000-0000-0000-0000-000000000003"
	testOtherGroup = "20000000-0000-0000-0000-000000000099"
)

func TestClientFetchUsesSelectedFieldsPagingAndBatchedMemberships(t *testing.T) {
	const baseURL = "https://graph.test/v1.0"
	transport := &graphFetchTransport{t: t, baseURL: baseURL}
	client := newTestClient(t, &http.Client{Transport: transport}, baseURL)

	snapshot, err := client.Fetch(t.Context())
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if transport.batchCalls != 2 {
		t.Fatalf("batch calls = %d, want 2 for membership pagination", transport.batchCalls)
	}

	want := &Snapshot{
		Users: []User{
			{
				ID:          uuid.MustParse(testUserOne),
				UPN:         "user.one@example.invalid",
				DisplayName: "Example User One",
				Department:  "Teaching",
			},
			{
				ID:          uuid.MustParse(testUserTwo),
				UPN:         "user.two@example.invalid",
				DisplayName: "Example User Two",
				Department:  "Operations",
			},
		},
		Groups: []Group{
			{ID: uuid.MustParse(testGroupOne), DisplayName: "Example Group One", Description: "First group"},
			{ID: uuid.MustParse(testGroupTwo), DisplayName: "Example Group Two"},
			{ID: uuid.MustParse(testGroupThree), DisplayName: "Example Group Three"},
		},
		Members: map[uuid.UUID][]uuid.UUID{
			uuid.MustParse(testGroupOne):   {uuid.MustParse(testUserOne)},
			uuid.MustParse(testGroupTwo):   {uuid.MustParse(testUserTwo)},
			uuid.MustParse(testGroupThree): {uuid.MustParse(testUserOne)},
		},
	}
	if !reflect.DeepEqual(snapshot, want) {
		t.Fatalf("snapshot = %#v, want %#v", snapshot, want)
	}
}

type graphFetchTransport struct {
	t          *testing.T
	baseURL    string
	batchCalls int
}

func (transport *graphFetchTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Path {
	case "/v1.0/users":
		return transport.users(req), nil
	case "/v1.0/groups":
		return transport.groups(req), nil
	case "/v1.0/$batch":
		return transport.batch(req), nil
	default:
		return nil, errors.New("unexpected Graph request: " + req.URL.String())
	}
}

func (transport *graphFetchTransport) users(req *http.Request) *http.Response {
	checkAdvancedQuery(transport.t, req, "accountEnabled eq true and userType eq 'Member'")
	if req.URL.Query().Get("$skiptoken") == "users-2" {
		return jsonResponse(map[string]any{
			"value": []map[string]any{{
				"id":                testUserTwo,
				"userPrincipalName": "user.two@example.invalid",
				"displayName":       "Example User Two",
				"department":        "Operations",
			}},
		})
	}
	checkSelectedFields(transport.t, req, []string{
		"id",
		"userPrincipalName",
		"displayName",
		"department",
	})
	return jsonResponse(map[string]any{
		"@odata.nextLink": transport.baseURL + "/users?$select=id,userPrincipalName,displayName,department&$filter=accountEnabled%20eq%20true%20and%20userType%20eq%20%27Member%27&$count=true&$top=999&$skiptoken=users-2",
		"value": []map[string]any{{
			"id":                testUserOne,
			"userPrincipalName": "user.one@example.invalid",
			"displayName":       "Example User One",
			"department":        "Teaching",
		}},
	})
}

func (transport *graphFetchTransport) groups(req *http.Request) *http.Response {
	checkAdvancedQuery(transport.t, req, "securityEnabled eq true")
	if req.URL.Query().Get("$skiptoken") == "groups-2" {
		return jsonResponse(map[string]any{
			"value": []map[string]any{
				{"id": testGroupTwo, "displayName": "Example Group Two"},
				{"id": testGroupThree, "displayName": "Example Group Three"},
			},
		})
	}
	checkSelectedFields(transport.t, req, []string{"id", "displayName", "description"})
	return jsonResponse(map[string]any{
		"@odata.nextLink": transport.baseURL + "/groups?$select=id,displayName,description&$filter=securityEnabled%20eq%20true&$count=true&$top=999&$skiptoken=groups-2",
		"value": []map[string]any{{
			"id":          testGroupOne,
			"displayName": "Example Group One",
			"description": "First group",
		}},
	})
}

func (transport *graphFetchTransport) batch(req *http.Request) *http.Response {
	transport.batchCalls++
	var body testBatchRequestBody
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		transport.t.Fatalf("decode batch request: %v", err)
	}
	responses := make([]map[string]any, 0, len(body.Requests))
	for _, request := range body.Requests {
		requestURL, err := url.QueryUnescape(request.URL)
		if err != nil {
			transport.t.Fatalf("decode batch URL %q: %v", request.URL, err)
		}
		responses = append(responses, map[string]any{
			"id":      request.ID,
			"status":  http.StatusOK,
			"headers": map[string]string{"Content-Type": "application/json"},
			"body":    transport.membershipResponse(requestURL),
		})
	}
	return jsonResponse(map[string]any{"responses": responses})
}

func (transport *graphFetchTransport) membershipResponse(requestURL string) map[string]any {
	switch {
	case strings.Contains(requestURL, "/users/"+testUserOne+"/transitiveMemberOf/graph.group") &&
		strings.Contains(requestURL, "$skiptoken=membership-2"):
		return map[string]any{"value": []map[string]any{{"id": testGroupThree}}}
	case strings.Contains(requestURL, "/users/"+testUserOne+"/transitiveMemberOf/graph.group"):
		return map[string]any{
			"@odata.nextLink": transport.baseURL + "/users/" + testUserOne + "/transitiveMemberOf/graph.group?$select=id&$top=999&$skiptoken=membership-2",
			"value": []map[string]any{
				{"id": testGroupOne},
				{"id": testOtherGroup},
			},
		}
	case strings.Contains(requestURL, "/users/"+testUserTwo+"/transitiveMemberOf/graph.group"):
		return map[string]any{"value": []map[string]any{{"id": testGroupTwo}}}
	default:
		transport.t.Fatalf("unexpected membership request URL %q", requestURL)
		return nil
	}
}

func newTestClient(t *testing.T, httpClient *http.Client, baseURL string) *Client {
	t.Helper()
	adapter, err := msgraphsdk.NewGraphRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(
		&authentication.AnonymousAuthenticationProvider{},
		nil,
		nil,
		httpClient,
	)
	if err != nil {
		t.Fatalf("create Graph request adapter: %v", err)
	}
	adapter.SetBaseUrl(baseURL)
	return newClient(msgraphsdk.NewGraphServiceClient(adapter))
}

func checkAdvancedQuery(t *testing.T, req *http.Request, filter string) {
	t.Helper()
	if got := req.Header.Get("Consistencylevel"); got != "eventual" {
		t.Fatalf("ConsistencyLevel = %q, want eventual", got)
	}
	if got := req.URL.Query().Get("$filter"); got != filter {
		t.Fatalf("$filter = %q, want %q", got, filter)
	}
	if got := req.URL.Query().Get("$count"); got != "true" {
		t.Fatalf("$count = %q, want true", got)
	}
}

func checkSelectedFields(t *testing.T, req *http.Request, want []string) {
	t.Helper()
	got := strings.Split(req.URL.Query().Get("$select"), ",")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("$select = %#v, want %#v", got, want)
	}
}

type testBatchRequestBody struct {
	Requests []struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"requests"`
}

func jsonResponse(body any) *http.Response {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		panic(err)
	}
	return &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		Body:          io.NopCloser(&buf),
		ContentLength: int64(buf.Len()),
	}
}
