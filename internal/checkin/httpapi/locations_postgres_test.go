//go:build postgres

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/woodleighschool/goodies/auth/authn"
	"github.com/woodleighschool/goodies/auth/authz"
	"github.com/woodleighschool/goodies/bloby"
	blobydb "github.com/woodleighschool/goodies/bloby/pgxstore"

	"github.com/woodleighschool/woodgate/internal/checkin"
	"github.com/woodleighschool/woodgate/internal/testutil/testdb"
)

func TestLocationImageUploadContract(t *testing.T) {
	db, ctx := testdb.Open(t)
	logger := slog.New(slog.DiscardHandler)
	objects, err := bloby.New(ctx, blobydb.New(db), bloby.Config{Kind: bloby.KindFile, TransferTTL: time.Minute, File: bloby.FileConfig{Root: t.TempDir(), BaseURL: "https://storage.invalid", CapabilityKeyHex: strings.Repeat("42", 32)}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	routes := humachi.New(router, huma.DefaultConfig("test", "test"))
	deps := Dependencies{Service: checkin.NewService(checkin.NewStore(db, objects), objects), Authorizer: allowAll{}, Logger: logger}
	registerLocations(routes, deps)
	registerLocationAttachments(routes, deps)

	var imageBytes bytes.Buffer
	if err := jpeg.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	content, err := bloby.Digest(bytes.NewReader(imageBytes.Bytes()))
	if err != nil {
		t.Fatal(err)
	}

	if response := postJSON(t, router, locationBackgroundPath, map[string]any{"filename": "background.jpg"}); response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("undeclared upload: %d %s", response.Code, response.Body.String())
	}
	response := postJSON(t, router, locationBackgroundPath, map[string]any{
		"filename": "background.jpg", "size_bytes": content.SizeBytes, "sha256": content.SHA256, "crc64nvme": content.CRC64NVME,
	})
	if response.Code != http.StatusCreated {
		t.Fatalf("declared upload: %d %s", response.Code, response.Body.String())
	}
	var intent struct {
		ObjectID int64              `json:"object_id"`
		Upload   bloby.UploadAction `json:"upload"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &intent); err != nil {
		t.Fatal(err)
	}
	if intent.Upload.Strategy != bloby.StrategyDirectPut || intent.Upload.Target == nil {
		t.Fatalf("upload=%+v", intent.Upload)
	}

	location := checkin.LocationMutation{Name: "Uploaded location", Enabled: true, BackgroundObjectID: &intent.ObjectID, GroupIDs: []int64{}}
	if response := postJSON(t, router, "/api/locations", location); response.Code != http.StatusBadRequest {
		t.Fatalf("save before the upload arrived: %d %s", response.Code, response.Body.String())
	}

	target := intent.Upload.Target
	put := httptest.NewRequestWithContext(ctx, target.Method, target.URL, &imageBytes)
	for key, value := range target.Headers {
		put.Header.Set(key, value)
	}
	stored := httptest.NewRecorder()
	objects.TransferHandler().ServeHTTP(stored, put)
	if stored.Code != http.StatusNoContent {
		t.Fatalf("upload status=%d", stored.Code)
	}

	response = postJSON(t, router, "/api/locations", location)
	if response.Code != http.StatusCreated {
		t.Fatalf("save after the upload arrived: %d %s", response.Code, response.Body.String())
	}
	var created checkin.Location
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.BackgroundFile == nil || created.BackgroundFile.ContentType != "image/jpeg" ||
		created.BackgroundFile.SizeBytes != content.SizeBytes || created.BackgroundFile.SHA256 != content.SHA256 {
		t.Fatalf("background=%+v", created.BackgroundFile)
	}
}

func postJSON(t *testing.T, router http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	ctx := authn.WithPrincipal(t.Context(), &authn.Principal{ID: 1})
	request := httptest.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

type allowAll struct{}

func (allowAll) CanAll(context.Context, int64, ...authz.Requirement) (bool, error) { return true, nil }
