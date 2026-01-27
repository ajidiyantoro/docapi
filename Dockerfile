# Use Golang 1.22-alpine as the base image for the build stage
# Alpine is used to keep the image size small
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install git if needed for downloading certain dependencies
RUN apk add --no-cache git

# Copy module definition files to download dependencies first
# This leverages Docker layer caching to speed up the build process if dependencies don't change
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire source code into the container
COPY . .

# Build the application as a static binary (CGO_ENABLED=0) for Linux architecture
# The output binary is named 'docapi' and pointed to the entry point cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o docapi ./cmd/api/main.go

# Runtime stage uses a very lightweight Alpine image
FROM alpine:latest

# Install basic packages required at runtime:
# - tzdata: for time zone settings
# - ca-certificates: to support HTTPS connections (e.g., to MinIO/S3)
RUN apk --no-cache add tzdata ca-certificates

# Create a non-root user and group for application security
RUN addgroup -S apps && adduser -S apps -G apps

# Set the application's working directory
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/docapi .

# Give the non-root user ownership of the binary
RUN chown apps:apps /app/docapi

# Run the container using the non-root user
USER apps

# Expose the port used by the application (default 8080)
EXPOSE 8080

# Command to run the application
CMD ["./docapi"]
