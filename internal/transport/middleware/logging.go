package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
)

// sensitiveFields are field names that should be filtered from logs
var sensitiveFields = []string{
	"password",
	"password_hash",
	"passwordhash",
	"token",
	"access_token",
	"refresh_token",
	"authorization",
	"secret",
	"key",
	"api_key",
	"session",
	"credential",
	"auth",
}

func LoggingMiddleware(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			reqID := middleware.GetReqID(r.Context())

			logRequest(logger, r, reqID)

			ww := &responseWriter{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
			}

			next.ServeHTTP(ww, r)

			duration := time.Since(start)
			logResponse(logger, ww, duration, reqID)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture response body
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// logRequest logs the incoming HTTP request with sensitive data filtered
func logRequest(logger *slog.Logger, r *http.Request, reqID string) {
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	headers := filterSensitiveHeaders(r.Header)

	filteredBody := filterSensitiveBody(bodyBytes)

	logger.Info("incoming request",
		"request_id", reqID,
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
		"headers", headers,
		"body", filteredBody,
	)
}

func logResponse(logger *slog.Logger, rw *responseWriter, duration time.Duration, reqID string) {
	statusCode := rw.statusCode
	if statusCode == 0 {
		statusCode = 200
	}

	filteredBody := filterSensitiveBody(rw.body.Bytes())

	logLevel := slog.LevelInfo
	if statusCode >= 400 && statusCode < 500 {
		logLevel = slog.LevelWarn
	} else if statusCode >= 500 {
		logLevel = slog.LevelError
	}

	logger.Log(nil, logLevel, "response",
		"request_id", reqID,
		"status_code", statusCode,
		"duration_ms", duration.Milliseconds(),
		"response_size", rw.body.Len(),
		"body", filteredBody,
	)
}

// filterSensitiveHeaders removes or masks sensitive headers
func filterSensitiveHeaders(headers http.Header) map[string]string {
	filtered := make(map[string]string)

	for name, values := range headers {
		lowerName := strings.ToLower(name)

		// Check if header contains sensitive data
		isSensitive := false
		for _, sensitiveField := range sensitiveFields {
			if strings.Contains(lowerName, sensitiveField) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			filtered[name] = "[FILTERED]"
		} else {
			filtered[name] = strings.Join(values, ", ")
		}
	}

	return filtered
}

// filterSensitiveBody removes or masks sensitive fields from JSON body
func filterSensitiveBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	// Try to parse as JSON
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		// If not JSON, return as string but check for sensitive patterns
		bodyStr := string(body)
		for _, sensitiveField := range sensitiveFields {
			if strings.Contains(strings.ToLower(bodyStr), sensitiveField) {
				return "[FILTERED - Contains sensitive data]"
			}
		}
		return bodyStr
	}

	// Filter sensitive fields from JSON
	filtered := filterSensitiveJSON(jsonData)

	// Convert back to JSON string
	filteredBytes, err := json.Marshal(filtered)
	if err != nil {
		return "[ERROR - Failed to marshal filtered JSON]"
	}

	return string(filteredBytes)
}

// filterSensitiveJSON recursively filters sensitive fields from JSON data
func filterSensitiveJSON(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		filtered := make(map[string]interface{})
		for key, value := range v {
			lowerKey := strings.ToLower(key)

			// Check if key is sensitive
			isSensitive := false
			for _, sensitiveField := range sensitiveFields {
				if strings.Contains(lowerKey, sensitiveField) {
					isSensitive = true
					break
				}
			}

			if isSensitive {
				filtered[key] = "[FILTERED]"
			} else {
				filtered[key] = filterSensitiveJSON(value)
			}
		}
		return filtered
	case []interface{}:
		filtered := make([]interface{}, len(v))
		for i, item := range v {
			filtered[i] = filterSensitiveJSON(item)
		}
		return filtered
	default:
		return v
	}
}
