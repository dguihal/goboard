.PHONY: all check-go clean build install web_dependencies

MIN_GOVERSION = "1.14"

EXECUTABLES = go npm
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

GO := $(shell command -v go 2> /dev/null)
NPM := $(shell command -v npm 2> /dev/null)
GOVERSION := $(shell $(GO) version | egrep -o '[0-9.]*' | head -1 | cut -d '.' -f 1-2)
GOVERSION_MAX:=$(shell echo "$(GOVERSION)\n$(MIN_GOVERSION)" | sort -V | tail -1)

ifneq "$(GOVERSION_MAX)" "$(GOVERSION)"
    $(error Go version >= $(MIN_GOVERSION) required, $(GOVERSION) found)
endif

all: install

build:
	$(GO) build $(GOFLAGS) ./...

clean:
	$(GO) clean $(GOFLAGS) -i ./...
	cd web ; $(NPM) run-script clean

web_dependencies:
	cd web ; $(NPM) install ; $(NPM) run-script build

docker_image: web_dependencies
	docker build .
