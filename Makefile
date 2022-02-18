APP = trigger
OUTPUT_DIR ?= _output

CMD = ./cmd/$(APP)/...
PKG = ./pkg/...

BIN ?= $(OUTPUT_DIR)/$(APP)

GO_FLAGS ?= -v -mod=vendor
GO_TEST_FLAGS ?= -race -cover

ARGS ?=

.EXPORT_ALL_VARIABLES:

.PHONY: $(BIN)
$(BIN):
	go build $(GO_FLAGS) -o $(BIN) $(CMD)

build: $(BIN)

default: build

clean:
	rm -rf "$(OUTPUT_DIR)" || true

run:
	go run $(GO_FLAGS) $(CMD) $(ARGS)

test: test-unit test-e2e

.PHONY: test-unit
test-unit:
	go test $(GO_FLAGS) $(GO_TEST_FLAGS) $(CMD) $(PKG) $(ARGS)