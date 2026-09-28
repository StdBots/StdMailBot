.PHONY: all build run test clean docker-build docker-up docker-down

BINARY_NAME=bin/bot
MAIN_PATH=./cmd/bot

all: build

build:
	@echo "Building StdMailBot..."
	go build -ldflags="-w -s" -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BINARY_NAME)"

run: build
	@echo "Starting StdMailBot..."
	./$(BINARY_NAME)

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning up binaries..."
	rm -rf bin/

docker-build:
	@echo "Building Docker image..."
	docker build -t stdmailbot:latest .

docker-up:
	@echo "Starting stack with Docker Compose..."
	docker-compose up -d

docker-down:
	@echo "Stopping stack..."
	docker-compose down

docker-logs:
	docker-compose logs -f stdmailbot
