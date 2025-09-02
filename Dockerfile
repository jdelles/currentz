# syntax=docker/dockerfile:1
ARG GO_VERSION=1.25

# --- build stage ---
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/currentz ./cmd/currentz \
 && go build -trimpath -ldflags="-s -w" -o /out/server   ./cmd/server

# --- runtime (CLI by default) ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/currentz /usr/local/bin/currentz
COPY --from=build /out/server   /usr/local/bin/server
# Non-root user for safety
RUN adduser -D -H -u 10001 appuser
USER appuser

# The CLI reads DB_URL at runtime
ENTRYPOINT ["/usr/local/bin/currentz"]
