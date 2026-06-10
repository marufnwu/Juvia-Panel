.PHONY: all build build-api build-agent build-ui test lint clean install run run-api run-agent

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Binary names
API_BINARY=api
AGENT_BINARY=agent

# Build directories
BUILD_DIR=build
BACKEND_DIR=backend
FRONTEND_DIR=frontend

# Default target
all: build

build: build-api build-agent build-ui

build-api:
	@echo "Building API server..."
	@mkdir -p $(BUILD_DIR)
	cd $(BACKEND_DIR) && $(GOBUILD) -o ../$(BUILD_DIR)/$(API_BINARY) ./cmd/api

build-agent:
	@echo "Building agent..."
	@mkdir -p $(BUILD_DIR)
	cd $(BACKEND_DIR) && $(GOBUILD) -o ../$(BUILD_DIR)/$(AGENT_BINARY) ./cmd/agent

build-ui:
	@echo "Building UI..."
	cd $(FRONTEND_DIR) && npm install --legacy-peer-deps && npm run build

start-ui:
	@echo "Starting Next.js server..."
	cd $(FRONTEND_DIR) && npm run start

test:
	@echo "Running tests..."
	cd $(BACKEND_DIR) && $(GOTEST) ./...

lint:
	@echo "Linting backend..."
	cd $(BACKEND_DIR) && $(GOFMT) ./...
	@echo "Linting frontend..."
	cd $(FRONTEND_DIR) && npm run lint

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(FRONTEND_DIR)/.next

install:
	@echo "Installing dependencies..."
	cd $(BACKEND_DIR) && $(GOMOD) download
	cd $(FRONTEND_DIR) && npm install

run: run-api run-agent

run-api:
	@echo "Starting API server..."
	cd $(BACKEND_DIR) && $(GOCMD) run ./cmd/api

run-agent:
	@echo "Starting agent..."
	cd $(BACKEND_DIR) && $(GOCMD) run ./cmd/agent