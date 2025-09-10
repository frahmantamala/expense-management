package rest

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
)

type HealthResponse struct {
	Status     HealthStatus          `json:"status"`
	CheckedAt  time.Time             `json:"checked_at"`
	Components map[string]CheckEntry `json:"components"`
}

type CheckEntry struct {
	Status     HealthStatus   `json:"status"`
	Message    string         `json:"message,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
	CheckedAt  time.Time      `json:"checked_at"`
	DurationMs int64          `json:"duration_ms"`
}

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// HandleLiveness → just says service is up
func (h *HealthHandler) pingHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"status": "OK"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleReadiness → checks DB connection
func (h *HealthHandler) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	start := time.Now()
	err := h.db.PingContext(ctx)

	entry := CheckEntry{
		Status:     HealthHealthy,
		CheckedAt:  time.Now(),
		DurationMs: time.Since(start).Milliseconds(),
	}

	if err != nil {
		entry.Status = HealthUnhealthy
		entry.Message = err.Error()
	}

	resp := HealthResponse{
		Status:     entry.Status,
		CheckedAt:  time.Now(),
		Components: map[string]CheckEntry{"postgres": entry},
	}

	statusCode := http.StatusOK
	if entry.Status == HealthUnhealthy {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}
