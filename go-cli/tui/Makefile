BINARY_NAME := gocode
ENGINE_NAME := gocode-engine
VERSION     := 0.1.0
BUILD_DIR   := release
GO_SRC      := ../cmd/go-cli

PLATFORMS := darwin-arm64 darwin-amd64 linux-amd64 linux-arm64

.PHONY: all clean release build-engine build-tui $(PLATFORMS)

all: release-local

# Build for current platform only (fast, for development)
release-local: clean build-tui-standalone
	@mkdir -p $(BUILD_DIR)
	GOOS=$$(go env GOOS) GOARCH=$$(go env GOARCH) go build -o $(BUILD_DIR)/$(ENGINE_NAME) $(GO_SRC)
	cp standalone-bin $(BUILD_DIR)/$(BINARY_NAME)
	@echo ""
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME) + $(BUILD_DIR)/$(ENGINE_NAME)"
	@echo "Copy both to a directory in your PATH to install."

# Build standalone TUI binary via Bun
build-tui-standalone:
	npx tsc
	bun build --compile --outfile=standalone-bin \
		dist/standalone.js

# Cross-compile release archives for all platforms
release: clean build-tui-standalone
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d- -f1); \
		arch=$$(echo $$platform | cut -d- -f2); \
		dir=$(BUILD_DIR)/$(BINARY_NAME)-$$platform; \
		echo "Building $$platform..."; \
		mkdir -p $$dir; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -o $$dir/$(ENGINE_NAME) $(GO_SRC); \
		bun build --compile --outfile=$$dir/$(BINARY_NAME) \
			--target=bun-$$os-$$arch \
			dist/standalone.js; \
		tar -czf $(BUILD_DIR)/$(BINARY_NAME)-$$platform.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-$$platform; \
		echo "  -> $(BUILD_DIR)/$(BINARY_NAME)-$$platform.tar.gz"; \
	done
	@echo ""
	@echo "Release archives in $(BUILD_DIR)/"

clean:
	rm -rf $(BUILD_DIR) standalone-bin

# Install locally (copies to /usr/local/bin)
install: release-local
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	cp $(BUILD_DIR)/$(ENGINE_NAME) /usr/local/bin/$(ENGINE_NAME)
	@echo "Installed to /usr/local/bin/"
