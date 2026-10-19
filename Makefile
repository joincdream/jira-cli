.PHONY: build install clean test test-unit complexity complexity-top complexity-cyclo complexity-cognit

BINARY_NAME=jira
BIN_DIR=../../bin

# Complexity thresholds (fail if exceeded)
CYCLO_MAX ?= 40
COGNIT_MAX ?= 70

build:
	@mkdir -p $(BIN_DIR)
	@echo "🔨 Building Jira CLI..."
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd
	@echo "✅ Build complete: $(BIN_DIR)/$(BINARY_NAME)"

install: build
	@echo "📦 Installing to ~/.local/bin/$(BINARY_NAME)..."
	install -m 755 $(BIN_DIR)/$(BINARY_NAME) $(HOME)/.local/bin/$(BINARY_NAME)
	@echo "✅ Installed: $(HOME)/.local/bin/$(BINARY_NAME)"

clean:
	rm -rf $(BIN_DIR)/$(BINARY_NAME)

test: test-unit complexity

test-unit:
	@echo "🧪 Running unit tests..."
	go test -v ./...

complexity: complexity-cyclo complexity-cognit
	@echo "✅ All complexity checks passed!"

complexity-cyclo:
	@echo "🔍 Checking cyclomatic complexity (gocyclo <= $(CYCLO_MAX))..."
	@go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -ignore "_test\.go" -over $(CYCLO_MAX) .

complexity-cognit:
	@echo "🔍 Checking cognitive complexity (gocognit <= $(COGNIT_MAX))..."
	@go run github.com/uudashr/gocognit/cmd/gocognit@latest -ignore "_test\.go" -over $(COGNIT_MAX) .

complexity-top:
	@echo "📊 Top 10 Cyclomatic Complexity Functions (gocyclo):"
	@go run github.com/fzipp/gocyclo/cmd/gocyclo@latest -ignore "_test\.go" -top 10 .
	@echo ""
	@echo "📊 Top 10 Cognitive Complexity Functions (gocognit):"
	@go run github.com/uudashr/gocognit/cmd/gocognit@latest -ignore "_test\.go" -top 10 .
