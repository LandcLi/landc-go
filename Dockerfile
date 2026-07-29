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

# Build your application (replace with your main package path)
# Example: build a binary using the frame module
# COPY cmd/ cmd/
# RUN CGO_ENABLED=0 go build -o /build/app ./cmd/app

# For library modules, build all packages to verify compilation
RUN for mod in api log tools frame workflow saas; do \
        (cd "$mod" && go build ./...); \
    done

# =============================================================================
# Stage 2: Dev image with hot-reload (optional)
# =============================================================================
FROM golang:1.24-alpine AS dev

RUN go install github.com/air-verse/air@latest

WORKDIR /app
COPY --from=builder /build /app

CMD ["air"]

# =============================================================================
# Stage 3: Minimal production image (uncomment when you have a main package)
# =============================================================================
# FROM alpine:3.21 AS production
#
# RUN apk add --no-cache ca-certificates tzdata
#
# WORKDIR /app
# COPY --from=builder /build/app .
#
# EXPOSE 8080
#
# ENTRYPOINT ["./app"]
