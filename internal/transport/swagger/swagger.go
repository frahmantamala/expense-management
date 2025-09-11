package swagger

import (
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

func Handler() http.Handler {
	// Serve the comprehensive OpenAPI spec from api/openapi3.yml
	return httpSwagger.Handler(
		httpSwagger.URL("/openapi.yml"), // URL to the OpenAPI spec served at root
	)
}
