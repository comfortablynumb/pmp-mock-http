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

# Create mocks directory
RUN mkdir -p /app/mocks

# Expose default port
EXPOSE 8083

# Run the application
ENTRYPOINT ["/app/pmp-mock-http"]
CMD ["--mocks-dir", "/app/mocks"]
