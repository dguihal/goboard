.PHONY: all check-go clean build install web_dependencies

MINIMUM_SUPPORTED_GO_MAJOR_VERSION = 1
MINIMUM_SUPPORTED_GO_MINOR_VERSION = 14
GO_VERSION_VALIDATION_ERR_MSG = Golang version is not supported, please update to at least $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION).$(MINIMUM_SUPPORTED_GO_MINOR_VERSION)

EXECUTABLES = go npm
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

GO := $(shell command -v go 2> /dev/null)
NPM := $(shell command -v npm 2> /dev/null)
GO_MAJOR_VERSION:= $(shell $(GO) version | cut -d ' ' -f 3 | cut -d '.' -f 1 | cut -c 3)
GO_MINOR_VERSION:= $(shell $(GO) version | cut -d ' ' -f 3 | cut -d '.' -f 2)

all: web_dependencies build

validate-go-version: ## Validates the installed version of go against minimum requirement.
	@if [ $(GO_MAJOR_VERSION) -gt $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION) ]; then \
		exit 0 ;\
	elif [ $(GO_MAJOR_VERSION) -lt $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION) ]; then \
		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
		exit 1; \
	elif [ $(GO_MINOR_VERSION) -lt $(MINIMUM_SUPPORTED_GO_MINOR_VERSION) ] ; then \
		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
		exit 1; \
	fi

build: validate-go-version
	$(GO) build $(GOFLAGS) ./...

clean:
	$(GO) clean $(GOFLAGS) -i ./...
	cd web ; $(NPM) run-script clean

web_dependencies:
	cd web ; $(NPM) install ; $(NPM) run-script build

docker_image: web_dependencies
	docker build .
