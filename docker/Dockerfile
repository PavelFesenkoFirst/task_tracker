FROM golang:1.24-alpine AS builder

ARG SERVICE=api

# Install git and ca-certificates for fetching dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set necessary environment variables for Go
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the working directory
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the selected service binary (api or worker)
RUN go build -o main ./cmd/${SERVICE}

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app
USER appuser

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
