.PHONY: build install clean test

BINARY_NAME=jira
BIN_DIR=../../bin

build:
	@mkdir -p $(BIN_DIR)
	@echo "🔨 Building Jira CLI..."
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd
	@echo "✅ Build complete: $(BIN_DIR)/$(BINARY_NAME)"

install: build
	@echo "📦 Installing to ~/.local/bin/$(BINARY_NAME)..."
	@mkdir -p $(HOME)/.local/bin
	cp $(BIN_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)
	@echo "✅ Installed: $(HOME)/.local/bin/$(BINARY_NAME)"

clean:
	rm -rf $(BIN_DIR)/$(BINARY_NAME)

test:
	go test -v ./...
