BINARY  := azurenights
PKG     := ./...
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: run build test vet fmt fmtcheck balance tidy clean docker-build docker-run snapshot ci

run:
	go run ./cmd/rpg

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/rpg

test:
	go test $(PKG) -race -count=1

vet:
	go vet $(PKG)

fmt:
	gofmt -w .

fmtcheck:
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:"; gofmt -l .; exit 1)

balance:
	go run ./cmd/balance

tidy:
	go mod tidy

clean:
	rm -rf bin dist

docker-build:
	docker build -t $(BINARY) .

docker-run:
	docker run --rm -it $(BINARY)

# Dry-run the release build locally (requires goreleaser).
snapshot:
	goreleaser release --snapshot --clean

ci: fmtcheck vet test build