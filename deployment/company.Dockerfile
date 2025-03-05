# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o company cmd/company/main.go

# Final Stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/company .
COPY internal/company/config/config.yaml internal/company/config/config.yaml
EXPOSE 50051
CMD ["./company"]