package router

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/woodleighschool/woodgate/internal/platform/httpmiddleware"
)

type statusResponse struct {
	Status string `json:"status"`
}

const readinessTimeout = 2 * time.Second

// New creates the root HTTP router for WoodGate.
func New(
	logger *slog.Logger,
	readinessCheck func(context.Context) error,
	registerAuthRoutes func(chi.Router),
	registerAPIRoutes func(chi.Router),
	frontendDir string,
) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(httpmiddleware.RequestLogger(logger))
	router.Use(middleware.Recoverer)

	router.Get("/healthz", statusHandler(logger, http.StatusOK, statusResponse{Status: "ok"}))
	router.Get("/readyz", readinessHandler(logger, readinessCheck))
	router.Route("/auth", registerAuthRoutes)
	router.Route("/api/v1", registerAPIRoutes)
	mountFrontend(router, frontendDir)

	return router
}

func statusHandler(
	logger *slog.Logger,
	statusCode int,
	payload statusResponse,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(statusCode)
		if err := json.NewEncoder(writer).Encode(payload); err != nil {
			logger.Error("encode health response", "error", err)
		}
	}
}

func readinessHandler(logger *slog.Logger, readinessCheck func(context.Context) error) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if readinessCheck != nil {
			checkContext, cancel := context.WithTimeout(request.Context(), readinessTimeout)
			defer cancel()

			if err := readinessCheck(checkContext); err != nil {
				logger.Warn("readiness check failed", "error", err)
				statusHandler(
					logger,
					http.StatusServiceUnavailable,
					statusResponse{Status: "not_ready"},
				)(
					writer,
					request,
				)
				return
			}
		}

		statusHandler(logger, http.StatusOK, statusResponse{Status: "ready"})(writer, request)
	}
}
