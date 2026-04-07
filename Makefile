BIN_DIR    := bin
APP_SERVER := gophkeeper-server
APP_CLI    := gophkeeper

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")


LDFLAGS := -s -w \
	-X 'main.buildVersion=$(VERSION)' \
	-X 'main.buildDate=$(DATE)' \
	-X 'main.buildCommit=$(COMMIT)'


.PHONY: all
all: build-all

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)

.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go build -trimpath -ldflags "$(LDFLAGS)" \
	-o $(BIN_DIR)/linux_amd64/$(APP_SERVER) ./cmd/server
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go build -trimpath -ldflags "$(LDFLAGS)" \
	-o $(BIN_DIR)/linux_amd64/$(APP_CLI) ./cmd/cli

.PHONY: build-darwin
build-darwin:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 \
	go build -trimpath -ldflags "$(LDFLAGS)" \
	-o $(BIN_DIR)/darwin_arm64/$(APP_SERVER) ./cmd/server
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 \
	go build -trimpath -ldflags "$(LDFLAGS)" \
	-o $(BIN_DIR)/darwin_ard64/$(APP_CLI) ./cmd/cli

.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
	go build -trimpath -ldflags "$(LDFLAGS)" \
	-o $(BIN_DIR)/windows_amd64/$(APP_SERVER).exe ./cmd/server
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
	go build -trimpath -ldflags "$(LDFLAGS)" \
	-o $(BIN_DIR)/windows_amd64/$(APP_CLI).exe ./cmd/cli