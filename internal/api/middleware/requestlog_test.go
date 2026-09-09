package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRequestLogPathUsesRoutePattern(t *testing.T) {
	router := chi.NewRouter()
	router.Get("/api/stations/{id}", func(_ http.ResponseWriter, r *http.Request) {
		if got, want := requestLogPath(r), "/api/stations/{id}"; got != want {
			t.Fatalf("request log path = %q, want %q", got, want)
		}
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/stations/42", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
}
