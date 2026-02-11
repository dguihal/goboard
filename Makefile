.DEFAULT_GOAL := help

.PHONY: all build clean install web_dependencies docker_image validate-go-version help

# Build flags can be overridden from the command line.
# e.g. make build GOFLAGS="-ldflags=-s"
GOFLAGS ?= -v

MINIMUM_SUPPORTED_GO_MAJOR_VERSION = 1
MINIMUM_SUPPORTED_GO_MINOR_VERSION = 14
GO_VERSION_VALIDATION_ERR_MSG = "Golang version is not supported, please update to at least $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION).$(MINIMUM_SUPPORTED_GO_MINOR_VERSION)"

# Check for required executables at parse time.
EXECUTABLES = go npm docker
$(foreach exec,$(EXECUTABLES),$(if $(shell which $(exec)),,$(error "Required executable '$(exec)' not found in PATH")))

GO := $(shell command -v go 2> /dev/null)
NPM := $(shell command -v npm 2> /dev/null)

# Parse Go version in a robust way.
# This extracts major and minor version numbers from `go version` output.
GO_VERSIONS := $(shell $(GO) version | sed -n 's/.* go\([0-9]*\)\.\([0-9]*\).*/\1 \2/p')
GO_MAJOR_VERSION := $(word 1, $(GO_VERSIONS))
GO_MINOR_VERSION := $(word 2, $(GO_VERSIONS))

all: web_dependencies build ## Build everything

validate-go-version: ## Validates the installed version of go against minimum requirement.
	@if [ -z "$(GO_MAJOR_VERSION)" ]; then \
		echo "Error: Could not parse Go version from 'go version' command." >&2; \
		exit 1; \
	fi
	@if [ "$(GO_MAJOR_VERSION)" -lt "$(MINIMUM_SUPPORTED_GO_MAJOR_VERSION)" ] || \
	   ( [ "$(GO_MAJOR_VERSION)" -eq "$(MINIMUM_SUPPORTED_GO_MAJOR_VERSION)" ] && [ "$(GO_MINOR_VERSION)" -lt "$(MINIMUM_SUPPORTED_GO_MINOR_VERSION)" ] ); then \
		echo $(GO_VERSION_VALIDATION_ERR_MSG) >&2; \
		exit 1; \
	fi
	@echo "Go version check passed: $(GO_MAJOR_VERSION).$(GO_MINOR_VERSION)"

build: validate-go-version ## Build the Go application
	$(GO) build $(GOFLAGS) .

install: ## Install the Go application
	$(GO) install .

clean: ## Clean up build artifacts
	$(GO) clean $(GOFLAGS)
	@if [ -d "web/node_modules" ]; then cd web && $(NPM) run-script clean; fi

web_dependencies: ## Install and build web assets
	cd web ; $(NPM) install ; $(NPM) run-script build

docker_image: web_dependencies ## Build the Docker image
	docker build .

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
