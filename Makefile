BINARY_NAME := firestick_debloat
CMD_PKG     := ./cmd/firestick_debloat
DIST_DIR    := dist
LDFLAGS     := -s -w

.PHONY: all build test clean release

all: test build

## build: Compile binary for the current OS/Arch
build:
	go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) $(CMD_PKG)

## test: Run all unit tests
test:
	go test -v -race ./...

## release: Cross-compile binaries for Linux, macOS, and Windows into dist/
release: clean
	mkdir -p $(DIST_DIR)
	GOOS=linux   GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PKG)
	GOOS=linux   GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PKG)
	GOOS=darwin  GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PKG)
	GOOS=darwin  GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PKG)
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PKG)

## clean: Remove compiled binaries and dist directory
clean:
	rm -f $(BINARY_NAME)
	rm -rf $(DIST_DIR)
