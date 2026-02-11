# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Set Go proxy for faster downloads (optional but recommended)
ENV GOPROXY=https://proxy.golang.org,direct

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies with verbose output for debugging
RUN go mod download -x || (echo "Failed to download Go modules" && cat go.mod && exit 1)

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata mysql-client

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/server .

# Copy migrations (needed for database initialization)
COPY --from=builder /app/migrations ./migrations

# Expose application port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./server"]
