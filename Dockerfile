# ---- build stage ----
FROM golang:1.23-alpine AS build
WORKDIR /src

# Cache module downloads separately from source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Static, trimmed, stripped binary. All game content is baked in via go:embed,
# so the result is a single self-contained executable — no data files to ship.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/azurenights ./cmd/rpg

# ---- runtime stage ----
FROM alpine:3.20
RUN adduser -D -h /home/player player
USER player
WORKDIR /home/player
COPY --from=build /out/azurenights /usr/local/bin/azurenights

# It's a TUI: run with `docker run --rm -it`.
ENTRYPOINT ["azurenights"]