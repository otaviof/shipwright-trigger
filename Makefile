APP = trigger
OUTPUT_DIR ?= _output

CMD = ./cmd/$(APP)/...
PKG = ./pkg/...

BIN ?= $(OUTPUT_DIR)/$(APP)

KO_DOCKER_REPO ?= ghcr.io/otaviof
KO_DEPLOY_DIR ?= deploy/

GOFLAGS ?= -v -mod=vendor
GOFLAGS_TEST ?= -race -cover

ARGS ?=

.EXPORT_ALL_VARIABLES:

.PHONY: $(BIN)
$(BIN):
	go build -o $(BIN) $(CMD)

build: $(BIN)

default: build

build-image:
	ko publish --base-import-paths $(CMD)

install:
	ko apply --base-import-paths --recursive --filename $(KO_DEPLOY_DIR)

clean:
	rm -rf "$(OUTPUT_DIR)" || true

run:
	go run $(CMD) $(ARGS)

test: test-unit test-e2e

.PHONY: test-unit
test-unit:
	go test $(GOFLAGS_TEST) $(CMD) $(PKG) $(ARGS)

install-tekton:
	./hack/install-tekton.sh