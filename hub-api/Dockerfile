# Start from the official Go image to build your application
FROM golang:1.22.2 as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the workspace
COPY cmd/webapi .
COPY internal ./internal
COPY util ./util
COPY go.mod go.sum ./

# Download all the dependencies
RUN go mod download

# Copy the rest of the source code
COPY .. ./

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o opc-ua-time-series-hub-api

# Start a new stage from Debian slim to run the application
FROM debian:bullseye-slim

# Install ca-certificates
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/opc-ua-time-series-hub-api .

# Command to run the executable
CMD ["/app/opc-ua-time-series-hub-api"]