FROM golang:1.23-alpine

# Install git for fetching dependencies
RUN apk add --no-cache git

# Set necessary environment variables for Go
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the working directory
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Command to run the executable
EXPOSE 8080

# Run the application
CMD ["go", "run", "./cmd/server"]