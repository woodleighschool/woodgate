package router

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/rest"
)

func mountFrontend(router chi.Router, frontendFS fs.FS) {
	fileServer := http.FileServer(http.FS(frontendFS))

	frontend := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api" ||
			strings.HasPrefix(request.URL.Path, "/api/") ||
			request.URL.Path == "/auth" ||
			strings.HasPrefix(request.URL.Path, "/auth/") ||
			request.URL.Path == "/healthz" ||
			request.URL.Path == "/readyz" {
			http.NotFound(writer, request)
			return
		}

		name := strings.TrimPrefix(request.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(frontendFS, name); err != nil {
			fallback := request.Clone(request.Context())
			fallbackURL := *request.URL
			fallbackURL.Path = "/"
			fallback.URL = &fallbackURL
			request = fallback
		}

		if strings.HasPrefix(request.URL.Path, "/assets/") {
			writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			writer.Header().Set("Cache-Control", "no-cache")
		}
		fileServer.ServeHTTP(writer, request)
	})

	router.NotFound(rest.Gzip(
		"text/html",
		"text/css",
		"application/javascript",
		"application/json",
	)(frontend).ServeHTTP)
}
