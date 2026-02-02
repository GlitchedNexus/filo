FROM golang:1.25-alpine AS builder

# Install git (required for fetching some Go modules)
RUN apk add --no-cache git

WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build a statically linked binary (better for Alpine)
RUN CGO_ENABLED=0 GOOS=linux go build -o filo .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /root/

COPY --from=builder /app/filo .

# Match your .env LISTEN_ADDR and Prometheus port
EXPOSE 4000
EXPOSE 8081

CMD ["./filo"]