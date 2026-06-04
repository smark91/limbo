# Tailwind CSS build stage
FROM debian:bookworm-slim AS tailwind
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*
ARG TAILWIND_VERSION=v4.3.0
ARG TARGETARCH
RUN ARCH=$(case "${TARGETARCH}" in amd64) echo "x64" ;; arm64) echo "arm64" ;; *) echo "x64" ;; esac) && \
    curl -fsSL "https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-${ARCH}" -o /usr/local/bin/tailwindcss && \
    chmod +x /usr/local/bin/tailwindcss
WORKDIR /build
COPY frontend/ frontend/
RUN tailwindcss --input frontend/css/tailwind.css --output frontend/css/styles.css --minify

# Build stage
FROM golang:1.26.4-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=tailwind /build/frontend/css/styles.css frontend/css/styles.css

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
ARG VERSION=dev
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
