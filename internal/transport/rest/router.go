package rest

import (
	"database/sql"
	"net/http"

	"github.com/frahmantamala/expense-management/internal/transport/swagger"
	"github.com/go-chi/chi"
)

func RegisterAllRoutes(router *chi.Mux, db *sql.DB) {
	healthHandler := NewHealthHandler(db)

	// OpenAPI spec file
	router.Get("/openapi.yml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./api/openapi.yml")
	})
	// Swagger UI route
	router.Handle("/swagger/*", swagger.Handler())

	// Health check route
	router.Get("/health", healthHandler.healthCheckHandler)
	router.Get("/ping", healthHandler.pingHandler)
}
