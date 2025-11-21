# Simple Makefile for a Go project

DOCKER_USERNAME=duy0207
IMAGE_NAME=stockmind
IMAGE_TAG=latest

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_USERNAME)/$(IMAGE_NAME):$(IMAGE_TAG) .

# Push image to Docker Hub
docker-push: docker-build
	@echo "Pushing image to Docker Hub..."
	docker push $(DOCKER_USERNAME)/$(IMAGE_NAME):$(IMAGE_TAG)

# Build and push in one command
docker-deploy: docker-build docker-push
	@echo "Deployed $(DOCKER_USERNAME)/$(IMAGE_NAME):$(IMAGE_TAG) to Docker Hub"

# Build the application
all: build test

build:
	@echo "Building..."	
	
	@go build -o main cmd/main.go

# Run the application
run:
	@go run cmd/main.go server &
	@npm install --prefer-offline --no-fund --prefix ./frontend
	@npm run dev --prefix ./frontend

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v
# Integrations Tests for the application
itest:
	@echo "Running integration tests..."
	@go test ./internal/database -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Live Reload
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build run test clean watch itest docker-build docker-push docker-deploy
