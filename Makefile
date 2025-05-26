# Makefile for WebService Project

# 变量定义
APP_NAME=webservice
VERSION=1.0.0
GO_VERSION=1.21
DOCKER_IMAGE=$(APP_NAME):$(VERSION)
DOCKER_REGISTRY=your-registry.com

# Go相关变量
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# 构建相关变量
BINARY_NAME=main
BINARY_UNIX=$(BINARY_NAME)_unix
BUILD_DIR=build
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(shell date +%Y-%m-%d_%H:%M:%S)"

# 默认目标
.PHONY: all
all: clean deps fmt lint test build

# 安装依赖
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# 代码格式化
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# 代码检查
.PHONY: lint
lint:
	@echo "Running linter..."
	$(GOLINT) run

# 运行测试
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# 测试覆盖率
.PHONY: coverage
coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 构建应用
.PHONY: build
build:
	@echo "Building application..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) -v

# 构建Linux版本
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX) -v

# 构建多平台版本
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	# Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 -v
	# macOS
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 -v
	# Windows
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe -v

# 运行应用
.PHONY: run
run:
	@echo "Running application..."
	$(GOCMD) run main.go

# 开发模式运行（使用air热重载）
.PHONY: dev
dev:
	@echo "Running in development mode..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not found. Installing..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# 清理构建文件
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Docker相关命令
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --name $(APP_NAME) $(DOCKER_IMAGE)

.PHONY: docker-stop
docker-stop:
	@echo "Stopping Docker container..."
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true

.PHONY: docker-push
docker-push:
	@echo "Pushing Docker image..."
	docker tag $(DOCKER_IMAGE) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)

# Docker Compose相关命令
.PHONY: compose-up
compose-up:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

.PHONY: compose-down
compose-down:
	@echo "Stopping services with Docker Compose..."
	docker-compose down

.PHONY: compose-logs
compose-logs:
	@echo "Showing Docker Compose logs..."
	docker-compose logs -f

.PHONY: compose-restart
compose-restart:
	@echo "Restarting services with Docker Compose..."
	docker-compose restart

# 数据库相关命令
.PHONY: db-migrate
db-migrate:
	@echo "Running database migrations..."
	$(GOCMD) run main.go migrate

.PHONY: db-seed
db-seed:
	@echo "Seeding database..."
	$(GOCMD) run main.go seed

.PHONY: db-reset
db-reset:
	@echo "Resetting database..."
	$(GOCMD) run main.go reset

# 代码生成
.PHONY: generate
generate:
	@echo "Running code generation..."
	$(GOCMD) generate ./...

# 安装开发工具
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOCMD) install github.com/cosmtrek/air@latest
	$(GOCMD) install github.com/swaggo/swag/cmd/swag@latest

# 生成API文档
.PHONY: docs
docs:
	@echo "Generating API documentation..."
	@if command -v swag > /dev/null; then \
		swag init; \
	else \
		echo "Swag not found. Installing..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
		swag init; \
	fi

# 性能测试
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# 安全检查
.PHONY: security
security:
	@echo "Running security checks..."
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "Gosec not found. Installing..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

# 检查更新
.PHONY: check-updates
check-updates:
	@echo "Checking for dependency updates..."
	$(GOCMD) list -u -m all

# 更新依赖
.PHONY: update-deps
update-deps:
	@echo "Updating dependencies..."
	$(GOCMD) get -u ./...
	$(GOMOD) tidy

# 帮助信息
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  all           - Run clean, deps, fmt, lint, test, build"
	@echo "  deps          - Install dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  test          - Run tests"
	@echo "  coverage      - Generate test coverage report"
	@echo "  build         - Build application"
	@echo "  build-linux   - Build for Linux"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  run           - Run application"
	@echo "  dev           - Run in development mode with hot reload"
	@echo "  clean         - Clean build files"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  docker-stop   - Stop Docker container"
	@echo "  compose-up    - Start services with Docker Compose"
	@echo "  compose-down  - Stop services with Docker Compose"
	@echo "  install-tools - Install development tools"
	@echo "  docs          - Generate API documentation"
	@echo "  security      - Run security checks"
	@echo "  help          - Show this help message"