# Makefile for Expense Management (adapt as needed)

MODULE ?= github.com/yourusername/expense-management
OPENAPI ?= api/openapi.yml
OAPI_CFG ?= api/oapi_codegen.yaml
DB_SOURCE ?= postgresql://root:secret@localhost:5432/expense_management_db?sslmode=disable
STEP ?= 0

.PHONY: build run migrate migrate.rollback migration migration.go generate.openapi \
        swagger seed seed-fresh deps dev-setup dev-setup-with-data docker-build docker-run \
        lint clean test

# Build the application (expects main.go)
build:
	mkdir -p bin
	go build -o bin/expense-management main.go

# Run the application (expects main.go to accept a server command)
run:
	go run main.go server

# Create bin dir + build debug
build-debug:
	mkdir -p bin
	go build -gcflags "all=-N -l" -o bin/expense-management.debug main.go

run-debug:
	./bin/expense-management.debug server

# Generate OpenAPI code via oapi-codegen using config file
generate.openapi:
	@echo "generating openapi types "
	@command -v oapi-codegen >/dev/null 2>&1 || { \
		echo "oapi-codegen not found - installing..."; \
		go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest; \
	}
	@oapi-codegen -config $(OAPI_CFG) $(OPENAPI)

# Create a new SQL migration (sql)
# Usage: make migration NAME=create_users_table
migration:
	$(call check_defined, NAME)
	goose -dir="./db/migrations" create $(NAME) sql

# Create a new Go migration (go)
# Usage: make migration.go NAME=create_users_table
migration.go:
	$(call check_defined, NAME)
	goose -dir="./db/migrations" create $(NAME) go

# Run database migration (prefers goose CLI if available, else fallback to cobra migrate command)
migrate:
	@if command -v goose >/dev/null 2>&1; then \
		goose -dir="./db/migrations" postgres "$(DB_SOURCE)" up; \
	else \
		go run ./cmd migrate; \
	fi

migrate.rollback:
	@if command -v goose >/dev/null 2>&1; then \
		goose -dir="./db/migrations" postgres "$(DB_SOURCE)" down; \
	else \
		go run ./cmd migrate --rollback; \
	fi

# Generate swagger docs with swag (optional)
swagger:
	@swag init -g cmd/http_server.go -o ./internal/transport/swagger --parseDependency --parseInternal || true

# Run tests
test:
	go test ./... -v

# Clean build artifacts
clean:
	rm -rf bin/ dist/ docs/
	find . -name "*.gen.go" -type f -delete || true

# Install/resolve Go deps
deps:
	go mod download
	go mod tidy

# Seed DB (expects cmd seed implemented)

seed:
	@$(MAKE) build
	@./bin/expense-management seed

# Re-seed DB: clear + seed

seed-fresh:
	@$(MAKE) build
	@./bin/expense-management seed --clear

# Setup dev env
dev-setup: deps migrate
	@echo "Dev env ready."

dev-setup-with-data: deps migrate seed
	@echo "Dev env with seeded data ready."

# Docker targets
docker-build:
	docker build -t expense-management:latest .

docker-run:
	docker-compose up -d

# Linting (golangci-lint expected)
lint:
	golangci-lint run || true

# helper for check_defined
check_defined = \
    $(strip $(foreach 1,$1, \
        $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = \
    $(if $(value $1),, \
      $(error Undefined $1$(if $2, ($2))))
