# syntax=docker/dockerfile:1.7

# Multi-stage build for minimal image size

# Stage 1: Frontend builder
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder

WORKDIR /build/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Go builder
FROM --platform=$BUILDPLATFORM golang:1.25.6-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG BUILD_TIME

WORKDIR /build

# Install dependencies
RUN apk add --no-cache ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy frontend build
RUN rm -rf internal/web/dist && mkdir -p internal/web
COPY --from=frontend-builder /build/frontend/dist internal/web/dist

# Build binary
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o /build/moxapp \
    ./cmd/moxapp

# Stage 2: Runtime
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /build/moxapp /moxapp

# Expose API port
EXPOSE 8080

# Run
ENTRYPOINT ["/moxapp"]
CMD ["-y"]
