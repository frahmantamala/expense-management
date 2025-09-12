# Multi-stage build for Go application
FROM golang:1.23.6-alpine AS builder

# Install git and build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Production stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and postgresql-client for migrations
RUN apk --no-cache add ca-certificates postgresql-client

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy configuration files
COPY --from=builder /app/config.example.yml ./config.yml

# Copy database migration files
COPY --from=builder /app/db ./db

# Copy API specification
COPY --from=builder /app/api ./api

# Create startup script
COPY --from=builder /app/scripts/docker-entrypoint.sh ./docker-entrypoint.sh
RUN chmod +x ./docker-entrypoint.sh

# Expose port
EXPOSE 8080

# Set environment variables
ENV GIN_MODE=release

# Use entrypoint script
ENTRYPOINT ["./docker-entrypoint.sh"]
