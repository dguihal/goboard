.PHONY: all check-go clean build install web_dependencies

EXECUTABLES = go npm
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

GO := $(shell command -v go 2> /dev/null)
NPM := $(shell command -v npm 2> /dev/null)


all: install

check-go:
	GO_VER := $(shell $(GO) version | egrep -o '[0-9.]*' | head -1 | cut -d '.' -f 1-2)
	GO_VER_GTE15 := $(shell echo "$(GO_VER)" \>= 1.5 | bc)
	ifeq "$(GO_VER_GTE15)" "0"
		$(error Go version >= 1.5 required, $(GO_VER) found)
	endif

build: check-go
	$(GO) build $(GOFLAGS) ./...

install: check-go
	$(GO) get $(GOFLAGS) ./...

clean:
	$(GO) clean $(GOFLAGS) -i ./...

web_dependencies:
	cd web ; npm install ; npm run-script build

docker_image: web_dependencies
	docker build .
