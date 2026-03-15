package router

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/rest"
)

func mountFrontend(router chi.Router, dir string) {
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		return
	}

	fileServer, err := rest.NewFileServer("/", dir, rest.FsOptSPA)
	if err != nil {
		return
	}

	frontend := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
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
