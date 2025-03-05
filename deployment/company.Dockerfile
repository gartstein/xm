# Dockerfile
FROM golang:1.20-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main ./cmd/server

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 50051
CMD ["./main"]