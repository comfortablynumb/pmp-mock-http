# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o pmp-mock-http ./cmd/server

# Runtime stage
FROM alpine:latest

# Install git and ca-certificates
RUN apk --no-cache add git ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/pmp-mock-http .

# Create mocks and plugins directories
RUN mkdir -p /mocks /plugins

# Set environment variables
ENV MOCKS_DIR=/mocks
ENV PLUGINS_DIR=/plugins
ENV PORT=8080
ENV UI_PORT=8081

# Expose default ports
EXPOSE 8080 8081

# Run the application
ENTRYPOINT ["/app/pmp-mock-http"]
