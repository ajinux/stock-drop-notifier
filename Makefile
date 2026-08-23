# Stock Drop Notifier Makefile

# Build variables
BINARY_NAME=notifier
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

# Installation paths
INSTALL_DIR=$(HOME)/bin
CONFIG_DIR=$(HOME)/.stock-notifier
LAUNCHD_PLIST=$(HOME)/Library/LaunchAgents/com.user.stock-notifier.plist

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod

.PHONY: all build test clean run help lint tidy install install-gopath uninstall

# Default target
all: build

## build: Build the application
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/notifier/

## test: Run tests
test:
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## clean: Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

## run: Build and run the application
run: build
	./$(BINARY_NAME) check --verbose

## run-check: Run the check command
run-check: build
	./$(BINARY_NAME) check

## run-list: Run the list command
run-list: build
	./$(BINARY_NAME) list

## lint: Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

## tidy: Tidy Go modules
tidy:
	$(GOMOD) tidy

## install: Build and install for daily launchd automation (~/bin + ~/.stock-notifier)
install: build
	@echo "📦 Installing Stock Drop Notifier for daily automation..."
	@mkdir -p $(INSTALL_DIR)
	@mkdir -p $(CONFIG_DIR)
	@cp $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✅ Binary installed to $(INSTALL_DIR)/$(BINARY_NAME)"
	@if [ ! -f .env ]; then \
		echo "❌ .env not found. Run 'cp .env.example .env' and add your API key first."; \
		exit 1; \
	fi
	@if [ ! -f alerts.yaml ]; then \
		echo "❌ alerts.yaml not found. Run 'cp alerts.yaml.example alerts.yaml' first."; \
		exit 1; \
	fi
	@cp .env $(CONFIG_DIR)/.env
	@cp alerts.yaml $(CONFIG_DIR)/alerts.yaml
	@echo "✅ Configs copied to $(CONFIG_DIR)/"
	@if [ -f "$(LAUNCHD_PLIST)" ]; then \
		launchctl unload $(LAUNCHD_PLIST) 2>/dev/null || true; \
		launchctl load $(LAUNCHD_PLIST) || exit 1; \
		echo "🔄 Reloaded launchd agent"; \
	else \
		echo "⚠️  Warning: launchd agent not found at $(LAUNCHD_PLIST)"; \
		echo "   Daily automation not configured. Binary and configs installed successfully."; \
	fi
	@echo ""
	@echo "🎉 Installation complete!"
	@echo ""
	@echo "📍 Installed files:"
	@echo "   Binary:  $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo "   Config:  $(CONFIG_DIR)/.env"
	@echo "   Alerts:  $(CONFIG_DIR)/alerts.yaml"
	@echo ""
	@echo "🧪 Test installation:"
	@echo "   $(INSTALL_DIR)/$(BINARY_NAME) --version"
	@echo "   $(INSTALL_DIR)/$(BINARY_NAME) test-telegram"

## install-gopath: Install the binary to GOPATH/bin (standard Go installation)
install-gopath:
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) ./cmd/notifier/
	@echo "✅ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

## uninstall: Remove installed binary and stop launchd agent (keeps configs)
uninstall:
	@echo "🗑️  Uninstalling Stock Drop Notifier..."
	@if [ -f "$(LAUNCHD_PLIST)" ]; then \
		launchctl unload $(LAUNCHD_PLIST) 2>/dev/null || true; \
		echo "🛑 Unloaded launchd agent"; \
	fi
	@if [ -f "$(INSTALL_DIR)/$(BINARY_NAME)" ]; then \
		rm -f $(INSTALL_DIR)/$(BINARY_NAME); \
		echo "✅ Removed $(INSTALL_DIR)/$(BINARY_NAME)"; \
	fi
	@echo ""
	@echo "⚠️  Config files preserved at $(CONFIG_DIR)/"
	@echo "   To remove configs: rm -rf $(CONFIG_DIR)"
	@echo "   To remove launchd agent: rm -f $(LAUNCHD_PLIST)"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/ /'
