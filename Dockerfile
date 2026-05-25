# Build stage
FROM golang:1.26.3-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .

# Test stage (runs unit tests with caching)
FROM builder AS tester
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 go test -v ./...

# Compiler stage (compiles binary)
FROM builder AS compiler
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=1 go build -o limbo -ldflags="-s -w" .

# Runtime stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget
COPY --from=compiler /build/limbo /usr/local/bin/limbo

# OpenContainers metadata labels
ARG VERSION=1.0.0
ARG REVISION=unknown
LABEL org.opencontainers.image.title="Limbo" \
      org.opencontainers.image.description="Seerr - Unfulfilled Request Dashboard & Notifier" \
      org.opencontainers.image.licenses="AGPL-3.0-only" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.source="https://github.com/smark91/limbo"

EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:3000/api/health | grep '"status":"ok"' || exit 1
CMD ["limbo"]
