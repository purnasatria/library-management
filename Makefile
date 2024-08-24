# Variables
DOCKER_COMPOSE = docker-compose
GO = go
GOTEST = $(GO) test
GOVET = $(GO) vet
BINARY_NAME = main
MIGRATION_TOOL = migrate
PROTOC = protoc

# Service paths
SERVICES = auth author category book
SERVICE_PATHS = $(addprefix ./cmd/,$(SERVICES))

# Docker commands
.PHONY: up
up:
	$(DOCKER_COMPOSE) up --build -d

.PHONY: down
down:
	$(DOCKER_COMPOSE) down

.PHONY: logs
logs:
	$(DOCKER_COMPOSE) logs -f

# Build commands
.PHONY: build
build:
	@for service in $(SERVICES); do \
		echo "Building $$service service..."; \
		$(GO) build -o $(BINARY_NAME)-$$service ./cmd/$$service; \
	done

# Run commands (for local development without Docker)
.PHONY: run-auth run-author run-category run-book
run-auth:
	$(GO) run ./cmd/auth

run-author:
	$(GO) run ./cmd/author

run-category:
	$(GO) run ./cmd/category

run-book:
	$(GO) run ./cmd/book

# Test commands
.PHONY: test
test:
	$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

# Lint and format
.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	gofmt -s -w .

# Database migration commands
.PHONY: migrate-up migrate-down
migrate-up:
	@for service in $(SERVICES); do \
		echo "Running migrations for $$service..."; \
		$(MIGRATION_TOOL) -path=./migrations/$$service -database=$${service^^}_DATABASE_URL up; \
	done

migrate-down:
	@for service in $(SERVICES); do \
		echo "Reverting migrations for $$service..."; \
		$(MIGRATION_TOOL) -path=./migrations/$$service -database=$${service^^}_DATABASE_URL down; \
	done

# Proto generation command
.PHONY: proto-generate
proto-generate:
	@if [ -z "$(SVC)" ]; then \
		echo "Error: SVC is not set. Use 'make proto-generate SVC=<service_name>'"; \
		exit 1; \
	fi
	$(PROTOC) -I ./api/proto \
		--go_out=./api/gen/$(SVC) --go_opt=paths=source_relative \
		--go-grpc_out=./api/gen/$(SVC) --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=./api/gen/$(SVC) --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=./api/swagger ./api/proto/$(SVC).proto

# Create migration table command
.PHONY: create-migration
create-table:
	@if [ -z "$(SVC)" ]; then \
		echo "Error: SVC is not set. Use 'make create-table SVC=<service_name> SEQ=<migration_name>'"; \
		exit 1; \
	fi
	@if [ -z "$(SEQ)" ]; then \
		echo "Error: SEQ is not set. Use 'make create-table SVC=<service_name> SEQ=<migration_name>'"; \
		exit 1; \
	fi
	$(MIGRATION_TOOL) create -ext sql -dir migrations/$(SVC) -seq $(SEQ)

# Clean up
.PHONY: clean
clean:
	@for service in $(SERVICES); do \
		rm -f $(BINARY_NAME)-$$service; \
	done
	rm -f coverage.out

# Help command
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  up                    - Start all services with Docker Compose"
	@echo "  down                  - Stop all services"
	@echo "  logs                  - View logs of all services"
	@echo "  build                 - Build all services"
	@echo "  run-auth              - Run auth service locally"
	@echo "  run-author            - Run author service locally"
	@echo "  run-category          - Run category service locally"
	@echo "  run-book              - Run book service locally"
	@echo "  test                  - Run tests for all services"
	@echo "  test-coverage         - Run tests with coverage"
	@echo "  lint                  - Run linter"
	@echo "  fmt                   - Format code"
	@echo "  migrate-up            - Run database migrations"
	@echo "  migrate-down          - Revert database migrations"
	@echo "  proto-generate        - Generate Proto files (use with SVC=<service_name>)"
	@echo "  create-migration      - Create a new migration (use with SVC=<service_name> SEQ=<migration_name>)"
	@echo "  clean                 - Remove built binaries and coverage files"
