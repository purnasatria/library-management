# Variables
DOCKER_COMPOSE = docker compose
GO = go
GOTEST = $(GO) test
GOVET = $(GO) vet
BINARY_NAME = main
MIGRATION_TOOL = migrate
PROTOC = protoc
DOCKER_USERNAME := purnasatria
BUILD_DATE := $(shell date -u +'%Y%m%d')
BUILD_NUMBER ?= 0

# Service paths
SERVICES = auth author category book
SERVICE_PATHS = $(addprefix cmd/,$(SERVICES))

# Function to get service-specific version
define get_service_version
$(shell cat internal/$(1)/VERSION)
endef

# Function to get service-specific git hash
define get_service_hash
$(shell git log -1 --format=%h -- ./cmd/$(1) ./internal/$(1))
endef

# Docker commands
.PHONY: up down logs

up:
	$(DOCKER_COMPOSE) up --build -d

down:
	$(DOCKER_COMPOSE) down

logs:
	$(DOCKER_COMPOSE) logs -f

# Build commands
.PHONY: build build-images push-images release-all $(SERVICES) $(addprefix build-image-,$(SERVICES)) $(addprefix push-image-,$(SERVICES)) $(addprefix release-,$(SERVICES))

build:
	@for service in $(SERVICES); do \
		echo "Building $$service service..."; \
		$(GO) build -o $(BINARY_NAME)-$$service ./cmd/$$service; \
	done

build-images:
	@for service in $(SERVICES); do \
		$(MAKE) build-image-$$service; \
	done

push-images: docker-login
	@for service in $(SERVICES); do \
		$(MAKE) push-image-$$service; \
	done

release-all: docker-login
	@for service in $(SERVICES); do \
		$(MAKE) release-$$service; \
	done

# Generic targets for building each service image
define make-build-image-target
build-image-$(1):
	$$(eval SERVICE_VERSION := $$(call get_service_version,$(1)))
	$$(eval SERVICE_HASH := $$(call get_service_hash,$(1)))
	@echo "Building $(1) service image..."
	docker build -t $$(DOCKER_USERNAME)/$(1)service:$$(SERVICE_VERSION) \
		--build-arg SERVICE_PATH=cmd/$(1)service \
		-f Dockerfile .
	docker tag $$(DOCKER_USERNAME)/$(1)service:$$(SERVICE_VERSION) $$(DOCKER_USERNAME)/$(1)service:$$(SERVICE_VERSION)-$$(SERVICE_HASH)
	docker tag $$(DOCKER_USERNAME)/$(1)service:$$(SERVICE_VERSION) $$(DOCKER_USERNAME)/$(1)service:latest
endef

# Generic targets for pushing each service image
define make-push-image-target
push-image-$(1):
	$$(eval SERVICE_VERSION := $$(call get_service_version,$(1)))
	$$(eval SERVICE_HASH := $$(call get_service_hash,$(1)))
	@echo "Pushing $(1) service image..."
	docker push $$(DOCKER_USERNAME)/$(1)service:$$(SERVICE_VERSION)
	docker push $$(DOCKER_USERNAME)/$(1)service:$$(SERVICE_VERSION)-$$(SERVICE_HASH)
	docker push $$(DOCKER_USERNAME)/$(1)service:latest
endef

# Generic targets for releasing each service
define make-release-target
release-$(1): build-image-$(1) push-image-$(1)
	@echo "$(1) service released successfully."
endef

# Generate targets for each service
$(foreach service,$(SERVICES),$(eval $(call make-build-image-target,$(service))))
$(foreach service,$(SERVICES),$(eval $(call make-push-image-target,$(service))))
$(foreach service,$(SERVICES),$(eval $(call make-release-target,$(service))))

# Docker login
.PHONY: docker-login
docker-login:
	@echo "Logging in to Docker Hub..."
	@docker login -u $(DOCKER_USERNAME)

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
.PHONY: test test-coverage
test:
	$(GOTEST) -v ./...

test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

# Lint and format
.PHONY: lint fmt
lint:
	golangci-lint run

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

# Create migration command
.PHONY: create-migration
create-migration:
	@if [ -z "$(SVC)" ]; then \
		echo "Error: SVC is not set. Use 'make create-migration SVC=<service_name> NAME=<migration_name>'"; \
		exit 1; \
	fi
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is not set. Use 'make create-migration SVC=<service_name> NAME=<migration_name>'"; \
		exit 1; \
	fi
	$(MIGRATION_TOOL) create -ext sql -dir migrations/$(SVC) -seq $(NAME)

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
	@echo "  build                 - Build all service binaries"
	@echo "  build-images          - Build Docker images for all services"
	@echo "  push-images           - Push Docker images for all services to Docker Hub"
	@echo "  release-all           - Build, push, and release all services"
	@echo "  build-image-<service> - Build Docker image for a specific service (e.g., build-image-auth)"
	@echo "  push-image-<service>  - Push Docker image for a specific service to Docker Hub (e.g., push-image-auth)"
	@echo "  release-<service>     - Build, push, and release a specific service (e.g., release-auth)"
	@echo "  run-<service>         - Run a specific service locally (e.g., run-auth)"
	@echo "  test                  - Run tests for all services"
	@echo "  test-coverage         - Run tests with coverage"
	@echo "  lint                  - Run linter"
	@echo "  fmt                   - Format code"
	@echo "  migrate-up            - Run database migrations for all services"
	@echo "  migrate-down          - Revert database migrations for all services"
	@echo "  proto-generate        - Generate Proto files (use with SVC=<service_name>)"
	@echo "  create-migration      - Create a new migration (use with SVC=<service_name> NAME=<migration_name>)"
	@echo "  clean                 - Remove built binaries and coverage files"
	@echo "  docker-login          - Log in to Docker Hub"
