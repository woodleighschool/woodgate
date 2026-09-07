package webui

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-chi/chi/v5"
)

func TestHandlerServesKnownRootAsset(t *testing.T) {
	t.Parallel()

	recorder := requestWeb(t, "/apple-touch-icon.png")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.String(); got != "png" {
		t.Fatalf("body = %q, want root asset content", got)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "image/png") {
		t.Fatalf("content type = %q, want image/png", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=86400" {
		t.Fatalf("cache control = %q, want root asset cache", got)
	}
}

func TestHandlerServesHashedAssetWithImmutableCache(t *testing.T) {
	t.Parallel()

	recorder := requestWeb(t, "/assets/app-abcd1234.js")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.String(); got != "console.log('ok')" {
		t.Fatalf("body = %q, want bundled asset content", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("cache control = %q, want immutable asset cache", got)
	}
}

func TestHandlerReturnsNotFoundForAssetLikeMiss(t *testing.T) {
	t.Parallel()

	recorder := requestWeb(t, "/missing.png")

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if strings.Contains(recorder.Body.String(), "__WOODGATE__") {
		t.Fatal("missing asset returned SPA index")
	}
}

func TestHandlerDoesNotTreatServerPathsAsSPARoutes(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"/api/not-a-route",
		"/storage/not-an-object",
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			recorder := requestWeb(t, path)
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
			}
			if strings.Contains(recorder.Body.String(), "__WOODGATE__") {
				t.Fatal("server path returned SPA index")
			}
		})
	}
}

func TestHandlerServesSPAIndex(t *testing.T) {
	t.Parallel()

	recorder := requestWeb(t, "/checkins")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `<meta name="woodgate-version" content="test">`) ||
		!strings.Contains(
			body,
			`<meta name="woodgate-server-url" content="https://woodgate.example">`,
		) {
		t.Fatalf("body did not include runtime config: %q", body)
	}
	if strings.Contains(body, "window.__WOODGATE__") {
		t.Fatalf("body included executable runtime config: %q", body)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache control = %q, want no-store", got)
	}
}

func TestHandlerEscapesRuntimeMetadata(t *testing.T) {
	t.Parallel()

	router := chi.NewRouter()
	NewHandler(HandlerOptions{
		FS: fstest.MapFS{
			"index.html": {Data: []byte("<!doctype html><html><head></head><body></body></html>")},
		},
		Version:   `\"><script>alert("version")</script>`,
		ServerURL: `https://woodgate.example/?value=\"><script>alert("url")</script>`,
		Logger:    slog.New(slog.DiscardHandler),
	}).RegisterRoutes(router)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	body := recorder.Body.String()
	if strings.Contains(body, `<script>alert`) {
		t.Fatalf("body included executable runtime metadata: %q", body)
	}
	if !strings.Contains(body, `&lt;script&gt;`) {
		t.Fatalf("body did not HTML-escape runtime metadata: %q", body)
	}
}

func TestHandlerRedirectsIndexHTMLToRoot(t *testing.T) {
	t.Parallel()

	recorder := requestWeb(t, "/index.html?from=sso")

	if recorder.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMovedPermanently)
	}
	if got := recorder.Header().Get("Location"); got != "/?from=sso" {
		t.Fatalf("location = %q, want root redirect with query", got)
	}
}

func requestWeb(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()

	router := chi.NewRouter()
	NewHandler(HandlerOptions{
		FS: fstest.MapFS{
			"apple-touch-icon.png": {Data: []byte("png")},
			"assets/app-abcd1234.js": {
				Data: []byte("console.log('ok')"),
			},
			"index.html": {Data: []byte("<!doctype html><html><head></head><body></body></html>")},
		},
		Version:   "test",
		ServerURL: "https://woodgate.example",
		Logger:    slog.New(slog.DiscardHandler),
	}).RegisterRoutes(router)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}
