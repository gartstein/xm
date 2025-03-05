# Use Go as the base image
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files first, then download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the authentication service
RUN go build -o auth-service cmd/authentication/main.go

# Use a minimal base image for deployment
FROM alpine:latest

WORKDIR /root/

# Copy the binary from the builder
COPY --from=builder /app/auth-service .

# Set environment variables
ENV AUTH_SERVICE_PORT=8081
ENV JWT_SECRET=jwt_secret

# Expose port
EXPOSE 8081

# Run the service
CMD ["./auth-service"]