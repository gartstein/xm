APP_NAME         := company-service
CMD_DIR          := cmd/server
BIN_DIR          := bin
BUILD_OUTPUT     := $(BIN_DIR)/$(APP_NAME)

# Protobuf generation settings
PROTO_FILES      := $(wildcard $(PROTO_DIR)/*.proto)
PROTO_DIR        := api
PROTO_OUT_DIR	 := api/gen

.PHONY: proto build test docker-build docker-run clean lint help integration-test

# Default target
.DEFAULT_GOAL := help

## 🔄 Generate Go stubs from .proto files. Fetch and update module dependencies, then generate Go code from .proto files.
proto:
	cd $(PROTO_DIR) && buf dep update
	cd $(PROTO_DIR) && buf generate

## 🔍 Lint your protobuf definitions using Buf.
proto-lint:
	cd $(PROTO_DIR) && buf lint

## 🧹 Clean generated files.
proto-clean:
	rm -rf $(PROTO_OUT_DIR)

## 🛠️ Run linter to check for issues.
lint:
	golangci-lint run --config config/.golangci.yaml

## 🔨 Build the Go binary.
build:
	mkdir -p $(BIN_DIR)
	go build -o $(BUILD_OUTPUT) ./$(CMD_DIR)

## 🧪 Run unit tests.
test:
	go test ./internal/company/auth ./internal/company/controller ./internal/company/db ./internal/company/events ./internal/company/handlers

## 🔗 Run integration tests.
integration-test:
	@echo "🚀 Running integration tests..."
	@docker-compose -f internal/company/test/docker-compose.yaml up -d postgres kafka zookeeper  # Ensure dependencies are running
	@sleep 5  # Wait for services to be ready
	@DATABASE_URL="postgres://test:test@localhost:5432/test?sslmode=disable" \
	go test -v ./internal/company/test -tags=integration
	@docker-compose -f internal/company/test/docker-compose.yaml down  # Clean up services after tests

## 🛑 Stop all integration test Docker containers
stop-integration-dockers:
	@echo "🛑 Stopping integration test Docker containers..."
	@docker-compose -f internal/company/test/docker-compose.yaml down
	@echo "✅ All integration test containers stopped."

## 🚀 Delete all messages in Kafka topics (reset offsets)
clear-kafka-messages:
	@echo "🗑️ Clearing Kafka messages..."
	@docker exec -it $(docker ps --filter name=kafka --format "{{.ID}}") \
	  kafka-topics.sh --bootstrap-server localhost:9092 --delete --topic company.created || true
	@docker exec -it $(docker ps --filter name=kafka --format "{{.ID}}") \
	  kafka-topics.sh --bootstrap-server localhost:9092 --create --topic company.created --partitions 1 --replication-factor 1
	@echo "✅ Kafka messages cleared!"

## 🐳 Build a Docker image.
docker-build:
	docker build -t $(APP_NAME):latest .

## 🚀 Run services locally via Docker Compose (Postgres, Kafka, gRPC service, etc.).
docker-run:
	cd deployment && docker-compose up --build

docker-stop:
	cd deployment && docker-compose down

## 🗑️ Clean up local build artifacts.
clean:
	rm -rf $(BIN_DIR)

## 📌 Show help message listing available Makefile commands.
help:
	@echo "Available commands:"
	@awk '/^## /{help=$$0; sub(/^## /,"",help); next} /^[a-zA-Z0-9_-]+:/ && help { \
	  split($$1, target, ":"); \
	  printf "\033[36m%-20s\033[0m %s\n", target[1], help; \
	  help=""; \
	}' $(MAKEFILE_LIST)