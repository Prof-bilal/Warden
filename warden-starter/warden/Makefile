GO      ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X github.com/warden-sandbox/warden/internal/version.Version=$(VERSION)"

.PHONY: build build-all test vet fmt clean

build:
	CGO_ENABLED=0 GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) $(GO) build $(LDFLAGS) -o warden ./cmd/warden

build-all:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/warden-linux-amd64   ./cmd/warden
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 $(GO) build $(LDFLAGS) -o dist/warden-darwin-arm64  ./cmd/warden
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o dist/warden-windows-amd64.exe ./cmd/warden

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w .

clean:
	rm -f warden
	rm -rf dist/
	rm -f *.test *.out
