.PHONY: all test lint build clean fmt vet coverage tidy help

MODULES = log tools api frame workflow
ROOT_DIR = $(shell pwd)

all: fmt vet test build

test: ## Run tests for all modules with race detection
	@for mod in $(MODULES); do \
		echo "=== Testing $$mod ==="; \
		cd "$(ROOT_DIR)/$$mod" && go test -race -count=1 -coverprofile=coverage.out ./... 2>&1; \
		cd "$(ROOT_DIR)"; \
	done

lint: ## Run golangci-lint at project root
	golangci-lint run ./...

fmt: ## Format all Go code
	@for mod in $(MODULES); do \
		echo "=== Formatting $$mod ==="; \
		cd "$(ROOT_DIR)/$$mod" && go fmt ./... && cd "$(ROOT_DIR)"; \
	done

vet: ## Run go vet for all modules
	@for mod in $(MODULES); do \
		echo "=== Vetting $$mod ==="; \
		cd "$(ROOT_DIR)/$$mod" && go vet ./... 2>&1; \
		cd "$(ROOT_DIR)"; \
	done

build: ## Build all modules
	@for mod in $(MODULES); do \
		echo "=== Building $$mod ==="; \
		cd "$(ROOT_DIR)/$$mod" && go build ./... 2>&1; \
		cd "$(ROOT_DIR)"; \
	done

tidy: ## Run go mod tidy for all modules
	@for mod in $(MODULES); do \
		echo "=== Tidying $$mod ==="; \
		cd "$(ROOT_DIR)/$$mod" && go mod tidy && cd "$(ROOT_DIR)"; \
	done

coverage: test ## Show test coverage
	@for mod in $(MODULES); do \
		if [ -f "$(ROOT_DIR)/$$mod/coverage.out" ]; then \
			echo "=== Coverage $$mod ==="; \
			cd "$(ROOT_DIR)/$$mod" && go tool cover -func=coverage.out | tail -1 && cd "$(ROOT_DIR)"; \
		fi; \
	done

clean: ## Clean build artifacts and coverage files
	@for mod in $(MODULES); do \
		rm -f "$(ROOT_DIR)/$$mod/coverage.out"; \
	done

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
