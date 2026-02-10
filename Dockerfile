# Multi-stage build for minimal image size

# Stage 1: Builder
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata nodejs npm

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build frontend
RUN npm --prefix frontend install && npm --prefix frontend run build
RUN rm -rf internal/web/dist && mkdir -p internal/web && cp -r frontend/dist internal/web/dist

# Build binary
ARG VERSION=1.0.0
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" \
    -o /build/moxapp \
    ./cmd/moxapp

# Stage 2: Runtime
FROM scratch

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /build/moxapp /moxapp

# Copy config
COPY configs/endpoints.yaml /configs/endpoints.yaml

# Expose API port
EXPOSE 8080

# Set timezone
ENV TZ=UTC

# Run
ENTRYPOINT ["/moxapp"]
CMD ["--config", "/configs/endpoints.yaml"]
