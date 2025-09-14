# Makefile for Expense Management (adapt as needed)

MODULE ?= github.com/frahmantamala/expense-management
OPENAPI ?= api/openapi.yml
OAPI_CFG ?= api/oapi_codegen.yaml
DB_SOURCE ?= postgresql://root:secret@localhost:5432/expense_management_db?sslmode=disable
STEP ?= 0

.PHONY: build run migrate migrate.rollback migration migration.go generate.openapi \
        swagger seed seed-fresh deps dev-setup dev-setup-with-data \
        lint clean test test-coverage test-cover test-auth test-payment test-expense \
        test-postgres test-race test-short test-flaky test-summary \
        docker-up docker-down docker-logs docker-clean

build:
	mkdir -p bin
	go build -o bin/expense-management main.go

run:
	go run main.go server

build-debug:
	mkdir -p bin
	go build -gcflags "all=-N -l" -o bin/expense-management.debug main.go

run-debug:
	./bin/expense-management.debug server

generate.openapi:
	@echo "generating openapi types "
	@command -v oapi-codegen >/dev/null 2>&1 || { \
		echo "oapi-codegen not found - installing..."; \
		go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest; \
	}
	@oapi-codegen -config $(OAPI_CFG) $(OPENAPI)

migration:
	$(call check_defined, NAME)
	goose -dir="./db/migrations" create $(NAME) sql

migration.go:
	$(call check_defined, NAME)
	goose -dir="./db/migrations" create $(NAME) go

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

swagger:
	@swag init -g cmd/http_server.go -o ./internal/transport/swagger --parseDependency --parseInternal || true

test:
	@echo "Running all tests..."
	go test ./... -v

test-coverage:
	@echo "Running tests with coverage..."
	go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-cover:
	@echo "Running tests with coverage summary..."
	go test ./... -v -coverprofile=coverage.out
	go tool cover -func=coverage.out

test-auth:
	@echo "Running auth module tests..."
	go test ./internal/auth -v

test-payment:
	@echo "Running payment module tests..."
	go test ./internal/payment -v

test-expense:
	@echo "Running expense module tests..."
	go test ./internal/expense -v

test-postgres:
	@echo "Running PostgreSQL repository tests..."
	go test ./internal/payment/postgres ./internal/expense/postgres -v

#
test-race:
	@echo "Running tests with race detection..."
	go test ./... -v -race


test-short:
	@echo "Running tests in short mode..."
	go test ./... -v -short


test-flaky:
	@echo "Running tests 10 times to catch flaky tests..."
	go test ./... -v -count=10


test-summary:
	@echo "=== TEST SUMMARY ==="
	@echo "Auth: $$(go test ./internal/auth -v | grep 'Ran.*Specs' | awk '{print $$2" tests"}' || echo 'No tests')"
	@echo "Payment Service: $$(go test ./internal/payment -v | grep 'Ran.*Specs' | awk '{print $$2" tests"}' || echo 'No tests')"
	@echo "Expense Service: $$(go test ./internal/expense -v | grep 'Ran.*Specs' | awk '{print $$2" tests"}' || echo 'No tests')"
	@echo "Payment PostgreSQL: $$(go test ./internal/payment/postgres -v | grep 'Ran.*Specs' | awk '{print $$2" tests"}' || echo 'No tests')"
	@echo "Expense PostgreSQL: $$(go test ./internal/expense/postgres -v | grep 'Ran.*Specs' | awk '{print $$2" tests"}' || echo 'No tests')"
	@echo "================================"


clean:
	rm -rf bin/ dist/ docs/
	find . -name "*.gen.go" -type f -delete || true


deps:
	go mod download
	go mod tidy



seed:
	@$(MAKE) build
	@./bin/expense-management seed



seed-fresh:
	@$(MAKE) build
	@./bin/expense-management seed --clear


dev-setup: deps migrate
	@echo "Dev env ready."

dev-setup-with-data: deps migrate seed
	@echo "Dev env with seeded data ready."


docker-up:
	@echo "Starting Expense Management..."
	docker compose up -d

docker-down:
	@echo "Stopping..."
	docker compose down


lint:
	golangci-lint run || true


check_defined = \
    $(strip $(foreach 1,$1, \
        $(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = \
    $(if $(value $1),, \
      $(error Undefined $1$(if $2, ($2))))
