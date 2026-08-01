# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# Copy all module go.mod files for dependency caching
COPY go.work go.sum* ./
COPY api/go.mod api/go.mod
COPY log/go.mod log/go.mod
COPY tools/go.mod tools/go.mod
COPY frame/go.mod frame/go.mod
COPY workflow/go.mod workflow/go.mod
COPY saas/go.mod saas/go.mod

# Download dependencies for all modules
RUN for mod in api log tools frame workflow saas; do \
        (cd "$mod" && go mod download); \
    done

# Copy all source code
COPY api/ api/
COPY log/ log/
COPY tools/ tools/
COPY frame/ frame/
COPY workflow/ workflow/
COPY saas/ saas/

# Verify all library modules compile
RUN for mod in api log tools frame workflow saas; do \
        (cd "$mod" && go build ./...); \
    done

# Build the landc CLI binary (from frame/cmd)
RUN cd frame && CGO_ENABLED=0 go build -o /build/landc ./cmd

# =============================================================================
# Stage 2: Dev image with hot-reload
# =============================================================================
FROM golang:1.24-alpine AS dev

RUN go install github.com/air-verse/air@latest

WORKDIR /app
COPY --from=builder /build /app

CMD ["air"]

# =============================================================================
# Stage 3: Minimal production image
# =============================================================================
FROM alpine:3.21 AS production

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the landc CLI binary
COPY --from=builder /build/landc /usr/local/bin/landc

# Non-root user for security
RUN addgroup -S app && adduser -S app -G app
USER app

# Health check placeholder (override with your application's endpoint)
# HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/health || exit 1

# Default command: show help
ENTRYPOINT ["landc"]
CMD ["--help"]
