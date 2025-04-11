# Use a multi-stage build for a smaller final image
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o nexusprobe ./cmd/blc

# Use Chrome in a distroless container for the final image
FROM zenika/alpine-chrome:latest

# Add a non-root user
RUN addgroup -S app && adduser -S -g app app

# Copy the binary from the builder stage
COPY --from=builder /app/nexusprobe /usr/local/bin/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Create directories for app data
RUN mkdir -p /app/reports && chown -R app:app /app

WORKDIR /app

# Switch to non-root user
USER app

# Expose the API port
EXPOSE 8080

# Set the entrypoint
ENTRYPOINT ["nexusprobe"]

# Command can be overridden at runtime
CMD ["--port", "8080"]