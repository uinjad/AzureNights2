BINARY := azurenights
PKG := ./...

.PHONY: run build test vet fmt fmtcheck balance tidy clean docker-build docker-run ci

run:
	go run ./cmd/rpg

build:
	go build -trimpath -ldflags="-s -w" -o bin/$(BINARY) ./cmd/rpg

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
	rm -rf bin

docker-build:
	docker build -t $(BINARY) .

docker-run:
	docker run --rm -it $(BINARY)

# What CI runs.
ci: fmtcheck vet test build