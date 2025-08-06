# Makefile for ECR Deploy Tool

# Variables
APP_NAME := pg-schema-migrate
VERSION := 1.0.0
BUILD_DIR := build
DIST_DIR := dist

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Build flags
LDFLAGS := -ldflags="-s -w"

# Default target
.PHONY: all
all: clean deps build

# Install dependencies
.PHONY: deps
deps:
	@echo "📥 Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Clean build artifacts
.PHONY: clean
clean:
	@echo "🧹 Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -f $(APP_NAME)

# Build for current platform
.PHONY: build
build:
	@echo "🏗️  Building $(APP_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(APP_NAME) .

# Run the application
.PHONY: run
run:
	@echo "🚀 Running $(APP_NAME)..."
	$(GOCMD) run main.go

# Test the application
.PHONY: test
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

# Build for all platforms
.PHONY: build-all
build-all: clean
	@echo "🏗️  Building for all platforms..."
	mkdir -p $(BUILD_DIR)

	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .

	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 .

	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .

	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .

	# macOS ARM64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .

	@echo "✅ Built all platforms in $(BUILD_DIR)/"

# Create distribution packages
.PHONY: dist
dist: build-all
	@echo "📦 Creating distribution packages..."
	mkdir -p $(DIST_DIR)

	# Linux packages
	$(MAKE) dist-linux-amd64
	$(MAKE) dist-linux-arm64

	# Windows package
	$(MAKE) dist-windows

	# macOS packages
	$(MAKE) dist-darwin-amd64
	$(MAKE) dist-darwin-arm64

	# Create checksums
	cd $(DIST_DIR) && sha256sum *.deb *.tar.gz *.zip > checksums.txt

	@echo "✅ Distribution packages created in $(DIST_DIR)/"

# Linux distribution for AMD64
.PHONY: dist-linux-amd64
dist-linux-amd64:
	@echo "📦 Creating Linux AMD64 package..."
	mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/usr/local/bin
	mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/DEBIAN

	cp $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/usr/local/bin/$(APP_NAME)
	chmod +x $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/usr/local/bin/$(APP_NAME)

	# Create control file
	@echo "Package: $(APP_NAME)" > $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/DEBIAN/control
	@echo "Version: $(VERSION)" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/DEBIAN/control
	@echo "Section: utils" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/DEBIAN/control
	@echo "Priority: optional" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/DEBIAN/control
	@echo "Architecture: amd64" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/DEBIAN/control
	@echo "Depends: docker.io, awscli" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/DEBIAN/control
	@echo "Maintainer: Your Team <team@company.com>" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/DEBIAN/control
	@echo "Description: ECR Deploy Tool" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/DEBIAN/control
	@echo " A tool to automate ECR repository creation and Docker image deployment" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64/DEBIAN/control

	# Create .deb package
	dpkg-deb --build $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64 $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64.deb

	# Create tar.gz
	cd $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64 && tar -czf ../$(APP_NAME)-$(VERSION)-linux-amd64.tar.gz usr/

# Linux distribution for ARM64
.PHONY: dist-linux-arm64
dist-linux-arm64:
	@echo "📦 Creating Linux ARM64 package..."
	mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/usr/local/bin
	mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/DEBIAN

	cp $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/usr/local/bin/$(APP_NAME)
	chmod +x $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/usr/local/bin/$(APP_NAME)

	# Create control file
	@echo "Package: $(APP_NAME)" > $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/DEBIAN/control
	@echo "Version: $(VERSION)" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/DEBIAN/control
	@echo "Section: utils" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/DEBIAN/control
	@echo "Priority: optional" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/DEBIAN/control
	@echo "Architecture: arm64" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/DEBIAN/control
	@echo "Depends: docker.io, awscli" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/DEBIAN/control
	@echo "Maintainer: Your Team <team@company.com>" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/DEBIAN/control
	@echo "Description: ECR Deploy Tool" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/DEBIAN/control
	@echo " A tool to automate ECR repository creation and Docker image deployment" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64/DEBIAN/control

	# Create .deb package
	dpkg-deb --build $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64 $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64.deb

	# Create tar.gz
	cd $(DIST_DIR)/$(APP_NAME)-$(VERSION)-linux-arm64 && tar -czf ../$(APP_NAME)-$(VERSION)-linux-arm64.tar.gz usr/

# Windows distribution
.PHONY: dist-windows
dist-windows:
	@echo "📦 Creating Windows package..."
	mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64

	cp $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/$(APP_NAME).exe

	# Create install script
	@echo "@echo off" > $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/install.bat
	@echo "echo Installing ECR Deploy Tool..." >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/install.bat
	@echo "if not exist \"C:\\Program Files\\ECRDeploy\" mkdir \"C:\\Program Files\\ECRDeploy\"" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/install.bat
	@echo "copy /Y $(APP_NAME).exe \"C:\\Program Files\\ECRDeploy\\\"" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/install.bat
	@echo "setx /M PATH \"%%PATH%%;C:\\Program Files\\ECRDeploy\"" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/install.bat
	@echo "echo Installation completed!" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/install.bat
	@echo "pause" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64/install.bat

	# Create zip
	cd $(DIST_DIR) && zip -r $(APP_NAME)-$(VERSION)-windows-amd64.zip $(APP_NAME)-$(VERSION)-windows-amd64/

# macOS distribution for AMD64
.PHONY: dist-darwin-amd64
dist-darwin-amd64:
	@echo "📦 Creating macOS AMD64 package..."
	mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64

	cp $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64/$(APP_NAME)
	chmod +x $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64/$(APP_NAME)

	# Create install script
	@echo "#!/bin/bash" > $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64/install.sh
	@echo "echo \"Installing ECR Deploy Tool...\"" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64/install.sh
	@echo "if command -v brew &> /dev/null; then" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64/install.sh
	@echo "    cp $(APP_NAME) \$$(brew --prefix)/bin/" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64/install.sh
	@echo "else" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64/install.sh
	@echo "    sudo cp $(APP_NAME) /usr/local/bin/" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64/install.sh
	@echo "fi" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64/install.sh
	@echo "echo \"Installation completed!\"" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64/install.sh

	chmod +x $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-amd64/install.sh

	# Create tar.gz
	cd $(DIST_DIR) && tar -czf $(APP_NAME)-$(VERSION)-darwin-amd64.tar.gz $(APP_NAME)-$(VERSION)-darwin-amd64/

# macOS distribution for ARM64
.PHONY: dist-darwin-arm64
dist-darwin-arm64:
	@echo "📦 Creating macOS ARM64 package..."
	mkdir -p $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-arm64

	cp $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-arm64/$(APP_NAME)
	chmod +x $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-arm64/$(APP_NAME)

	# Create install script
	@echo "#!/bin/bash" > $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-arm64/install.sh
	@echo "echo \"Installing ECR Deploy Tool...\"" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-arm64/install.sh
	@echo "if command -v brew &> /dev/null; then" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-arm64/install.sh
	@echo "    cp $(APP_NAME) \$$(brew --prefix)/bin/" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-arm64/install.sh
	@echo "else" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-arm64/install.sh
	@echo "    sudo cp $(APP_NAME) /usr/local/bin/" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-arm64/install.sh
	@echo "fi" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-arm64/install.sh
	@echo "echo \"Installation completed!\"" >> $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-arm64/install.sh

	chmod +x $(DIST_DIR)/$(APP_NAME)-$(VERSION)-darwin-arm64/install.sh

	# Create tar.gz
	cd $(DIST_DIR) && tar -czf $(APP_NAME)-$(VERSION)-darwin-arm64.tar.gz $(APP_NAME)-$(VERSION)-darwin-arm64/

# Install locally (for development)
.PHONY: install
install: build
	@echo "📦 Installing locally..."
	sudo cp $(APP_NAME) /usr/local/bin/
	@echo "✅ Installed to /usr/local/bin/$(APP_NAME)"

# Development helpers
.PHONY: dev
dev:
	@echo "🔄 Running in development mode..."
	$(GOCMD) run main.go

.PHONY: fmt
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...

.PHONY: vet
vet:
	@echo "🔍 Vetting code..."
	go vet ./...

.PHONY: lint
lint: fmt vet

# Help
.PHONY: help
help:
	@echo "ECR Deploy Tool - Makefile Commands"
	@echo "=================================="
	@echo ""
	@echo "Development:"
	@echo "  make deps      - Install dependencies"
	@echo "  make build     - Build for current platform"
	@echo "  make run       - Run the application"
	@echo "  make dev       - Run in development mode"
	@echo "  make test      - Run tests"
	@echo "  make lint      - Format and vet code"
	@echo ""
	@echo "Building:"
	@echo "  make build-all - Build for all platforms"
	@echo "  make dist      - Create distribution packages"
	@echo "  make clean     - Clean build artifacts"
	@echo ""
	@echo "Installation:"
	@echo "  make install   - Install locally"
	@echo ""
	@echo "Examples:"
	@echo "  make           - Clean, install deps, and build"
	@echo "  make dist      - Create all distribution packages"


release-binaries: build ## Prepare raw binaries only for GitHub release
	@echo "$(BLUE)Copying raw binaries to $(DIST_DIR)...$(RESET)"
	@mkdir -p $(DIST_DIR)
	@for file in $(BUILD_DIR)/*; do \
		if [ -f "$$file" ]; then \
			cp "$$file" $(DIST_DIR)/; \
		fi; \
	done
	@echo "$(GREEN)Raw binaries copied to $(DIST_DIR)/$(RESET)"
	@ls -la $(DIST_DIR)/
