# Build stage
FROM golang:1.23.5-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o randimg ./cmd/server

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /build/randimg .

COPY --from=builder /build/.env.example .env

# Copy static files
COPY --from=builder /build/web/dist ./web/dist

# Create data directory for SQLite database
RUN mkdir -p /app/data

# Set environment variables
# ENV PORT=8080 \
#     DB_PATH=/app/data/randimg.db \
#     GIN_MODE=release

# Expose port
# EXPOSE 8080

# Health check
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#     CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/stats || exit 1

# Run the application
CMD ["./randimg"]
