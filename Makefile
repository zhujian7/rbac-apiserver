# MyTest API Server Makefile

# Variables
BINARY_NAME := rbac-apiserver
GO_VERSION := 1.24

IMAGE_NAME ?= quay.io/stolostron/rbac-apiserver
IMAGE_TAG ?= latest

# Go related variables
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOHOSTOS ?=$(shell $(GOCMD) env GOHOSTOS)
GOHOSTARCH ?=$(shell $(GOCMD) env GOHOSTARCH)

# Docker related variables
# Auto-detect container runtime: prefer podman locally, but use docker if available (e.g., in CI)
DOCKER := $(shell command -v docker 2>/dev/null || echo podman)
DOCKER_BUILD_ARGS := --platform linux/amd64

# Kubernetes related variables
KUBECTL := kubectl


# Build flags
LDFLAGS := -w -s
BUILD_FLAGS := -ldflags "$(LDFLAGS)"

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# Test
TEST_TMP :=./_output/test

# Tools
TOOLS_BIN := ./_output/tools/bin

export KUBEBUILDER_ASSETS ?=$(TEST_TMP)/kubebuilder/bin
K8S_VERSION ?=1.30.0
KB_TOOLS_ARCHIVE_NAME :=kubebuilder-tools-$(K8S_VERSION)-$(GOHOSTOS)-$(GOHOSTARCH).tar.gz
KB_TOOLS_ARCHIVE_PATH := $(TEST_TMP)/$(KB_TOOLS_ARCHIVE_NAME)

.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)RBAC API Server - Available Commands:$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "$(BLUE)Usage:$(NC)\n  make $(GREEN)<target>$(NC)\n\n$(BLUE)Targets:$(NC)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: all
all: clean fmt vet test build ## Run all checks and build

# Development targets
.PHONY: fmt
fmt: ## Format Go code
	@echo "$(YELLOW)Formatting Go code...$(NC)"
	@$(GOFMT) ./...

.PHONY: generate
generate: ## Generate code (deepcopy, register, client, OpenAPI, etc.)
	@echo "$(YELLOW)Generating code...$(NC)"
	@bash hack/update-codegen.sh

.PHONY: vet
vet: ## Run go vet
	@echo "$(YELLOW)Running go vet...$(NC)"
	@$(GOCMD) vet ./...

.PHONY: lint
lint: ## Run golangci-lint using local binary
	@echo "$(YELLOW)Running golangci-lint...$(NC)"
	@if [ ! -f $(TOOLS_BIN)/golangci-lint ]; then \
		echo "$(YELLOW)Installing golangci-lint...$(NC)"; \
		mkdir -p $(TOOLS_BIN); \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(TOOLS_BIN) latest; \
	fi
	@$(TOOLS_BIN)/golangci-lint run --config golangci.yml --tests=false

.PHONY: test-unit
test-unit: ensure-kubebuilder-tools
	go test -race -coverprofile=$(TEST_TMP)/coverage.out ./pkg/...

.PHONY: build-e2e
build-e2e: ## Build e2e test binary using Ginkgo
	@echo "$(YELLOW)Building e2e test binary...$(NC)"
	@go test -c -o $(TEST_TMP)/e2e.test ./test/e2e/
	@echo "$(GREEN)E2E test binary built: $(TEST_TMP)/e2e.test$(NC)"

.PHONY: run-e2e
run-e2e: ## Run e2e test binary
	@echo "$(YELLOW)Running e2e tests...$(NC)"
	@$(TEST_TMP)/e2e.test -ginkgo.v

.PHONY: test-e2e
test-e2e: build-e2e run-e2e ## Build and run e2e tests

# Build targets
.PHONY: build
build: generate fmt vet build-bin

.PTHONY: build-bin
build-bin:
	@echo "$(YELLOW)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(GOBIN)
	@$(GOBUILD) $(BUILD_FLAGS) -o $(GOBIN)/$(BINARY_NAME) ./cmd
	@echo "$(GREEN)Binary built: $(GOBIN)/$(BINARY_NAME)$(NC)"


.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@$(GOCLEAN)
	@rm -rf $(GOBIN)
	@rm -rf coverage/
	@echo "$(GREEN)Clean completed$(NC)"


.PHONY: build-image
build-image: ## Build Docker image
	@echo "$(YELLOW)Building Docker image $(IMAGE_NAME):$(IMAGE_TAG)"
	@$(DOCKER) build $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "$(GREEN)Docker image built: $(IMAGE_NAME):$(IMAGE_TAG)"

.PHONY: push-image
push-image: ## Push Docker image to registry
	@echo "$(YELLOW)Pushing Docker image $(IMAGE_NAME):$(IMAGE_TAG)"
	@$(DOCKER) push $(IMAGE_NAME):$(IMAGE_TAG)
	@echo "$(GREEN)Docker image pushed$(NC)"


# Info targets
.PHONY: info
info: ## Show project information
	@echo "$(BLUE)Project Information:$(NC)"
	@echo "  Binary: $(BINARY_NAME)"
	@echo "  Image: $(IMAGE_NAME):$(IMAGE_TAG)"
	@echo "  Go Version: $(GO_VERSION)"


.PHONY: check-tools
check-tools: ## Check if required tools are installed
	@echo "$(YELLOW)Checking required tools...$(NC)"
	@command -v go >/dev/null 2>&1 || (echo "$(RED)❌ Go not installed$(NC)" && exit 1)
	@command -v docker >/dev/null 2>&1 || (echo "$(RED)❌ Docker not installed$(NC)" && exit 1)
	@command -v kubectl >/dev/null 2>&1 || (echo "$(RED)❌ kubectl not installed$(NC)" && exit 1)
	@command -v kind >/dev/null 2>&1 || (echo "$(RED)❌ Kind not installed$(NC)" && exit 1)
	@echo "$(GREEN)✅ All required tools are installed$(NC)"


# download the kubebuilder-tools to get kube-apiserver binaries from it
.PHONY: ensure-kubebuilder-tools
ensure-kubebuilder-tools:
ifeq "" "$(wildcard $(KUBEBUILDER_ASSETS))"
	$(info Downloading kube-apiserver into '$(KUBEBUILDER_ASSETS)')
	mkdir -p '$(KUBEBUILDER_ASSETS)'
	curl -s -f -L https://storage.googleapis.com/kubebuilder-tools/$(KB_TOOLS_ARCHIVE_NAME) -o '$(KB_TOOLS_ARCHIVE_PATH)'
	tar -C '$(KUBEBUILDER_ASSETS)' --strip-components=2 -zvxf '$(KB_TOOLS_ARCHIVE_PATH)'
else
	$(info Using existing kube-apiserver from "$(KUBEBUILDER_ASSETS)")
endif


# Default target
.DEFAULT_GOAL := help
