# Use the official Golang image to build the binary
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o filo .

# Use a minimal alpine image for the final container
FROM alpine:latest
WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/filo .

# Expose the P2P port (e.g., 3000) and the Metrics port (8081)
EXPOSE 3000
EXPOSE 8081

# Run the application
CMD ["./filo"]